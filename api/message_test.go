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

func TestSendChannelMessageAPI(t *testing.T) {
	user, _ := randomUser(t)
	workspace := randomWorkspace(user.OrganizationID)
	channel := randomChannel(workspace.ID, user.ID)

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
				"content": "Hello, world!",
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
					Times(2).
					Return(user.Role, nil)

				arg := db.CreateChannelMessageParams{
					WorkspaceID: workspace.ID,
					ChannelID:   sql.NullInt64{Int64: channel.ID, Valid: true},
					SenderID:    user.ID,
					Content:     "Hello, world!",
					ContentType: "text",
				}

				message := db.Message{
					ID:          1,
					WorkspaceID: workspace.ID,
					ChannelID:   sql.NullInt64{Int64: channel.ID, Valid: true},
					SenderID:    user.ID,
					Content:     "Hello, world!",
					MessageType: "channel",
					CreatedAt:   time.Now(),
				}

				store.EXPECT().
					CreateChannelMessage(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(message, nil)

				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.ID)).
					Times(1).
					Return(user, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
			},
		},
		{
			name: "InvalidContent",
			body: gin.H{
				"content": "", // Empty content
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Email, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).
					Times(1).
					Return(user, nil)

				// Check workspace membership for middleware
				roleArg := db.CheckUserWorkspaceRoleParams{
					ID:          user.ID,
					WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
				}
				store.EXPECT().
					CheckUserWorkspaceRole(gomock.Any(), gomock.Eq(roleArg)).
					Times(1). // Only middleware check since validation fails
					Return(user.Role, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "NoAuthorization",
			body: gin.H{
				"content": "Hello, world!",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				// No authorization
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

			url := fmt.Sprintf("/workspace/%d/channels/%d/messages", workspace.ID, channel.ID)
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestSendDirectMessageAPI(t *testing.T) {
	user, _ := randomUser(t)
	receiver, _ := randomUser(t)
	workspace := randomWorkspace(user.OrganizationID)

	// Make users members of the same workspace
	user.WorkspaceID = sql.NullInt64{Int64: workspace.ID, Valid: true}
	user.Role = "member"
	receiver.WorkspaceID = user.WorkspaceID
	receiver.Role = "member"

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
				"receiver_id": receiver.ID,
				"content":     "Hello there!",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Email, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).
					Times(1).
					Return(user, nil)

				// Check workspace membership for middleware
				roleArg := db.CheckUserWorkspaceRoleParams{
					ID:          user.ID,
					WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
				}
				store.EXPECT().
					CheckUserWorkspaceRole(gomock.Any(), gomock.Eq(roleArg)).
					Times(2). // Middleware + sender check
					Return(user.Role, nil)

				// Check receiver workspace membership
				receiverRoleArg := db.CheckUserWorkspaceRoleParams{
					ID:          receiver.ID,
					WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
				}
				store.EXPECT().
					CheckUserWorkspaceRole(gomock.Any(), gomock.Eq(receiverRoleArg)).
					Times(1). // Receiver check
					Return(receiver.Role, nil)

				arg := db.CreateDirectMessageParams{
					WorkspaceID: workspace.ID,
					SenderID:    user.ID,
					ReceiverID:  sql.NullInt64{Int64: receiver.ID, Valid: true},
					Content:     "Hello there!",
					ContentType: "text",
				}

				message := db.Message{
					ID:          1,
					WorkspaceID: workspace.ID,
					SenderID:    user.ID,
					ReceiverID:  sql.NullInt64{Int64: receiver.ID, Valid: true},
					Content:     "Hello there!",
					MessageType: "direct",
					CreatedAt:   time.Now(),
				}

				store.EXPECT().
					CreateDirectMessage(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(message, nil)

				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.ID)).
					Times(1).
					Return(user, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
			},
		},
		{
			name: "InvalidReceiverID",
			body: gin.H{
				"receiver_id": 0,
				"content":     "Hello there!",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Email, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).
					Times(1).
					Return(user, nil)

				// Check workspace membership for middleware
				roleArg := db.CheckUserWorkspaceRoleParams{
					ID:          user.ID,
					WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
				}
				store.EXPECT().
					CheckUserWorkspaceRole(gomock.Any(), gomock.Eq(roleArg)).
					Times(1). // Only middleware check since validation fails
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

			url := fmt.Sprintf("/workspace/%d/messages/direct", workspace.ID)
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestGetChannelMessagesAPI(t *testing.T) {
	user, _ := randomUser(t)
	workspace := randomWorkspace(user.OrganizationID)
	channel := randomChannel(workspace.ID, user.ID)

	// Make user a member of the workspace
	user.WorkspaceID = sql.NullInt64{Int64: workspace.ID, Valid: true}
	user.Role = "member"

	testCases := []struct {
		name          string
		query         string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:  "OK",
			query: "?limit=10&offset=0",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Email, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).
					Times(1).
					Return(user, nil)

				// Check workspace membership for middleware and service
				roleArg := db.CheckUserWorkspaceRoleParams{
					ID:          user.ID,
					WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
				}
				store.EXPECT().
					CheckUserWorkspaceRole(gomock.Any(), gomock.Eq(roleArg)).
					Times(2). // Middleware + service check
					Return(user.Role, nil)

				messages := []db.GetChannelMessagesRow{
					{
						ID:              1,
						WorkspaceID:     workspace.ID,
						ChannelID:       sql.NullInt64{Int64: channel.ID, Valid: true},
						SenderID:        user.ID,
						Content:         "Hello, world!",
						MessageType:     "channel",
						CreatedAt:       time.Now(),
						SenderFirstName: user.FirstName,
						SenderLastName:  user.LastName,
						SenderEmail:     user.Email,
					},
				}

				store.EXPECT().
					GetChannelMessages(gomock.Any(), gomock.Any()).
					Times(1).
					Return(messages, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:  "InvalidLimit",
			query: "?limit=-1&offset=0", // Invalid limit
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Email, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).
					Times(1).
					Return(user, nil)

				// Check workspace membership for middleware
				roleArg := db.CheckUserWorkspaceRoleParams{
					ID:          user.ID,
					WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
				}
				store.EXPECT().
					CheckUserWorkspaceRole(gomock.Any(), gomock.Eq(roleArg)).
					Times(1). // Only middleware check since validation fails
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

			url := fmt.Sprintf("/workspace/%d/channels/%d/messages%s", workspace.ID, channel.ID, tc.query)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestEditMessageAPI(t *testing.T) {
	user, _ := randomUser(t)
	workspace := randomWorkspace(user.OrganizationID)
	channel := randomChannel(workspace.ID, user.ID)
	message := randomMessage(workspace.ID, channel.ID, user.ID)

	// Make user a member of the workspace
	user.WorkspaceID = sql.NullInt64{Int64: workspace.ID, Valid: true}
	user.Role = "member"

	testCases := []struct {
		name          string
		messageID     int64
		body          gin.H
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "OK",
			messageID: message.ID,
			body: gin.H{
				"content": "Updated content",
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
					CheckMessageAuthor(gomock.Any(), gomock.Eq(message.ID)).
					Times(1).
					Return(user.ID, nil)

				updatedMessage := message
				updatedMessage.Content = "Updated content"
				editedAt := time.Now()
				updatedMessage.EditedAt = sql.NullTime{Time: editedAt, Valid: true}

				store.EXPECT().
					UpdateMessageContent(gomock.Any(), gomock.Any()).
					Times(1).
					Return(updatedMessage, nil)

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
			name:      "NotAuthor",
			messageID: message.ID,
			body: gin.H{
				"content": "Updated content",
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
					CheckMessageAuthor(gomock.Any(), gomock.Eq(message.ID)).
					Times(1).
					Return(util.RandomInt(1000, 2000), nil) // Different author
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

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("/messages/%d", tc.messageID)
			request, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

// Helper functions for testing
func randomMessage(workspaceID, channelID, senderID int64) db.Message {
	return db.Message{
		ID:          util.RandomInt(1, 1000),
		WorkspaceID: workspaceID,
		ChannelID:   sql.NullInt64{Int64: channelID, Valid: true},
		SenderID:    senderID,
		Content:     util.RandomString(50),
		MessageType: "channel",
		CreatedAt:   time.Now(),
	}
}

func randomDirectMessage(workspaceID, senderID, receiverID int64) db.Message {
	return db.Message{
		ID:          util.RandomInt(1, 1000),
		WorkspaceID: workspaceID,
		SenderID:    senderID,
		ReceiverID:  sql.NullInt64{Int64: receiverID, Valid: true},
		Content:     util.RandomString(50),
		MessageType: "direct",
		CreatedAt:   time.Now(),
	}
}
