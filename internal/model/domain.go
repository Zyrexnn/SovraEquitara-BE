package model

import (
	"time"

	"github.com/google/uuid"
)

// Profile Model
type Profile struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Email     string    `gorm:"unique;not null" json:"email"`
	Points    int       `gorm:"default:0" json:"points"`
	Role      string    `gorm:"default:'USER'" json:"role"`
	FullName  string    `json:"full_name"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Report Model - latitude and longitude are computed from PostGIS 'location' column
type Report struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProfileID   uuid.UUID `gorm:"type:uuid;not null" json:"profile_id"`
	Description    string    `gorm:"not null" json:"description"`
	PhoneNumber    *string   `gorm:"" json:"phone_number,omitempty"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	LocationDetail string    `json:"location_detail"`
	Status         string    `gorm:"default:'PENDING'" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Request & Response Payloads
type AuthRequest struct {
	Name     string `json:"name,omitempty"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type VerifyOTPRequest struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

type CreateReportRequest struct {
	Description    string  `json:"description" validate:"required"`
	PhoneNumber    *string `json:"phone_number"` // Optional
	Lat            float64 `json:"latitude" validate:"required"`
	Lng            float64 `json:"longitude" validate:"required"`
	LocationDetail string  `json:"location_detail"`
}

type UpdateProfileRequest struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Email    string `json:"email"`
	Token    string `json:"token"`
	Password string `json:"password"`
}
