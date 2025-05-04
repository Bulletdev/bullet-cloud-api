package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds application configuration.
type Config struct {
	DatabaseURL string
	JWTSecret   string
}

// Load loads configuration from environment variables.
// It looks for a .env file first for local development.
func Load() *Config {
	// Load .env file if it exists (useful for local development)
	err := godotenv.Load()
	if err != nil {
		// Don't fail if .env is not found, just log it
		log.Println("No .env file found, proceeding with environment variables")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Fallback or default can be set here if needed, but for Supabase it's required
		log.Fatal("DATABASE_URL environment variable not set")
	}

	// Carregar o segredo JWT
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable not set")
	}

	return &Config{
		DatabaseURL: dbURL,
		JWTSecret:   jwtSecret,
	}
}
