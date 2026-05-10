package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	// Database
	DatabaseURL string

	// JWT (self-managed, no Supabase dependency)
	JWTSecret string

	// Server
	Port string

	// SMTP
	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	SMTPFrom string

	// AI APIs
	GeminiAPIKey string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, reading configuration from environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "sovra-equitara-secret-key-change-in-production"
		log.Println("[WARNING] JWT_SECRET not set, using default. SET THIS IN PRODUCTION!")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbPassword := os.Getenv("DB_PASSWORD")
		if dbPassword == "" {
			dbPassword = "password" // fallback
		}
		dbURL = "postgresql://postgres:" + dbPassword + "@localhost:5432/sovra_equitara?sslmode=disable"
	}

	return &Config{
		DatabaseURL: dbURL,
		JWTSecret:   jwtSecret,
		Port:        port,
		SMTPHost:    os.Getenv("SMTP_HOST"),
		SMTPPort:    os.Getenv("SMTP_PORT"),
		SMTPUser:    os.Getenv("SMTP_USER"),
		SMTPPass:    os.Getenv("SMTP_PASS"),
		SMTPFrom:    os.Getenv("SMTP_FROM"),
		GeminiAPIKey: os.Getenv("GEMINI_API_KEY"),
	}
}
