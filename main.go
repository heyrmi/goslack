// @title GoSlack API
// @version 1.0
// @description A Slack-like collaboration platform API with real-time messaging, file sharing, and workspace management.
// @termsOfService https://github.com/heyrmi/goslack

// @contact.name API Support
// @contact.url https://github.com/heyrmi/goslack/issues
// @contact.email support@goslack.dev

// @license.name MIT
// @license.url https://github.com/heyrmi/goslack/blob/main/LICENSE

// @host localhost:8080
// @BasePath /
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/heyrmi/goslack/api"
	db "github.com/heyrmi/goslack/db/sqlc"
	"github.com/heyrmi/goslack/service"
	"github.com/heyrmi/goslack/util"
	_ "github.com/lib/pq"

	_ "github.com/heyrmi/goslack/docs" // Import generated docs
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	store := db.NewStore(conn)
	server, err := api.NewServer(config, store)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}

	// Start background services
	startBackgroundServices(store)

	err = server.Start(config.HTTPServerAddress)
	if err != nil {
		log.Fatal("cannot start server:", err)
	}
}

// startBackgroundServices starts background services like inactivity monitoring
func startBackgroundServices(store db.Store) {
	// Background services don't need WebSocket broadcasting, so pass nil
	statusService := service.NewStatusService(store, nil)

	// Start inactivity monitor in background
	go statusService.StartInactivityMonitor(context.Background())
}
