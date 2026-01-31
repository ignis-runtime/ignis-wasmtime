package config

import (
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

// Config holds the application's configuration.
type Config struct {
	RedisAddr        string
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
	DBSSLMode        string
	S3Endpoint       string
	S3AccessKeyID    string
	S3SecretKey      string
	S3BucketName     string
	S3Region         string
}

var (
	once     sync.Once
	instance *Config
)

// GetConfig returns the singleton instance of the Config.
// It loads the configuration from an .env file on its first call.
func GetConfig() *Config {
	once.Do(func() {
		// Load .env file. You can specify the path to your .env file.
		// If no path is provided, it will look for a .env file in the current directory.
		err := godotenv.Load()
		if err != nil {
			log.Println("No .env file found, using default environment variables")
		}

		instance = &Config{
			RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
			DBHost:           getEnv("DB_HOST", "localhost"),
			DBPort:           getEnv("DB_PORT", "5432"),
			DBUser:           getEnv("DB_USER", "postgres"),
			DBPassword:       getEnv("DB_PASSWORD", "postgres"),
			DBName:           getEnv("DB_NAME", "postgres"),
			DBSSLMode:        getEnv("DB_SSL_MODE", "disable"),
			S3Endpoint:       getEnv("S3_ENDPOINT", ""),
			S3AccessKeyID:    getEnv("S3_ACCESS_KEY_ID", ""),
			S3SecretKey:      getEnv("S3_SECRET_KEY", ""),
			S3BucketName:     getEnv("S3_BUCKET_NAME", ""),
			S3Region:         getEnv("S3_REGION", "auto"),
		}
	})
	return instance
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
