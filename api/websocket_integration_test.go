package api

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/websocket"
	mockdb "github.com/heyrmi/goslack/db/mock"
	db "github.com/heyrmi/goslack/db/sqlc"
	"github.com/heyrmi/goslack/service"
	"github.com/heyrmi/goslack/util"
	"github.com/stretchr/testify/require"
)

func TestWebSocketMessageFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	config := util.Config{
		TokenSymmetricKey:       util.RandomString(32),
		AccessTokenDuration:     time.Minute,
		WSMaxConnectionsPerUser: 5,
		WSPingInterval:          54 * time.Second,
		WSPongTimeout:           60 * time.Second,
		WSReadBufferSize:        1024,
		WSWriteBufferSize:       1024,
	}

	server, err := NewServer(config, store)
	require.NoError(t, err)

	// Create test data
	workspace := randomWSWorkspace()
	channel := randomWSChannel()
	sender := randomWSUser()
	receiver := randomWSUser()

	// Mock database calls for WebSocket connections
	store.EXPECT().
		GetUserByEmail(gomock.Any(), gomock.Eq(sender.Email)).
		Times(2). // Called for both WebSocket connection and message sending
		Return(db.User{
			ID:             sender.ID,
			OrganizationID: 1,
			Email:          sender.Email,
			FirstName:      sender.FirstName,
			LastName:       sender.LastName,
			WorkspaceID:    sql.NullInt64{Int64: workspace.ID, Valid: true},
			Role:           "member",
		}, nil)

	store.EXPECT().
		GetUserByEmail(gomock.Any(), gomock.Eq(receiver.Email)).
		Times(1).
		Return(db.User{
			ID:             receiver.ID,
			OrganizationID: 1,
			Email:          receiver.Email,
			FirstName:      receiver.FirstName,
			LastName:       receiver.LastName,
			WorkspaceID:    sql.NullInt64{Int64: workspace.ID, Valid: true},
			Role:           "member",
		}, nil)

	// Mock workspace member check for message sending (called multiple times)
	store.EXPECT().
		CheckUserWorkspaceRole(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return("member", nil)

	// Mock message sending
	mockMessage := db.Message{
		ID:          util.RandomInt(1, 1000),
		WorkspaceID: workspace.ID,
		ChannelID:   sql.NullInt64{Int64: channel.ID, Valid: true},
		SenderID:    sender.ID,
		Content:     "Hello, WebSocket!",
		MessageType: "channel",
		CreatedAt:   time.Now(),
	}

	store.EXPECT().
		CreateChannelMessage(gomock.Any(), gomock.Any()).
		Times(1).
		Return(mockMessage, nil)

	// Mock GetUser for message response (to get sender info)
	store.EXPECT().
		GetUser(gomock.Any(), gomock.Eq(sender.ID)).
		Times(1).
		Return(db.User{
			ID:             sender.ID,
			OrganizationID: 1,
			Email:          sender.Email,
			FirstName:      sender.FirstName,
			LastName:       sender.LastName,
			WorkspaceID:    sql.NullInt64{Int64: workspace.ID, Valid: true},
			Role:           "member",
			CreatedAt:      time.Now(),
		}, nil)

	// Create access tokens
	senderToken, _, err := server.tokenMaker.CreateToken(sender.Email, time.Minute)
	require.NoError(t, err)

	receiverToken, _, err := server.tokenMaker.CreateToken(receiver.Email, time.Minute)
	require.NoError(t, err)

	// Start the hub
	go server.hub.Run()

	// Start server
	testServer := httptest.NewServer(server.router)
	defer testServer.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"

	// Create WebSocket connections for both users
	senderHeader := http.Header{}
	senderHeader.Set("Authorization", "Bearer "+senderToken)

	receiverHeader := http.Header{}
	receiverHeader.Set("Authorization", "Bearer "+receiverToken)

	senderConn, _, err := websocket.DefaultDialer.Dial(wsURL, senderHeader)
	require.NoError(t, err)
	defer senderConn.Close()

	receiverConn, _, err := websocket.DefaultDialer.Dial(wsURL, receiverHeader)
	require.NoError(t, err)
	defer receiverConn.Close()

	// Read connection established messages
	var senderEstablished service.WSMessage
	err = senderConn.ReadJSON(&senderEstablished)
	require.NoError(t, err)
	require.Equal(t, WSConnectionEstablished, senderEstablished.Type)

	var receiverEstablished service.WSMessage
	err = receiverConn.ReadJSON(&receiverEstablished)
	require.NoError(t, err)
	require.Equal(t, WSConnectionEstablished, receiverEstablished.Type)

	// Send a message via REST API
	messageBody := `{"content": "Hello, WebSocket!"}`
	messageURL := "/workspace/" + util.IntToString(workspace.ID) + "/channels/" + util.IntToString(channel.ID) + "/messages"

	req, err := http.NewRequest(http.MethodPost, messageURL, strings.NewReader(messageBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+senderToken)

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusCreated, recorder.Code)

	// Both users should receive the message via WebSocket
	receiverConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var receiverMessage service.WSMessage
	err = receiverConn.ReadJSON(&receiverMessage)
	require.NoError(t, err)
	require.Equal(t, WSMessageSent, receiverMessage.Type)
	require.Equal(t, workspace.ID, receiverMessage.WorkspaceID)
	require.Equal(t, channel.ID, *receiverMessage.ChannelID)

	senderConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var senderMessage service.WSMessage
	err = senderConn.ReadJSON(&senderMessage)
	require.NoError(t, err)
	require.Equal(t, WSMessageSent, senderMessage.Type)
	require.Equal(t, workspace.ID, senderMessage.WorkspaceID)
	require.Equal(t, channel.ID, *senderMessage.ChannelID)
}

func TestWebSocketStatusUpdates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	config := util.Config{
		TokenSymmetricKey:       util.RandomString(32),
		AccessTokenDuration:     time.Minute,
		WSMaxConnectionsPerUser: 5,
		WSPingInterval:          54 * time.Second,
		WSPongTimeout:           60 * time.Second,
		WSReadBufferSize:        1024,
		WSWriteBufferSize:       1024,
	}

	server, err := NewServer(config, store)
	require.NoError(t, err)

	// Create test data
	workspace := randomWSWorkspace()
	user1 := randomWSUser()
	user2 := randomWSUser()

	// Mock database calls for WebSocket connections
	store.EXPECT().
		GetUserByEmail(gomock.Any(), gomock.Eq(user1.Email)).
		Times(2). // Called for WebSocket connection and status update
		Return(db.User{
			ID:             user1.ID,
			OrganizationID: 1,
			Email:          user1.Email,
			FirstName:      user1.FirstName,
			LastName:       user1.LastName,
			WorkspaceID:    sql.NullInt64{Int64: workspace.ID, Valid: true},
			Role:           "member",
		}, nil)

	store.EXPECT().
		GetUserByEmail(gomock.Any(), gomock.Eq(user2.Email)).
		Times(1).
		Return(db.User{
			ID:             user2.ID,
			OrganizationID: 1,
			Email:          user2.Email,
			FirstName:      user2.FirstName,
			LastName:       user2.LastName,
			WorkspaceID:    sql.NullInt64{Int64: workspace.ID, Valid: true},
			Role:           "member",
		}, nil)

	// Mock workspace member check for status update (called multiple times)
	store.EXPECT().
		CheckUserWorkspaceRole(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return("member", nil)

	// Mock status update
	mockUserStatus := db.UserStatus{
		UserID:         user1.ID,
		WorkspaceID:    workspace.ID,
		Status:         "away",
		CustomStatus:   sql.NullString{String: "In a meeting", Valid: true},
		LastActivityAt: time.Now(),
		LastSeenAt:     time.Now(),
		UpdatedAt:      time.Now(),
	}

	store.EXPECT().
		UpsertUserStatus(gomock.Any(), gomock.Any()).
		Times(1).
		Return(mockUserStatus, nil)

	// Mock GetUser for status response (to get user info) - called by toUserStatusResponse
	store.EXPECT().
		GetUser(gomock.Any(), gomock.Eq(user1.ID)).
		Times(1).
		Return(db.User{
			ID:             user1.ID,
			OrganizationID: 1,
			Email:          user1.Email,
			FirstName:      user1.FirstName,
			LastName:       user1.LastName,
			WorkspaceID:    sql.NullInt64{Int64: workspace.ID, Valid: true},
			Role:           "member",
			CreatedAt:      time.Now(),
		}, nil)

	// Create access tokens
	user1Token, _, err := server.tokenMaker.CreateToken(user1.Email, time.Minute)
	require.NoError(t, err)

	user2Token, _, err := server.tokenMaker.CreateToken(user2.Email, time.Minute)
	require.NoError(t, err)

	// Start the hub
	go server.hub.Run()

	// Start server
	testServer := httptest.NewServer(server.router)
	defer testServer.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"

	// Create WebSocket connections for both users
	user1Header := http.Header{}
	user1Header.Set("Authorization", "Bearer "+user1Token)

	user2Header := http.Header{}
	user2Header.Set("Authorization", "Bearer "+user2Token)

	user1Conn, _, err := websocket.DefaultDialer.Dial(wsURL, user1Header)
	require.NoError(t, err)
	defer user1Conn.Close()

	user2Conn, _, err := websocket.DefaultDialer.Dial(wsURL, user2Header)
	require.NoError(t, err)
	defer user2Conn.Close()

	// Read connection established messages
	var user1Established service.WSMessage
	err = user1Conn.ReadJSON(&user1Established)
	require.NoError(t, err)
	require.Equal(t, WSConnectionEstablished, user1Established.Type)

	var user2Established service.WSMessage
	err = user2Conn.ReadJSON(&user2Established)
	require.NoError(t, err)
	require.Equal(t, WSConnectionEstablished, user2Established.Type)

	// Update user1's status via REST API
	statusBody := `{"status": "away", "custom_status": "In a meeting"}`
	statusURL := "/workspace/" + util.IntToString(workspace.ID) + "/status"

	req, err := http.NewRequest(http.MethodPut, statusURL, strings.NewReader(statusBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+user1Token)

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)

	// Both users should receive the status update via WebSocket
	user2Conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var user2StatusMsg service.WSMessage
	err = user2Conn.ReadJSON(&user2StatusMsg)
	require.NoError(t, err)
	require.Equal(t, WSStatusChanged, user2StatusMsg.Type)
	require.Equal(t, workspace.ID, user2StatusMsg.WorkspaceID)
	require.Equal(t, user1.ID, user2StatusMsg.UserID)

	user1Conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var user1StatusMsg service.WSMessage
	err = user1Conn.ReadJSON(&user1StatusMsg)
	require.NoError(t, err)
	require.Equal(t, WSStatusChanged, user1StatusMsg.Type)
	require.Equal(t, workspace.ID, user1StatusMsg.WorkspaceID)
	require.Equal(t, user1.ID, user1StatusMsg.UserID)
}

func TestWebSocketTypingIndicatorFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	config := util.Config{
		TokenSymmetricKey:       util.RandomString(32),
		AccessTokenDuration:     time.Minute,
		WSMaxConnectionsPerUser: 5,
		WSPingInterval:          54 * time.Second,
		WSPongTimeout:           60 * time.Second,
		WSReadBufferSize:        1024,
		WSWriteBufferSize:       1024,
	}

	server, err := NewServer(config, store)
	require.NoError(t, err)

	// Create test data
	workspace := randomWSWorkspace()
	channel := randomWSChannel()
	user1 := randomWSUser()
	user2 := randomWSUser()

	// Mock database calls for WebSocket connections
	store.EXPECT().
		GetUserByEmail(gomock.Any(), gomock.Eq(user1.Email)).
		Times(1).
		Return(db.User{
			ID:             user1.ID,
			OrganizationID: 1,
			Email:          user1.Email,
			FirstName:      user1.FirstName,
			LastName:       user1.LastName,
			WorkspaceID:    sql.NullInt64{Int64: workspace.ID, Valid: true},
			Role:           "member",
		}, nil)

	store.EXPECT().
		GetUserByEmail(gomock.Any(), gomock.Eq(user2.Email)).
		Times(1).
		Return(db.User{
			ID:             user2.ID,
			OrganizationID: 1,
			Email:          user2.Email,
			FirstName:      user2.FirstName,
			LastName:       user2.LastName,
			WorkspaceID:    sql.NullInt64{Int64: workspace.ID, Valid: true},
			Role:           "member",
		}, nil)

	// Create access tokens
	user1Token, _, err := server.tokenMaker.CreateToken(user1.Email, time.Minute)
	require.NoError(t, err)

	user2Token, _, err := server.tokenMaker.CreateToken(user2.Email, time.Minute)
	require.NoError(t, err)

	// Start the hub
	go server.hub.Run()

	// Start server
	testServer := httptest.NewServer(server.router)
	defer testServer.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"

	// Create WebSocket connections for both users
	user1Header := http.Header{}
	user1Header.Set("Authorization", "Bearer "+user1Token)

	user2Header := http.Header{}
	user2Header.Set("Authorization", "Bearer "+user2Token)

	user1Conn, _, err := websocket.DefaultDialer.Dial(wsURL, user1Header)
	require.NoError(t, err)
	defer user1Conn.Close()

	user2Conn, _, err := websocket.DefaultDialer.Dial(wsURL, user2Header)
	require.NoError(t, err)
	defer user2Conn.Close()

	// Read connection established messages
	var user1Established service.WSMessage
	err = user1Conn.ReadJSON(&user1Established)
	require.NoError(t, err)

	var user2Established service.WSMessage
	err = user2Conn.ReadJSON(&user2Established)
	require.NoError(t, err)

	// Send typing indicator via WebSocket from user1
	typingMsg := map[string]interface{}{
		"type":       "typing_start",
		"channel_id": float64(channel.ID),
	}
	err = user1Conn.WriteJSON(typingMsg)
	require.NoError(t, err)

	// User2 should receive the typing indicator
	user2Conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var user2TypingMsg service.WSMessage
	err = user2Conn.ReadJSON(&user2TypingMsg)
	require.NoError(t, err)
	require.Equal(t, WSUserTyping, user2TypingMsg.Type)
	require.Equal(t, workspace.ID, user2TypingMsg.WorkspaceID)
	require.Equal(t, channel.ID, *user2TypingMsg.ChannelID)
	require.Equal(t, user1.ID, user2TypingMsg.UserID)

	// Check typing data
	typingData, ok := user2TypingMsg.Data.(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, float64(user1.ID), typingData["user_id"])
	require.Equal(t, true, typingData["typing"])

	// Send typing stop
	typingStopMsg := map[string]interface{}{
		"type":       "typing_stop",
		"channel_id": float64(channel.ID),
	}
	err = user1Conn.WriteJSON(typingStopMsg)
	require.NoError(t, err)

	// User2 should receive the typing stop indicator
	user2Conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var user2TypingStopMsg service.WSMessage
	err = user2Conn.ReadJSON(&user2TypingStopMsg)
	require.NoError(t, err)
	require.Equal(t, WSUserTyping, user2TypingStopMsg.Type)

	// Check typing stop data
	typingStopData, ok := user2TypingStopMsg.Data.(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, false, typingStopData["typing"])
}
