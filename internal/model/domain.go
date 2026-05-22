package model

import (
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email        string    `gorm:"unique;not null" json:"email"`
	PasswordHash string    `gorm:"column:password_hash;not null" json:"-"`
	FullName     string    `json:"full_name"`
	Phone        string    `json:"phone"`
	AvatarURL    *string   `json:"avatar_url"`
	Points       int       `gorm:"default:0" json:"points"`
	Role         string    `gorm:"default:'USER'" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Category struct {
	ID   int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"not null" json:"name"`
	Slug string `gorm:"not null;unique" json:"slug"`
}

type Report struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProfileID      uuid.UUID `gorm:"type:uuid;not null" json:"profile_id"`
	CategoryID     *int      `json:"category_id"`
	ImageURL       *string   `json:"image_url"`
	Description    string    `gorm:"not null" json:"description"`
	PhoneNumber    *string   `json:"phone_number,omitempty"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	LocationDetail string    `json:"location_detail"`
	VoteCount      int       `gorm:"default:0" json:"vote_count"`
	CommentCount   int       `gorm:"default:0" json:"comment_count"`
	Status         string    `gorm:"default:'PENDING'" json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Comment struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ReportID  uuid.UUID `gorm:"type:uuid;not null" json:"report_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Content   string    `gorm:"not null" json:"content"`
	CreatedAt time.Time `json:"created_at"`
	
	// Optional relationships
	User      *Profile  `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type Vote struct {
	UserID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	ReportID uuid.UUID `gorm:"type:uuid;primaryKey" json:"report_id"`
	VoteType int       `gorm:"not null" json:"vote_type"`
}

type SavedReport struct {
	AdminID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"admin_id"`
	ReportID  uuid.UUID `gorm:"type:uuid;primaryKey" json:"report_id"`
	CreatedAt time.Time `json:"created_at"`

	Report *Report `gorm:"foreignKey:ReportID" json:"report,omitempty"`
}

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
	CategoryID     *int    `json:"category_id"`
	Description    string  `json:"description" validate:"required"`
	PhoneNumber    *string `json:"phone_number"`
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
	Email       string `json:"email"`
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

type VoteRequest struct {
	VoteType int `json:"vote_type"` // 1 or -1
}

type CommentRequest struct {
	Content string `json:"content" validate:"required"`
}

type ReportStats struct {
	Total    int64 `json:"total"`
	Pending  int64 `json:"pending"`
	Resolved int64 `json:"resolved"`
}

type AIAssistantRequest struct {
	Query string `json:"query"`
	Model string `json:"model"` // "local" or "gemini"
}

// ============================================================
// CHAT SYSTEM
// ============================================================

type Conversation struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ParticipantID uuid.UUID `gorm:"type:uuid;not null" json:"participant_id"`
	LastMessage   string    `json:"last_message"`
	LastMessageAt time.Time `json:"last_message_at"`
	UnreadCount   int       `gorm:"default:0" json:"unread_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Relationships
	Participant *Profile `gorm:"foreignKey:ParticipantID" json:"participant,omitempty"`
}

type Message struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ConversationID uuid.UUID `gorm:"type:uuid;not null" json:"conversation_id"`
	SenderID       uuid.UUID `gorm:"type:uuid;not null" json:"sender_id"`
	Content        string    `gorm:"not null" json:"content"`
	IsRead         bool      `gorm:"default:false" json:"is_read"`
	CreatedAt      time.Time `json:"created_at"`

	// Relationships
	Sender *Profile `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
}

type SendMessageRequest struct {
	Content string `json:"content" validate:"required"`
}

// ============================================================
// NOTIFICATIONS SYSTEM
// ============================================================

type Notification struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Title        string     `gorm:"not null" json:"title"`
	Message      string     `gorm:"not null" json:"message"`
	Type         string     `gorm:"default:'INFO'" json:"type"`               // INFO, WARNING, EMERGENCY
	TargetRole   string     `gorm:"default:'ALL'" json:"target_role"`         // ALL, USER, ADMIN, SUPERADMIN, SPECIFIC_USER
	TargetUserID *uuid.UUID `gorm:"type:uuid" json:"target_user_id"`          // For targeted notifications
	ActionURL    *string    `json:"action_url"`                               // URL to redirect user when clicked
	CreatedBy    *uuid.UUID `gorm:"type:uuid" json:"created_by"`              // Can be null if system generated
	CreatedAt    time.Time  `json:"created_at"`

	Creator *Profile `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

type CreateNotificationRequest struct {
	Title        string     `json:"title" validate:"required"`
	Message      string     `json:"message" validate:"required"`
	Type         string     `json:"type" validate:"required"`
	TargetRole   string     `json:"target_role" validate:"required"`
	TargetUserID *uuid.UUID `json:"target_user_id,omitempty"`
	ActionURL    *string    `json:"action_url,omitempty"`
}

// ============================================================
// SSE (Server-Sent Events) SYSTEM
// ============================================================

// SSEEvent is the payload broadcast to all connected SSE clients.
// EventType can be "vote_update" or "comment_update".
type SSEEvent struct {
	EventType    string    `json:"event"`
	ReportID     uuid.UUID `json:"report_id"`
	VoteCount    int       `json:"vote_count,omitempty"`
	CommentCount int       `json:"comment_count,omitempty"`
}
