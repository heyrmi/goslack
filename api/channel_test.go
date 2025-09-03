package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	mockdb "github.com/heyrmi/goslack/db/mock"
	db "github.com/heyrmi/goslack/db/sqlc"
	"github.com/heyrmi/goslack/service"
	"github.com/heyrmi/goslack/token"
	"github.com/heyrmi/goslack/util"
	"github.com/stretchr/testify/require"
)

func TestCreateChannelAPI(t *testing.T) {
	user, _ := randomUser(t)
	workspace := randomWorkspace(user.OrganizationID)
	channel := randomChannel(workspace.ID, user.ID)

	// Make user a member of the workspace
	user.WorkspaceID = sql.NullInt64{Int64: workspace.ID, Valid: true}
	user.Role = "member"

	testCases := []struct {
		name          string
		workspaceID   int64
		body          gin.H
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:        "OK",
			workspaceID: workspace.ID,
			body: gin.H{
				"name":       channel.Name,
				"is_private": channel.IsPrivate,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Email, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).
					Times(1).
					Return(user, nil)

				// Check workspace membership
				roleArg := db.CheckUserWorkspaceRoleParams{
					ID:          user.ID,
					WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
				}
				store.EXPECT().
					CheckUserWorkspaceRole(gomock.Any(), gomock.Eq(roleArg)).
					Times(1).
					Return(user.Role, nil)

				arg := db.CreateChannelParams{
					WorkspaceID: workspace.ID,
					Name:        channel.Name,
					IsPrivate:   channel.IsPrivate,
					CreatedBy:   user.ID,
				}
				store.EXPECT().
					CreateChannel(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(channel, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchChannel(t, recorder.Body, channel)
			},
		},
		{
			name:        "NoAuthorization",
			workspaceID: workspace.ID,
			body: gin.H{
				"name":       channel.Name,
				"is_private": channel.IsPrivate,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:        "NotWorkspaceMember",
			workspaceID: workspace.ID,
			body: gin.H{
				"name":       channel.Name,
				"is_private": channel.IsPrivate,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Email, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).
					Times(1).
					Return(user, nil)

				// User is not a member of this workspace
				roleArg := db.CheckUserWorkspaceRoleParams{
					ID:          user.ID,
					WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
				}
				store.EXPECT().
					CheckUserWorkspaceRole(gomock.Any(), gomock.Eq(roleArg)).
					Times(1).
					Return("", sql.ErrNoRows)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name:        "InternalError",
			workspaceID: workspace.ID,
			body: gin.H{
				"name":       channel.Name,
				"is_private": channel.IsPrivate,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Email, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).
					Times(1).
					Return(user, nil)

				roleArg := db.CheckUserWorkspaceRoleParams{
					ID:          user.ID,
					WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
				}
				store.EXPECT().
					CheckUserWorkspaceRole(gomock.Any(), gomock.Eq(roleArg)).
					Times(1).
					Return(user.Role, nil)

				store.EXPECT().
					CreateChannel(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Channel{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:        "InvalidJSON",
			workspaceID: workspace.ID,
			body: gin.H{
				"name": "",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Email, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).
					Times(1).
					Return(user, nil)

				// Check workspace membership (middleware runs before JSON validation)
				roleArg := db.CheckUserWorkspaceRoleParams{
					ID:          user.ID,
					WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
				}
				store.EXPECT().
					CheckUserWorkspaceRole(gomock.Any(), gomock.Eq(roleArg)).
					Times(1).
					Return(user.Role, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("/workspaces/%d/channels", tc.workspaceID)
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestListChannelsAPI(t *testing.T) {
	user, _ := randomUser(t)
	workspace := randomWorkspace(user.OrganizationID)

	// Make user a member of the workspace
	user.WorkspaceID = sql.NullInt64{Int64: workspace.ID, Valid: true}
	user.Role = "member"

	n := 5
	channels := make([]db.Channel, n)
	for i := 0; i < n; i++ {
		channels[i] = randomChannel(workspace.ID, user.ID)
	}

	type Query struct {
		pageID   int
		pageSize int
	}

	testCases := []struct {
		name          string
		workspaceID   int64
		query         Query
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:        "OK",
			workspaceID: workspace.ID,
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Email, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).
					Times(1).
					Return(user, nil)

				// Check workspace membership
				roleArg := db.CheckUserWorkspaceRoleParams{
					ID:          user.ID,
					WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
				}
				store.EXPECT().
					CheckUserWorkspaceRole(gomock.Any(), gomock.Eq(roleArg)).
					Times(1).
					Return(user.Role, nil)

				arg := db.ListChannelsByWorkspaceParams{
					WorkspaceID: workspace.ID,
					Limit:       int32(n),
					Offset:      0,
				}

				store.EXPECT().
					ListChannelsByWorkspace(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(channels, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchChannels(t, recorder.Body, channels)
			},
		},
		{
			name:        "NoAuthorization",
			workspaceID: workspace.ID,
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:        "NotWorkspaceMember",
			workspaceID: workspace.ID,
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Email, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).
					Times(1).
					Return(user, nil)

				// User is not a member, middleware should block them
				roleArg := db.CheckUserWorkspaceRoleParams{
					ID:          user.ID,
					WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
				}
				store.EXPECT().
					CheckUserWorkspaceRole(gomock.Any(), gomock.Eq(roleArg)).
					Times(1).
					Return("", sql.ErrNoRows)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/workspaces/%d/channels", tc.workspaceID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			// Add query parameters
			q := request.URL.Query()
			q.Add("page_id", fmt.Sprintf("%d", tc.query.pageID))
			q.Add("page_size", fmt.Sprintf("%d", tc.query.pageSize))
			request.URL.RawQuery = q.Encode()

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func randomChannel(workspaceID, createdBy int64) db.Channel {
	return db.Channel{
		ID:          util.RandomInt(1, 1000),
		WorkspaceID: workspaceID,
		Name:        util.RandomString(10),
		IsPrivate:   util.RandomBool(),
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
	}
}

func requireBodyMatchChannel(t *testing.T, body *bytes.Buffer, channel db.Channel) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotChannel service.ChannelResponse
	err = json.Unmarshal(data, &gotChannel)
	require.NoError(t, err)

	require.Equal(t, channel.ID, gotChannel.ID)
	require.Equal(t, channel.WorkspaceID, gotChannel.WorkspaceID)
	require.Equal(t, channel.Name, gotChannel.Name)
	require.Equal(t, channel.IsPrivate, gotChannel.IsPrivate)
	require.Equal(t, channel.CreatedBy, gotChannel.CreatedBy)
	require.WithinDuration(t, channel.CreatedAt, gotChannel.CreatedAt, time.Second)
}

func requireBodyMatchChannels(t *testing.T, body *bytes.Buffer, channels []db.Channel) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotChannels []service.ChannelResponse
	err = json.Unmarshal(data, &gotChannels)
	require.NoError(t, err)

	require.Equal(t, len(channels), len(gotChannels))
	for i, channel := range channels {
		require.Equal(t, channel.ID, gotChannels[i].ID)
		require.Equal(t, channel.WorkspaceID, gotChannels[i].WorkspaceID)
		require.Equal(t, channel.Name, gotChannels[i].Name)
		require.Equal(t, channel.IsPrivate, gotChannels[i].IsPrivate)
		require.Equal(t, channel.CreatedBy, gotChannels[i].CreatedBy)
		require.WithinDuration(t, channel.CreatedAt, gotChannels[i].CreatedAt, time.Second)
	}
}
