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
	config                     util.Config
	store                      db.Store
	tokenMaker                 token.Maker
	router                     *gin.Engine
	userService                *service.UserService
	organizationService        *service.OrganizationService
	workspaceService           *service.WorkspaceService
	workspaceInvitationService *service.WorkspaceInvitationService
	channelService             *service.ChannelService
	messageService             *service.MessageService
	messageEnhancedService     *service.MessageEnhancedService
	statusService              *service.StatusService
	fileService                *service.FileService
	hub                        *Hub // WebSocket hub

	// New security services
	emailService     *service.EmailService
	authService      *service.AuthService
	twoFactorService *service.TwoFactorService
	rateLimiter      *RateLimiter
}

// NewServer creates a new HTTP server and set up routing.
func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	// Create WebSocket hub
	hub := NewHub(config)

	// Create email service
	emailConfig := service.EmailConfig{
		SMTPHost:     config.SMTPHost,
		SMTPPort:     config.SMTPPort,
		SMTPUsername: config.SMTPUsername,
		SMTPPassword: config.SMTPPassword,
		FromEmail:    config.FromEmail,
		FromName:     config.FromName,
		BaseURL:      config.BaseURL,
	}
	emailService := service.NewEmailService(emailConfig)

	// Create rate limiter
	var rateLimitConfig RateLimitConfig
	switch config.RateLimitMode {
	case "strict":
		rateLimitConfig = StrictRateLimitConfig()
	case "permissive":
		rateLimitConfig = PermissiveRateLimitConfig()
	default:
		rateLimitConfig = DefaultRateLimitConfig()
	}

	// Override with custom values if provided
	if config.AuthRequestsPerMinute > 0 {
		rateLimitConfig.AuthRequestsPerMinute = config.AuthRequestsPerMinute
	}
	if config.APIRequestsPerMinute > 0 {
		rateLimitConfig.APIRequestsPerMinute = config.APIRequestsPerMinute
	}
	if config.UploadRequestsPerMinute > 0 {
		rateLimitConfig.UploadRequestsPerMinute = config.UploadRequestsPerMinute
	}
	if config.MessageRequestsPerMinute > 0 {
		rateLimitConfig.MessageRequestsPerMinute = config.MessageRequestsPerMinute
	}

	var rateLimiter *RateLimiter
	if config.EnableRateLimit {
		rateLimiter = NewRateLimiter(rateLimitConfig)
	}

	userService := service.NewUserService(store, tokenMaker, config)
	organizationService := service.NewOrganizationService(store)
	workspaceService := service.NewWorkspaceService(store, userService)
	workspaceInvitationService := service.NewWorkspaceInvitationService(store)
	channelService := service.NewChannelService(store, userService, workspaceService)
	messageService := service.NewMessageService(store, userService, hub) // Pass hub to message service

	// Create hub adapter for enhanced message service
	hubAdapter := NewHubAdapter(hub)
	messageEnhancedService := service.NewMessageEnhancedService(store, hubAdapter) // Enhanced message features

	statusService := service.NewStatusService(store, hub) // Pass hub to status service
	fileService := service.NewFileService(store, config)  // Add file service

	// Create security services
	authService := service.NewAuthService(store, tokenMaker, emailService, config)
	twoFactorService := service.NewTwoFactorService(store)

	// Set up circular dependency (user service needs auth service for lockout checks)
	userService.SetAuthService(authService)

	server := &Server{
		config:                     config,
		store:                      store,
		tokenMaker:                 tokenMaker,
		userService:                userService,
		organizationService:        organizationService,
		workspaceService:           workspaceService,
		workspaceInvitationService: workspaceInvitationService,
		channelService:             channelService,
		messageService:             messageService,
		messageEnhancedService:     messageEnhancedService,
		statusService:              statusService,
		fileService:                fileService,
		hub:                        hub,
		emailService:               emailService,
		authService:                authService,
		twoFactorService:           twoFactorService,
		rateLimiter:                rateLimiter,
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

	// Apply rate limiting if enabled
	var authRateLimit, apiRateLimit, uploadRateLimit, messageRateLimit gin.HandlerFunc
	if server.rateLimiter != nil {
		authRateLimit = server.rateLimiter.RateLimitMiddleware("auth")
		apiRateLimit = server.rateLimiter.RateLimitMiddleware("api")
		uploadRateLimit = server.rateLimiter.RateLimitMiddleware("upload")
		messageRateLimit = server.rateLimiter.RateLimitMiddleware("message")
	} else {
		// No-op middleware if rate limiting is disabled
		noOp := func(ctx *gin.Context) { ctx.Next() }
		authRateLimit = noOp
		apiRateLimit = noOp
		uploadRateLimit = noOp
		messageRateLimit = noOp
	}

	// Public routes (no authentication required, with auth rate limiting)
	publicAuthRoutes := router.Group("/").Use(authRateLimit)
	publicAuthRoutes.POST("/organizations", server.createOrganization)
	publicAuthRoutes.POST("/users", server.createUser)
	publicAuthRoutes.POST("/users/login", server.loginUser)

	// Authentication endpoints (public, with auth rate limiting)
	publicAuthRoutes.POST("/auth/send-verification", server.sendEmailVerification)
	publicAuthRoutes.POST("/auth/verify-email", server.verifyEmail)
	publicAuthRoutes.POST("/auth/forgot-password", server.requestPasswordReset)
	publicAuthRoutes.POST("/auth/reset-password", server.resetPassword)

	// Public API routes (with general API rate limiting)
	publicAPIRoutes := router.Group("/").Use(apiRateLimit)
	publicAPIRoutes.GET("/organizations/:id", server.getOrganization)
	publicAPIRoutes.GET("/organizations", server.listOrganizations)

	// Protected routes (authentication required, with API rate limiting)
	protectedRoutes := router.Group("/").Use(authMiddleware(server.tokenMaker), apiRateLimit)
	protectedRoutes.GET("/users/:id", server.getUser)
	protectedRoutes.PUT("/users/:id/profile", server.updateUserProfile)
	protectedRoutes.PUT("/users/:id/password", server.changePassword)
	protectedRoutes.GET("/users", server.listUsers)
	protectedRoutes.PUT("/organizations/:id", server.updateOrganization)
	protectedRoutes.DELETE("/organizations/:id", server.deleteOrganization)

	// Protected routes with user context (with API rate limiting)
	authWithUserRoutes := router.Group("/").Use(authWithUserMiddleware(server.tokenMaker, server.userService), apiRateLimit)

	// 2FA and security endpoints (protected)
	authWithUserRoutes.POST("/auth/2fa/setup", server.setup2FA)
	authWithUserRoutes.POST("/auth/2fa/verify", server.verify2FA)
	authWithUserRoutes.POST("/auth/2fa/disable", server.disable2FA)
	authWithUserRoutes.POST("/auth/2fa/backup-codes", server.regenerateBackupCodes)
	authWithUserRoutes.GET("/auth/2fa/status", server.get2FAStatus)
	authWithUserRoutes.GET("/auth/security-events", server.getSecurityEvents)

	// WebSocket endpoint
	authWithUserRoutes.GET("/ws", server.handleWebSocket)

	// Workspace routes (no workspace-specific auth needed)
	authWithUserRoutes.POST("/workspaces", server.createWorkspace)
	authWithUserRoutes.GET("/workspaces", server.listWorkspaces)
	authWithUserRoutes.GET("/workspaces/:id", server.getWorkspace)

	// Workspace admin routes (require admin of the workspace)
	authWithUserRoutes.PUT("/workspaces/:id", requireWorkspaceAdmin(server.userService), server.updateWorkspace)
	authWithUserRoutes.DELETE("/workspaces/:id", requireWorkspaceAdmin(server.userService), server.deleteWorkspace)

	// Workspace invitation routes (require workspace admin)
	authWithUserRoutes.POST("/workspaces/:id/invitations", requireWorkspaceAdmin(server.userService), server.inviteUserToWorkspace)
	authWithUserRoutes.GET("/workspaces/:id/invitations", requireWorkspaceAdmin(server.userService), server.listWorkspaceInvitations)

	// Join workspace route (any authenticated user)
	authWithUserRoutes.POST("/workspaces/join", server.joinWorkspace)

	// Workspace member management routes
	authWithUserRoutes.GET("/workspaces/:id/members", requireWorkspaceMember(server.userService), server.listWorkspaceMembers)
	authWithUserRoutes.DELETE("/workspaces/:id/members/:user_id", requireWorkspaceAdmin(server.userService), server.removeUserFromWorkspace)
	authWithUserRoutes.PUT("/workspaces/:id/members/:user_id/role", requireWorkspaceAdmin(server.userService), server.updateWorkspaceMemberRole)

	// Workspace member routes (require membership of the workspace)
	authWithUserRoutes.POST("/workspaces/:id/channels", requireWorkspaceMember(server.userService), server.createChannel)
	authWithUserRoutes.GET("/workspaces/:id/channels", requireWorkspaceMember(server.userService), server.listChannels)

	// Channel routes (with individual access checks)
	authWithUserRoutes.GET("/channels/:id", server.getChannel)
	authWithUserRoutes.PUT("/channels/:id", server.updateChannel)
	authWithUserRoutes.DELETE("/channels/:id", server.deleteChannel)

	// User role management (admin only, same workspace)
	authWithUserRoutes.PATCH("/users/:user_id/role", requireSameWorkspaceForUserRole(server.userService), server.updateUserRole)

	// Message routes (with message rate limiting)
	messageRoutes := router.Group("/").Use(authWithUserMiddleware(server.tokenMaker, server.userService), messageRateLimit)
	messageRoutes.POST("/workspace/:id/channels/:channel_id/messages", requireWorkspaceMember(server.userService), server.sendChannelMessage)
	messageRoutes.POST("/workspace/:id/messages/direct", requireWorkspaceMember(server.userService), server.sendDirectMessage)
	messageRoutes.GET("/workspace/:id/channels/:channel_id/messages", requireWorkspaceMember(server.userService), server.getChannelMessages)
	messageRoutes.GET("/workspace/:id/messages/direct/:user_id", requireWorkspaceMember(server.userService), server.getDirectMessages)
	messageRoutes.PUT("/messages/:message_id", server.editMessage)
	messageRoutes.DELETE("/messages/:message_id", server.deleteMessage)
	messageRoutes.GET("/messages/:message_id", server.getMessage)

	// Status routes
	authWithUserRoutes.PUT("/workspace/:id/status", requireWorkspaceMember(server.userService), server.updateUserStatus)
	authWithUserRoutes.GET("/workspace/:id/status/:user_id", requireWorkspaceMember(server.userService), server.getUserStatus)
	authWithUserRoutes.GET("/workspace/:id/status", requireWorkspaceMember(server.userService), server.getWorkspaceUserStatuses)
	authWithUserRoutes.POST("/workspace/:id/activity", requireWorkspaceMember(server.userService), server.updateUserActivity)

	// Typing indicator endpoint
	authWithUserRoutes.POST("/workspaces/:id/channels/:channel_id/typing", requireWorkspaceMember(server.userService), server.handleTyping)

	// File routes (with upload rate limiting)
	fileRoutes := router.Group("/").Use(authWithUserMiddleware(server.tokenMaker, server.userService), uploadRateLimit)
	fileRoutes.POST("/files/upload", server.uploadFile)
	fileRoutes.GET("/files/:id", server.getFile)
	fileRoutes.GET("/files/:id/download", server.downloadFile)
	fileRoutes.DELETE("/files/:id", server.deleteFile)
	fileRoutes.GET("/workspaces/:id/files", requireWorkspaceMember(server.userService), server.listWorkspaceFiles)
	fileRoutes.GET("/workspaces/:id/files/stats", requireWorkspaceMember(server.userService), server.getFileStats)
	fileRoutes.POST("/files/message", server.sendFileMessage)

	// Enhanced message routes (threading, reactions, mentions, search, etc.)
	enhancedMessageRoutes := router.Group("/").Use(authWithUserMiddleware(server.tokenMaker, server.userService), messageRateLimit)

	// Threading
	enhancedMessageRoutes.POST("/messages/thread/reply", server.createThreadReply)
	enhancedMessageRoutes.GET("/messages/thread/:thread_id", server.getThreadMessages)
	enhancedMessageRoutes.GET("/messages/thread/:thread_id/info", server.getThreadInfo)

	// Reactions
	enhancedMessageRoutes.POST("/messages/reactions/add", server.addMessageReaction)
	enhancedMessageRoutes.POST("/messages/reactions/remove", server.removeMessageReaction)
	enhancedMessageRoutes.GET("/messages/:message_id/reactions", server.getMessageReactions)

	// Search
	enhancedMessageRoutes.POST("/messages/search", server.searchMessages)

	// Pinning
	enhancedMessageRoutes.POST("/messages/pin", server.pinMessage)
	enhancedMessageRoutes.POST("/messages/:message_id/unpin", server.unpinMessage)
	enhancedMessageRoutes.GET("/channels/:id/pinned", server.getPinnedMessages)

	// Drafts
	enhancedMessageRoutes.POST("/messages/drafts", server.saveDraft)
	enhancedMessageRoutes.GET("/workspaces/:id/drafts", server.getUserDrafts)

	// Mentions and unread
	enhancedMessageRoutes.GET("/workspaces/:id/mentions", server.getUserMentions)
	enhancedMessageRoutes.GET("/workspaces/:id/unread", server.getUnreadMessages)
	enhancedMessageRoutes.POST("/messages/mark-read", server.markAsRead)

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
