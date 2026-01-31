package main

import (
	"fmt"
	"log"

	"github.com/ignis-runtime/ignis-wasmtime/api/rest/server"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/v1/routes"
	_ "github.com/ignis-runtime/ignis-wasmtime/docs"
	"github.com/ignis-runtime/ignis-wasmtime/internal/cache"
	"github.com/ignis-runtime/ignis-wasmtime/internal/config"
	"github.com/ignis-runtime/ignis-wasmtime/internal/models"
	"github.com/ignis-runtime/ignis-wasmtime/internal/repository"
	"github.com/ignis-runtime/ignis-wasmtime/internal/services"
	"github.com/ignis-runtime/ignis-wasmtime/internal/storage"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// @title Ignis Wasmtime API
// @version 1.0
// @description A serverless runtime for WebAssembly and JavaScript modules
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and then your personal token.
//
//go:generate swag init -g main.go --parseInternal --parseDependency --output docs
func main() {
	// Initialize configuration
	cfg := config.GetConfig()

	// Initialize Redis cache
	redisCache := cache.NewRedisCache(cfg.RedisAddr)

	// Initialize PostgreSQL database
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort, cfg.DBSSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(&models.Runtime{}); err != nil {
		log.Fatalf("Failed to migrate database schema: %v", err)
	}

	// Initialize runtime repository
	deploymentRepository := repository.NewDeploymentRepository(db)

	// Initialize S3 storage - it's required now
	if cfg.S3Endpoint == "" || cfg.S3AccessKeyID == "" || cfg.S3SecretKey == "" || cfg.S3BucketName == "" {
		log.Fatal("S3 configuration is required: S3_ENDPOINT, S3_ACCESS_KEY_ID, S3_SECRET_KEY, and S3_BUCKET_NAME must be set")
	}

	s3Cfg := storage.S3Config{
		Endpoint:        cfg.S3Endpoint,
		AccessKeyID:     cfg.S3AccessKeyID,
		SecretAccessKey: cfg.S3SecretKey,
		BucketName:      cfg.S3BucketName,
		Region:          cfg.S3Region,
	}

	s3Storage, err := storage.NewS3Storage(s3Cfg)
	if err != nil {
		log.Fatalf("Failed to initialize S3 storage: %v", err)
	}
	log.Println("S3 storage initialized successfully")

	addr := ":8080"

	// Initialize deploymentService
	deployService := services.NewDeploymentService(deploymentRepository, s3Storage, cfg)
	srv := server.NewServer(addr, redisCache, deployService)

	// Register Swagger documentation and UI
	srv.Engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	routes.RegisterRoutes(srv, deployService, redisCache)

	log.Printf("Starting Gin HTTP server on port %s", addr)
	if err := srv.Run(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
