package main

import (
	"log"

	"sovraequitara-be/internal/config"
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
		log.Fatalf("Gagal connect ke DB: %v", err)
	}

	log.Println("Menghapus comments...")
	db.Exec("DELETE FROM comments;")
	
	log.Println("Menghapus votes...")
	db.Exec("DELETE FROM votes;")
	
	log.Println("Menghapus reports...")
	db.Exec("DELETE FROM reports;")
	
	log.Println("Menghapus user biasa...")
	db.Exec("DELETE FROM profiles WHERE role != 'admin';")
	
	log.Println("Database berhasil di-reset!")
}
