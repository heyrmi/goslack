package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/heyrmi/goslack/service"
	"github.com/heyrmi/goslack/util"
)

// WebSocket message types
const (
	WSMessageSent           = "message_sent"
	WSMessageEdited         = "message_edited"
	WSMessageDeleted        = "message_deleted"
	WSStatusChanged         = "status_changed"
	WSUserTyping            = "user_typing"
	WSUserJoinedChannel     = "user_joined_channel"
	WSUserLeftChannel       = "user_left_channel"
	WSConnectionEstablished = "connection_established"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin for now
		// In production, implement proper CORS checking
		return true
	},
}

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from the clients
	broadcast chan *service.WSMessage

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Workspace-based client mapping for efficient broadcasting
	workspaces map[int64]map[*Client]bool

	// Channel-based client mapping for targeted messaging
	channels map[int64]map[*Client]bool

	// User connection tracking for connection limits
	userConnections map[int64][]*Client

	// Configuration
	config util.Config

	// Mutex for thread-safe operations
	mutex sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub(config util.Config) *Hub {
	return &Hub{
		broadcast:       make(chan *service.WSMessage),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		clients:         make(map[*Client]bool),
		workspaces:      make(map[int64]map[*Client]bool),
		channels:        make(map[int64]map[*Client]bool),
		userConnections: make(map[int64][]*Client),
		config:          config,
	}
}

// Client is a middleman between the websocket connection and the hub
type Client struct {
	hub *Hub

	// The websocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan *service.WSMessage

	// User information
	userID      int64
	workspaceID int64
	user        service.UserResponse

	// Connection state
	isActive bool
}

// Run starts the WebSocket hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// registerClient adds a new client to the hub
func (h *Hub) registerClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Check connection limit per user
	userConns := h.userConnections[client.userID]
	if len(userConns) >= h.config.WSMaxConnectionsPerUser {
		// Close the oldest connection
		if len(userConns) > 0 {
			oldestClient := userConns[0]
			oldestClient.conn.Close()
		}
	}

	// Register the client
	h.clients[client] = true

	// Add to workspace mapping
	if h.workspaces[client.workspaceID] == nil {
		h.workspaces[client.workspaceID] = make(map[*Client]bool)
	}
	h.workspaces[client.workspaceID][client] = true

	// Add to user connections tracking
	h.userConnections[client.userID] = append(h.userConnections[client.userID], client)

	// Send connection established message
	connectionMsg := &service.WSMessage{
		Type:        WSConnectionEstablished,
		Data:        gin.H{"message": "WebSocket connection established", "user": client.user},
		WorkspaceID: client.workspaceID,
		UserID:      client.userID,
		Timestamp:   time.Now(),
	}

	select {
	case client.send <- connectionMsg:
	default:
		close(client.send)
		delete(h.clients, client)
	}

	log.Printf("Client registered: user_id=%d, workspace_id=%d, total_clients=%d",
		client.userID, client.workspaceID, len(h.clients))
}

// unregisterClient removes a client from the hub
func (h *Hub) unregisterClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)

		// Remove from workspace mapping
		if workspaceClients, exists := h.workspaces[client.workspaceID]; exists {
			delete(workspaceClients, client)
			if len(workspaceClients) == 0 {
				delete(h.workspaces, client.workspaceID)
			}
		}

		// Remove from user connections
		userConns := h.userConnections[client.userID]
		for i, conn := range userConns {
			if conn == client {
				h.userConnections[client.userID] = append(userConns[:i], userConns[i+1:]...)
				break
			}
		}
		if len(h.userConnections[client.userID]) == 0 {
			delete(h.userConnections, client.userID)
		}

		// Remove from channel mappings
		for channelID, channelClients := range h.channels {
			if _, exists := channelClients[client]; exists {
				delete(channelClients, client)
				if len(channelClients) == 0 {
					delete(h.channels, channelID)
				}
			}
		}

		log.Printf("Client unregistered: user_id=%d, workspace_id=%d, total_clients=%d",
			client.userID, client.workspaceID, len(h.clients))
	}
}

// broadcastMessage sends a message to relevant clients
func (h *Hub) broadcastMessage(message *service.WSMessage) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	// Broadcast to workspace members
	if workspaceClients, exists := h.workspaces[message.WorkspaceID]; exists {
		for client := range workspaceClients {
			// For channel messages, only send to clients that have access to the channel
			if message.ChannelID != nil {
				// TODO: Add channel membership check here if needed
				// For now, send to all workspace members
			}

			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(h.clients, client)
				delete(workspaceClients, client)
			}
		}
	}
}

// BroadcastToWorkspace sends a message to all clients in a workspace
func (h *Hub) BroadcastToWorkspace(workspaceID int64, message *service.WSMessage) {
	message.WorkspaceID = workspaceID
	message.Timestamp = time.Now()

	select {
	case h.broadcast <- message:
	default:
		log.Printf("Warning: broadcast channel full, dropping message")
	}
}

// BroadcastToChannel sends a message to all clients in a specific channel
func (h *Hub) BroadcastToChannel(workspaceID, channelID int64, message *service.WSMessage) {
	message.WorkspaceID = workspaceID
	message.ChannelID = &channelID
	message.Timestamp = time.Now()

	select {
	case h.broadcast <- message:
	default:
		log.Printf("Warning: broadcast channel full, dropping message")
	}
}

// BroadcastToUser sends a message to all connections of a specific user
func (h *Hub) BroadcastToUser(userID int64, message *service.WSMessage) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	message.Timestamp = time.Now()

	if userConns, exists := h.userConnections[userID]; exists {
		for _, client := range userConns {
			select {
			case client.send <- message:
			default:
				// Client's send channel is full, skip
				log.Printf("Warning: client send channel full for user %d", userID)
			}
		}
	}
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(c.hub.config.WSPongTimeout))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.hub.config.WSPongTimeout))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming messages (like typing indicators, ping, etc.)
		var incomingMsg map[string]interface{}
		if err := json.Unmarshal(message, &incomingMsg); err == nil {
			c.handleIncomingMessage(incomingMsg)
		}
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(c.hub.config.WSPingInterval)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleIncomingMessage processes incoming WebSocket messages
func (c *Client) handleIncomingMessage(message map[string]interface{}) {
	messageType, exists := message["type"].(string)
	if !exists {
		return
	}

	switch messageType {
	case "ping":
		// Respond to ping with pong
		pongMsg := &service.WSMessage{
			Type:        "pong",
			Data:        gin.H{"timestamp": time.Now()},
			WorkspaceID: c.workspaceID,
			UserID:      c.userID,
			Timestamp:   time.Now(),
		}
		select {
		case c.send <- pongMsg:
		default:
		}
	case "typing_start":
		// Handle typing indicator start
		if channelID, ok := message["channel_id"].(float64); ok {
			typingMsg := &service.WSMessage{
				Type:        WSUserTyping,
				Data:        gin.H{"user_id": c.userID, "user": c.user, "typing": true},
				WorkspaceID: c.workspaceID,
				ChannelID:   func() *int64 { id := int64(channelID); return &id }(),
				UserID:      c.userID,
				Timestamp:   time.Now(),
			}
			c.hub.BroadcastToChannel(c.workspaceID, int64(channelID), typingMsg)
		}
	case "typing_stop":
		// Handle typing indicator stop
		if channelID, ok := message["channel_id"].(float64); ok {
			typingMsg := &service.WSMessage{
				Type:        WSUserTyping,
				Data:        gin.H{"user_id": c.userID, "user": c.user, "typing": false},
				WorkspaceID: c.workspaceID,
				ChannelID:   func() *int64 { id := int64(channelID); return &id }(),
				UserID:      c.userID,
				Timestamp:   time.Now(),
			}
			c.hub.BroadcastToChannel(c.workspaceID, int64(channelID), typingMsg)
		}
	}
}

// @Summary WebSocket Connection
// @Description Establish WebSocket connection for real-time communication (requires authentication)
// @Tags realtime
// @Security BearerAuth
// @Produce json
// @Success 101 {string} string "WebSocket connection established"
// @Failure 400 {object} map[string]string "WebSocket upgrade failed"
// @Failure 401 {object} map[string]string "Authentication required"
// @Router /ws [get]
func (server *Server) handleWebSocket(c *gin.Context) {
	// Get current user from middleware
	currentUser := getCurrentUser(c)

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Create client
	client := &Client{
		hub:         server.hub,
		conn:        conn,
		send:        make(chan *service.WSMessage, 256),
		userID:      currentUser.ID,
		workspaceID: *currentUser.WorkspaceID,
		user:        currentUser,
		isActive:    true,
	}

	// Register client and start goroutines
	client.hub.register <- client

	// Start the client goroutines
	go client.writePump()
	go client.readPump()
}
