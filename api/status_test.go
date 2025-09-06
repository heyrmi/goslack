package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	mockdb "github.com/heyrmi/goslack/db/mock"
	db "github.com/heyrmi/goslack/db/sqlc"
	"github.com/heyrmi/goslack/token"
	"github.com/heyrmi/goslack/util"
	"github.com/stretchr/testify/require"
)

func TestUpdateUserStatusAPI(t *testing.T) {
	user, _ := randomUser(t)
	workspace := randomWorkspace(user.OrganizationID)

	// Make user a member of the workspace
	user.WorkspaceID = sql.NullInt64{Int64: workspace.ID, Valid: true}
	user.Role = "member"

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
				"status":        "online",
				"custom_status": "Working on GoSlack",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Email, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).
					Times(1).
					Return(user, nil)

				arg := db.CheckUserWorkspaceRoleParams{
					ID:          user.ID,
					WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
				}
				store.EXPECT().
					CheckUserWorkspaceRole(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(user.Role, nil)

				userStatus := db.UserStatus{
					UserID:       user.ID,
					WorkspaceID:  workspace.ID,
					Status:       "online",
					CustomStatus: sql.NullString{String: "Working on GoSlack", Valid: true},
					LastSeenAt:   time.Now(),
					UpdatedAt:    time.Now(),
				}

				store.EXPECT().
					UpsertUserStatus(gomock.Any(), gomock.Any()).
					Times(1).
					Return(userStatus, nil)

				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.ID)).
					Times(1).
					Return(user, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "InvalidStatus",
			body: gin.H{
				"status": "invalid",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Email, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).
					Times(1).
					Return(user, nil)

				arg := db.CheckUserWorkspaceRoleParams{
					ID:          user.ID,
					WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
				}
				store.EXPECT().
					CheckUserWorkspaceRole(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(user.Role, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "NoAuthorization",
			body: gin.H{
				"status": "online",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				// No authorization header
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

			url := fmt.Sprintf("/workspace/%d/status", workspace.ID)
			request, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestGetUserStatusAPI(t *testing.T) {
	user, _ := randomUser(t)
	workspace := randomWorkspace(user.OrganizationID)

	// Make user a member of the workspace
	user.WorkspaceID = sql.NullInt64{Int64: workspace.ID, Valid: true}
	user.Role = "member"

	testCases := []struct {
		name          string
		userID        int64
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "OK",
			userID: user.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Email, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).
					Times(1).
					Return(user, nil)

				store.EXPECT().
					CheckUserWorkspaceRole(gomock.Any(), gomock.Any()).
					Times(1).
					Return(user.Role, nil)

				userStatus := db.UserStatus{
					UserID:      user.ID,
					WorkspaceID: workspace.ID,
					Status:      "online",
					LastSeenAt:  time.Now(),
					UpdatedAt:   time.Now(),
				}

				store.EXPECT().
					GetUserStatus(gomock.Any(), gomock.Any()).
					Times(1).
					Return(userStatus, nil)

				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.ID)).
					Times(1).
					Return(user, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:   "StatusNotFound",
			userID: user.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Email, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).
					Times(1).
					Return(user, nil)

				store.EXPECT().
					CheckUserWorkspaceRole(gomock.Any(), gomock.Any()).
					Times(1).
					Return(user.Role, nil)

				store.EXPECT().
					GetUserStatus(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.UserStatus{}, sql.ErrNoRows)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
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

			url := fmt.Sprintf("/workspace/%d/status/%d", workspace.ID, tc.userID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestUpdateUserActivityAPI(t *testing.T) {
	user, _ := randomUser(t)
	workspace := randomWorkspace(user.OrganizationID)

	// Make user a member of the workspace
	user.WorkspaceID = sql.NullInt64{Int64: workspace.ID, Valid: true}
	user.Role = "member"

	testCases := []struct {
		name          string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Email, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).
					Times(1).
					Return(user, nil)

				store.EXPECT().
					CheckUserWorkspaceRole(gomock.Any(), gomock.Any()).
					Times(1).
					Return(user.Role, nil)

				store.EXPECT().
					UpdateLastActivity(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
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

			url := fmt.Sprintf("/workspace/%d/activity", workspace.ID)
			request, err := http.NewRequest(http.MethodPost, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

// Helper functions for testing
func randomUserStatus(userID, workspaceID int64) db.UserStatus {
	return db.UserStatus{
		UserID:       userID,
		WorkspaceID:  workspaceID,
		Status:       "online",
		CustomStatus: sql.NullString{String: util.RandomString(20), Valid: true},
		LastSeenAt:   time.Now(),
		UpdatedAt:    time.Now(),
	}
}
