package config

import (
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

// Config holds the application's configuration.
type Config struct {
	RedisAddr string
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
			RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
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
