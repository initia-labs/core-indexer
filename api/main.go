package main

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"
	"github.com/initia-labs/core-indexer/api/config"
	_ "github.com/initia-labs/core-indexer/api/docs"
	"github.com/initia-labs/core-indexer/api/middleware"
	"github.com/initia-labs/core-indexer/api/routes"
	"github.com/initia-labs/core-indexer/pkg/logger"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// @title Core Indexer API
// @version 1.0
// @description This is the API service for the Core Indexer project
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:3000
// @BasePath /
func main() {

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Warn().Msg("No .env file found")
	}

	// Load configuration
	cfg := config.New()

	// Initialize logger
	log := logger.Init(logger.Config{
		Component:   "core-indexer-api",
		ChainID:     "initiation-2",
		Environment: cfg.Environment,
		Level:       zerolog.DebugLevel,
	})

	// Connect to database
	db, err := sql.Open("postgres", cfg.Database.ConnectionString)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatal().Err(err).Msg("Failed to ping database")
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:     "Core Indexer API",
		ReadTimeout: cfg.Server.ReadTimeout,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(middleware.RequestLogger(log))

	// Swagger documentation
	app.Get("/swagger/*", swagger.HandlerDefault)

	// Routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Welcome to Core Indexer API",
		})
	})

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
		})
	})

	// Setup routes
	routes.SetupRoutes(app, db)

	// Start server
	log.Info().Str("port", cfg.Server.Port).Msg("Starting server")
	if err := app.Listen(":" + cfg.Server.Port); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
