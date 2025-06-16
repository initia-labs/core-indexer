package main

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"gocloud.dev/blob"
	"gocloud.dev/blob/gcsblob"

	"gocloud.dev/gcp"

	"github.com/initia-labs/core-indexer/api/config"
	"github.com/initia-labs/core-indexer/api/docs"
	_ "github.com/initia-labs/core-indexer/api/docs"
	"github.com/initia-labs/core-indexer/api/middleware"
	"github.com/initia-labs/core-indexer/api/routes"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/logger"
	_ "github.com/lib/pq"
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

// @BasePath /

// @tag.name Block
// @tag.description Block related endpoints

// @tag.name Health
// @tag.description Health check endpoints

// @tag.name Module
// @tag.description Module related endpoints

// @tag.name Nft
// @tag.description Nft related endpoints

// @tag.name Proposal
// @tag.description Proposal related endpoints

// @tag.name Root
// @tag.description Root endpoints

// @tag.name Transaction
// @tag.description Transaction related endpoints

// @tag.name Validator
// @tag.description Validator related endpoints

// initDatabase initializes and returns a database connection
func initDatabase(cfg *config.Config) *gorm.DB {
	dbClient, err := db.NewClient(cfg.Database.ConnectionString)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}

	// Test database connection
	if err := db.Ping(context.Background(), dbClient); err != nil {
		log.Fatal().Err(err).Msg("Failed to ping database")
	}

	return dbClient
}

// initStorage initializes and returns a storage bucket
func initStorage(cfg *config.Config) *blob.Bucket {
	creds, err := gcp.DefaultCredentials(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create credentials")
	}

	client, err := gcp.NewHTTPClient(
		gcp.DefaultTransport(),
		gcp.CredentialsTokenSource(creds))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create HTTP client")
	}

	bucket, err := gcsblob.OpenBucket(context.Background(), client, cfg.Storage.Bucket, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open bucket")
	}

	return bucket
}

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
		ChainID:     cfg.ChainID,
		Environment: cfg.Environment,
		Level:       zerolog.InfoLevel,
	})

	// Initialize database
	dbClient := initDatabase(cfg)
	sqlDB, err := dbClient.DB()
	if err == nil {
		defer sqlDB.Close()
	}

	// Initialize storage
	bucket := initStorage(cfg)
	defer bucket.Close()

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
	swaggerConfig := swagger.Config{
		URL:         "/swagger/doc.json",
		DeepLinking: true,
	}
	app.Get("/swagger/*", swagger.New(swaggerConfig))

	// Update Swagger host with actual port
	docs.SwaggerInfo.Host = "localhost:" + cfg.Server.Port

	// Routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Welcome to Core Indexer API",
		})
	})

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "OK",
		})
	})

	// Setup routes
	routes.SetupRoutes(app, dbClient, bucket)

	// Start server
	log.Info().Str("port", cfg.Server.Port).Msg("Starting server")
	if err := app.Listen(":" + cfg.Server.Port); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
