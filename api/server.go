package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	db "github.com/rahulmishra/goslack/db/sqlc"
	"github.com/rahulmishra/goslack/service"
	"github.com/rahulmishra/goslack/token"
	"github.com/rahulmishra/goslack/util"
)

// Server serves HTTP requests for our banking service.
type Server struct {
	config              util.Config
	store               db.Store
	tokenMaker          token.Maker
	router              *gin.Engine
	userService         *service.UserService
	organizationService *service.OrganizationService
}

// NewServer creates a new HTTP server and set up routing.
func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	userService := service.NewUserService(store, tokenMaker, config)
	organizationService := service.NewOrganizationService(store)

	server := &Server{
		config:              config,
		store:               store,
		tokenMaker:          tokenMaker,
		userService:         userService,
		organizationService: organizationService,
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

	server.router = router
}

// Start runs the HTTP server on a specific address.
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
