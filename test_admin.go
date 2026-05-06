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
	
	var admins []model.Profile
	db.Where("role = ?", "admin").Find(&admins)
	fmt.Printf("Total Admin: %d\n", len(admins))
	for _, a := range admins {
		fmt.Printf("- Email: %s (Role: %s)\n", a.Email, a.Role)
	}
}
