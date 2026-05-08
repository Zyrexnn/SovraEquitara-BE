package main
import (
	"fmt"
	"log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"sovraequitara-be/internal/config"
	"sovraequitara-be/internal/model"
)
func main() {
	cfg := config.LoadConfig()
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: cfg.DatabaseURL,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil { log.Fatal(err) }
	
	var users []model.Profile
	db.Find(&users)
	fmt.Printf("Total Users in DB: %d\n", len(users))
	for _, u := range users {
		fmt.Printf("- ID: %s | Name: %s | Email: %s | Role: %s | Points: %d\n", u.ID, u.FullName, u.Email, u.Role, u.Points)
	}
}
