package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"sovraequitara-be/internal/config"

	_ "github.com/lib/pq"
)

func main() {
	cfg := config.LoadConfig()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}

	log.Println("Connected to database successfully!")

	sqlFile, err := os.ReadFile("setup.sql")
	if err != nil {
		log.Fatalf("Failed to read setup.sql: %v", err)
	}

	_, err = db.Exec(string(sqlFile))
	if err != nil {
		log.Fatalf("Failed to execute setup.sql: %v", err)
	}

	fmt.Println("Database initialized successfully!")
}
