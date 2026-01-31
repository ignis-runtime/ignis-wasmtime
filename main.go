package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/ignis-runtime/ignis-wasmtime/api/rest/server"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/v1/routes"
	"github.com/ignis-runtime/ignis-wasmtime/internal/cache"
	"github.com/ignis-runtime/ignis-wasmtime/internal/config"
	"github.com/ignis-runtime/ignis-wasmtime/internal/models"
	"github.com/ignis-runtime/ignis-wasmtime/internal/repository"
	"github.com/ignis-runtime/ignis-wasmtime/internal/storage"
)

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
	runtimeRepo := repository.NewRuntimeRepository(db)

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
	srv := server.NewServer(addr, redisCache, runtimeRepo, s3Storage)
	routes.RegisterRoutes(srv)

	log.Printf("Starting Gin HTTP server on port %s", addr)
	if err := srv.Run(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
