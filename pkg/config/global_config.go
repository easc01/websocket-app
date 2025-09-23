package config

import (
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

type Config struct {
	// general
	ServerID    string
	WSSPort     string
	JWTSecret   string
	Environment string

	// Redis
	RedisConnectionURI string
}

var AppConfig *Config

func LoadConfig() *Config {
	dotenvErr := godotenv.Load(".env.local")
	if dotenvErr != nil {
		log.Printf(
			"error loading .env file, continuing with system environment variables, %s",
			dotenvErr,
		)
	}

	config := &Config{
		ServerID:    uuid.New().String(),
		WSSPort:     getEnv("WSS_PORT", "8081"),
		JWTSecret:   getEnv("JWT_SECRET", "supersecret"),
		Environment: getEnv("ENVIRONMENT", "development"),

		RedisConnectionURI: getEnv("REDIS_URI", "localhost"),
	}

	AppConfig = config
	return config
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func init() {
	if AppConfig == nil {
		LoadConfig()
		log.Println("Configurations loaded successfully")
	}
}
