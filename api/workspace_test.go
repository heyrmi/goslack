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

func TestCreateWorkspaceAPI(t *testing.T) {
	user, _ := randomUser(t)
	workspace := randomWorkspace(user.OrganizationID)

	testCases := []struct {
		name          string
		body          gin.H
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"name": workspace.Name,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Email, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).
					Times(1).
					Return(user, nil)

				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.ID)).
					Times(1).
					Return(user, nil)

				arg := db.CreateWorkspaceParams{
					OrganizationID: user.OrganizationID,
					Name:           workspace.Name,
				}
				store.EXPECT().
					CreateWorkspace(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(workspace, nil)

				updateArg := db.UpdateUserWorkspaceParams{
					ID:          user.ID,
					WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
					Role:        "admin",
				}
				updatedUser := user
				updatedUser.WorkspaceID = sql.NullInt64{Int64: workspace.ID, Valid: true}
				updatedUser.Role = "admin"

				store.EXPECT().
					UpdateUserWorkspace(gomock.Any(), gomock.Eq(updateArg)).
					Times(1).
					Return(updatedUser, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchWorkspace(t, recorder.Body, workspace)
			},
		},
		{
			name: "NoAuthorization",
			body: gin.H{
				"name": workspace.Name,
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
			name: "InternalError",
			body: gin.H{
				"name": workspace.Name,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Email, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).
					Times(1).
					Return(user, nil)

				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.ID)).
					Times(1).
					Return(user, nil)

				store.EXPECT().
					CreateWorkspace(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Workspace{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidJSON",
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

			url := "/workspaces"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestListWorkspacesAPI(t *testing.T) {
	user, _ := randomUser(t)

	n := 5
	workspaces := make([]db.Workspace, n)
	for i := 0; i < n; i++ {
		workspaces[i] = randomWorkspace(user.OrganizationID)
	}

	type Query struct {
		pageID   int
		pageSize int
	}

	testCases := []struct {
		name          string
		query         Query
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
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

				arg := db.ListWorkspacesByOrganizationParams{
					OrganizationID: user.OrganizationID,
					Limit:          int32(n),
					Offset:         0,
				}

				store.EXPECT().
					ListWorkspacesByOrganization(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(workspaces, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchWorkspaces(t, recorder.Body, workspaces)
			},
		},
		{
			name: "NoAuthorization",
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
			name: "InternalError",
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

				store.EXPECT().
					ListWorkspacesByOrganization(gomock.Any(), gomock.Any()).
					Times(1).
					Return([]db.Workspace{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidPageID",
			query: Query{
				pageID:   -1,
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

			url := "/workspaces"
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

func randomWorkspace(organizationID int64) db.Workspace {
	return db.Workspace{
		ID:             util.RandomInt(1, 1000),
		OrganizationID: organizationID,
		Name:           util.RandomString(10),
		CreatedAt:      time.Now(),
	}
}

func requireBodyMatchWorkspace(t *testing.T, body *bytes.Buffer, workspace db.Workspace) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotWorkspace service.WorkspaceResponse
	err = json.Unmarshal(data, &gotWorkspace)
	require.NoError(t, err)

	require.Equal(t, workspace.ID, gotWorkspace.ID)
	require.Equal(t, workspace.OrganizationID, gotWorkspace.OrganizationID)
	require.Equal(t, workspace.Name, gotWorkspace.Name)
	require.WithinDuration(t, workspace.CreatedAt, gotWorkspace.CreatedAt, time.Second)
}

func requireBodyMatchWorkspaces(t *testing.T, body *bytes.Buffer, workspaces []db.Workspace) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotWorkspaces []service.WorkspaceResponse
	err = json.Unmarshal(data, &gotWorkspaces)
	require.NoError(t, err)

	require.Equal(t, len(workspaces), len(gotWorkspaces))
	for i, workspace := range workspaces {
		require.Equal(t, workspace.ID, gotWorkspaces[i].ID)
		require.Equal(t, workspace.OrganizationID, gotWorkspaces[i].OrganizationID)
		require.Equal(t, workspace.Name, gotWorkspaces[i].Name)
		require.WithinDuration(t, workspace.CreatedAt, gotWorkspaces[i].CreatedAt, time.Second)
	}
}
