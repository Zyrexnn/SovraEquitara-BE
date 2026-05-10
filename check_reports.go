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
	
	var reports []model.Report
	db.Find(&reports)
	fmt.Printf("Total Reports in DB: %d\n", len(reports))
	for _, r := range reports {
		fmt.Printf("- ID: %s | Status: %s | ProfileID: %s\n", r.ID, r.Status, r.ProfileID)
	}
}
