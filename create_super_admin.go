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

	// Upsert super admin profile
	query := `INSERT INTO profiles (email, password_hash, full_name, role)
	          VALUES ($1, $2, $3, $4)
	          ON CONFLICT (email) DO UPDATE SET 
	            password_hash = EXCLUDED.password_hash, 
	            role = EXCLUDED.role,
	            updated_at = NOW()`

	result := db.Exec(query, "super@admin.com", string(hashedPw), "Super Admin", "super_admin")
	if result.Error != nil {
		log.Fatalf("Failed to create super admin: %v", result.Error)
	}

	fmt.Println("Super Admin profile created/updated successfully!")
	fmt.Println("  Email:    super@admin.com")
	fmt.Println("  Password: 0721")
	fmt.Println("  Role:     super_admin")
}
