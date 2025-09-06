package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	db "github.com/heyrmi/goslack/db/sqlc"
	"github.com/heyrmi/goslack/service"
	"github.com/heyrmi/goslack/token"
	"github.com/heyrmi/goslack/util"
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
}

// NewServer creates a new HTTP server and set up routing.
func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	userService := service.NewUserService(store, tokenMaker, config)
	organizationService := service.NewOrganizationService(store)
	workspaceService := service.NewWorkspaceService(store, userService)
	channelService := service.NewChannelService(store, userService, workspaceService)
	messageService := service.NewMessageService(store, userService)
	statusService := service.NewStatusService(store)

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
	}

	server.setupRouter()
	return server, nil
}

func (server *Server) setupRouter() {
	router := gin.Default()

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

	server.router = router
}

// Start runs the HTTP server on a specific address.
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
