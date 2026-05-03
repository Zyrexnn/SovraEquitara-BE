package model

import (
	"time"

	"github.com/google/uuid"
)

// Profile Model
type Profile struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email     string    `gorm:"unique;not null"`
	Points    int       `gorm:"default:0"`
	Role      string    `gorm:"default:'USER'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Report Model
type Report struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ProfileID   uuid.UUID `gorm:"type:uuid;not null"`
	Description string    `gorm:"not null"`
	PhoneNumber *string   `gorm:""` // Optional
	Status      string    `gorm:"default:'PENDING'"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Request & Response Payloads
type OTPRequest struct {
	Email string `json:"email"`
}

type VerifyOTPRequest struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

type CreateReportRequest struct {
	Description string  `json:"description" validate:"required"`
	PhoneNumber *string `json:"phone_number"` // Optional
	Lat         float64 `json:"latitude" validate:"required"`
	Lng         float64 `json:"longitude" validate:"required"`
}
