package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	SupabaseURL       string
	SupabaseKey       string
	SupabaseJWTSecret string
	DatabaseURL       string
	Port              string
	SMTPHost          string
	SMTPPort          string
	SMTPUser          string
	SMTPPass          string
	SMTPFrom          string
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

	return &Config{
		SupabaseURL:       os.Getenv("SUPABASE_URL"),
		SupabaseKey:       os.Getenv("SUPABASE_KEY"),
		SupabaseJWTSecret: os.Getenv("SUPABASE_JWT_SECRET"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		Port:              port,
		SMTPHost:          os.Getenv("SMTP_HOST"),
		SMTPPort:          os.Getenv("SMTP_PORT"),
		SMTPUser:          os.Getenv("SMTP_USER"),
		SMTPPass:          os.Getenv("SMTP_PASS"),
		SMTPFrom:          os.Getenv("SMTP_FROM"),
	}
}
