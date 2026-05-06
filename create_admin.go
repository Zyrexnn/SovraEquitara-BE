package main

import (
	"fmt"
	"log"

	"sovraequitara-be/internal/config"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := config.LoadConfig()

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  cfg.DatabaseURL,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Connected to database successfully!")

	// Hash admin password using bcrypt
	adminPassword := "0721"
	hashedPw, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	// Upsert admin profile (insert or update if email already exists)
	query := `INSERT INTO profiles (email, password_hash, full_name, role)
	          VALUES ($1, $2, $3, $4)
	          ON CONFLICT (email) DO UPDATE SET 
	            password_hash = EXCLUDED.password_hash, 
	            role = EXCLUDED.role,
	            updated_at = NOW()`

	result := db.Exec(query, "ikhsan@admin.com", string(hashedPw), "Admin Ikhsan", "admin")
	if result.Error != nil {
		log.Fatalf("Failed to create admin: %v", result.Error)
	}

	fmt.Println("Admin profile created/updated successfully!")
	fmt.Println("  Email:    ikhsan@admin.com")
	fmt.Println("  Password: 0721")
	fmt.Println("  Role:     admin")
}
