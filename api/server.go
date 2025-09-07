package api

import (
	"fmt"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	db "github.com/heyrmi/goslack/db/sqlc"
	"github.com/heyrmi/goslack/service"
	"github.com/heyrmi/goslack/token"
	"github.com/heyrmi/goslack/util"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Server serves HTTP requests for our GoSlack service.
type Server struct {
	config              util.Config
	store               db.Store
	tokenMaker          token.Maker
	router              *gin.Engine
	userService         *service.UserService
	organizationService *service.OrganizationService
	workspaceService    *service.WorkspaceService
	channelService      *service.ChannelService
	messageService      *service.MessageService
	statusService       *service.StatusService
	fileService         *service.FileService
	hub                 *Hub // WebSocket hub
}

// NewServer creates a new HTTP server and set up routing.
func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	// Create WebSocket hub
	hub := NewHub(config)

	userService := service.NewUserService(store, tokenMaker, config)
	organizationService := service.NewOrganizationService(store)
	workspaceService := service.NewWorkspaceService(store, userService)
	channelService := service.NewChannelService(store, userService, workspaceService)
	messageService := service.NewMessageService(store, userService, hub) // Pass hub to message service
	statusService := service.NewStatusService(store, hub)                // Pass hub to status service
	fileService := service.NewFileService(store, config)                 // Add file service

	server := &Server{
		config:              config,
		store:               store,
		tokenMaker:          tokenMaker,
		userService:         userService,
		organizationService: organizationService,
		workspaceService:    workspaceService,
		channelService:      channelService,
		messageService:      messageService,
		statusService:       statusService,
		fileService:         fileService,
		hub:                 hub,
	}

	server.setupRouter()
	return server, nil
}

func (server *Server) setupRouter() {
	router := gin.Default()

	// Configure CORS middleware
	config := cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173", "http://localhost:8080"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(config))

	// Swagger documentation endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API info endpoint
	router.GET("/api/info", server.getAPIInfo)

	// Public routes (no authentication required)
	router.POST("/organizations", server.createOrganization)
	router.GET("/organizations/:id", server.getOrganization)
	router.GET("/organizations", server.listOrganizations)
	router.POST("/users", server.createUser)
	router.POST("/users/login", server.loginUser)

	// Protected routes (authentication required)
	authRoutes := router.Group("/").Use(authMiddleware(server.tokenMaker))
	authRoutes.GET("/users/:id", server.getUser)
	authRoutes.PUT("/users/:id/profile", server.updateUserProfile)
	authRoutes.PUT("/users/:id/password", server.changePassword)
	authRoutes.GET("/users", server.listUsers)
	authRoutes.PUT("/organizations/:id", server.updateOrganization)
	authRoutes.DELETE("/organizations/:id", server.deleteOrganization)

	// Protected routes with user context
	authWithUserRoutes := router.Group("/").Use(authWithUserMiddleware(server.tokenMaker, server.userService))

	// WebSocket endpoint
	authWithUserRoutes.GET("/ws", server.handleWebSocket)

	// Workspace routes (no workspace-specific auth needed)
	authWithUserRoutes.POST("/workspaces", server.createWorkspace)
	authWithUserRoutes.GET("/workspaces", server.listWorkspaces)
	authWithUserRoutes.GET("/workspaces/:id", server.getWorkspace)

	// Workspace admin routes (require admin of the workspace)
	authWithUserRoutes.PUT("/workspaces/:id", requireWorkspaceAdmin(server.userService), server.updateWorkspace)
	authWithUserRoutes.DELETE("/workspaces/:id", requireWorkspaceAdmin(server.userService), server.deleteWorkspace)

	// Workspace member routes (require membership of the workspace)
	authWithUserRoutes.POST("/workspaces/:id/channels", requireWorkspaceMember(server.userService), server.createChannel)
	authWithUserRoutes.GET("/workspaces/:id/channels", requireWorkspaceMember(server.userService), server.listChannels)

	// Channel routes (with individual access checks)
	authWithUserRoutes.GET("/channels/:id", server.getChannel)
	authWithUserRoutes.PUT("/channels/:id", server.updateChannel)
	authWithUserRoutes.DELETE("/channels/:id", server.deleteChannel)

	// User role management (admin only, same workspace)
	authWithUserRoutes.PATCH("/users/:user_id/role", requireSameWorkspaceForUserRole(server.userService), server.updateUserRole)

	// Message routes
	authWithUserRoutes.POST("/workspace/:id/channels/:channel_id/messages", requireWorkspaceMember(server.userService), server.sendChannelMessage)
	authWithUserRoutes.POST("/workspace/:id/messages/direct", requireWorkspaceMember(server.userService), server.sendDirectMessage)
	authWithUserRoutes.GET("/workspace/:id/channels/:channel_id/messages", requireWorkspaceMember(server.userService), server.getChannelMessages)
	authWithUserRoutes.GET("/workspace/:id/messages/direct/:user_id", requireWorkspaceMember(server.userService), server.getDirectMessages)
	authWithUserRoutes.PUT("/messages/:message_id", server.editMessage)
	authWithUserRoutes.DELETE("/messages/:message_id", server.deleteMessage)
	authWithUserRoutes.GET("/messages/:message_id", server.getMessage)

	// Status routes
	authWithUserRoutes.PUT("/workspace/:id/status", requireWorkspaceMember(server.userService), server.updateUserStatus)
	authWithUserRoutes.GET("/workspace/:id/status/:user_id", requireWorkspaceMember(server.userService), server.getUserStatus)
	authWithUserRoutes.GET("/workspace/:id/status", requireWorkspaceMember(server.userService), server.getWorkspaceUserStatuses)
	authWithUserRoutes.POST("/workspace/:id/activity", requireWorkspaceMember(server.userService), server.updateUserActivity)

	// Typing indicator endpoint
	authWithUserRoutes.POST("/workspaces/:id/channels/:channel_id/typing", requireWorkspaceMember(server.userService), server.handleTyping)

	// File routes
	authWithUserRoutes.POST("/files/upload", server.uploadFile)
	authWithUserRoutes.GET("/files/:id", server.getFile)
	authWithUserRoutes.GET("/files/:id/download", server.downloadFile)
	authWithUserRoutes.DELETE("/files/:id", server.deleteFile)
	authWithUserRoutes.GET("/workspaces/:id/files", requireWorkspaceMember(server.userService), server.listWorkspaceFiles)
	authWithUserRoutes.GET("/workspaces/:id/files/stats", requireWorkspaceMember(server.userService), server.getFileStats)
	authWithUserRoutes.POST("/files/message", server.sendFileMessage)

	server.router = router
}

// Start runs the HTTP server on a specific address.
func (server *Server) Start(address string) error {
	// Start the WebSocket hub in a separate goroutine
	go server.hub.Run()

	return server.router.Run(address)
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}

// @Summary Get API Information
// @Description Get general information about the GoSlack API
// @Tags system
// @Produce json
// @Success 200 {object} map[string]interface{} "API information"
// @Router /api/info [get]
func (server *Server) getAPIInfo(ctx *gin.Context) {
	info := gin.H{
		"name":        "GoSlack API",
		"version":     "1.0",
		"description": "A Slack-like collaboration platform API with real-time messaging, file sharing, and workspace management.",
		"endpoints": gin.H{
			"swagger_ui":  "/swagger/index.html",
			"swagger_doc": "/swagger/doc.json",
		},
		"features": []string{
			"User Management",
			"Organization Management",
			"Workspace Management",
			"Channel Management",
			"Real-time Messaging",
			"File Management",
			"Status Management",
			"WebSocket Support",
		},
	}

	ctx.JSON(200, info)
}
