package repository

import (
	"sovraequitara-be/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	// Auth
	CheckEmailExists(email string) (bool, error)
	CreateProfile(profile *model.Profile) error
	GetProfileByEmail(email string) (*model.Profile, error)
	UpdatePasswordByEmail(email, passwordHash string) error

	// Profile
	GetProfileByID(id uuid.UUID) (*model.Profile, error)
	UpdateProfile(id uuid.UUID, req model.UpdateProfileRequest) error
	UpdateAvatar(id uuid.UUID, avatarURL string) error

	// Categories
	GetCategories() ([]model.Category, error)

	// Reports
	CreateReport(report *model.Report) error
	GetReportsByProfileID(profileID uuid.UUID) ([]model.Report, error)
	GetAllReports(statusFilter, sortBy string) ([]model.Report, error)
	GetPublicReports(sortBy string) ([]model.Report, error)
	VerifyReport(reportID uuid.UUID) error
	ResolveReport(reportID uuid.UUID) error
	CancelReport(reportID uuid.UUID) error
	GetReportStats(profileID uuid.UUID) (*model.ReportStats, error)
	DeleteReport(reportID, profileID uuid.UUID) error

	// Admin Management
	GetAdmins() ([]model.Profile, error)
	CreateAdmin(profile *model.Profile) error
	UpdateAdmin(id uuid.UUID, fullName string, passwordHash string) error
	DeleteAdmin(id uuid.UUID) error

	// Comments
	CreateComment(comment *model.Comment) error
	GetCommentsByReportID(reportID uuid.UUID) ([]model.Comment, error)

	// Votes
	VoteReport(userID, reportID uuid.UUID, voteType int) error

	// Leaderboard
	GetLeaderboard() ([]model.Profile, error)

	// OTP (Registration)
	SaveOTP(email, code, name, passwordHash string) error
	VerifyOTP(email, code string) (name, passwordHash string, err error)
	DeleteOTP(email string) error

	// Forgot Password OTP
	SaveForgotPasswordOTP(email, code string) error
	VerifyForgotPasswordOTP(email, code string) error
	DeleteForgotPasswordOTP(email string) error

	// Chat
	GetOrCreateConversation(participantID uuid.UUID) (*model.Conversation, error)
	SendMessage(msg *model.Message) error
	UpdateConversationLastMessage(conversationID uuid.UUID, content string, incrementUnread bool) error
	GetMessagesByConversationID(conversationID uuid.UUID) ([]model.Message, error)
	GetAllConversations(roleFilter string) ([]model.Conversation, error)
	MarkConversationAsRead(conversationID uuid.UUID) error

	// Profile Listing
	GetAllProfiles(search string, roleFilter string) ([]model.Profile, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// ============================================================
// AUTH & PROFILE
// ============================================================

func (r *repository) CheckEmailExists(email string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Profile{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (r *repository) CreateProfile(profile *model.Profile) error {
	return r.db.Create(profile).Error
}

func (r *repository) GetProfileByEmail(email string) (*model.Profile, error) {
	var profile model.Profile
	err := r.db.Where("email = ?", email).First(&profile).Error
	return &profile, err
}

func (r *repository) GetProfileByID(id uuid.UUID) (*model.Profile, error) {
	var profile model.Profile
	err := r.db.First(&profile, "id = ?", id).Error
	return &profile, err
}

func (r *repository) UpdateProfile(id uuid.UUID, req model.UpdateProfileRequest) error {
	return r.db.Model(&model.Profile{}).Where("id = ?", id).Updates(map[string]interface{}{
		"full_name": req.FullName,
		"phone":     req.Phone,
	}).Error
}

func (r *repository) UpdateAvatar(id uuid.UUID, avatarURL string) error {
	return r.db.Model(&model.Profile{}).Where("id = ?", id).Update("avatar_url", avatarURL).Error
}

func (r *repository) UpdatePasswordByEmail(email, passwordHash string) error {
	return r.db.Model(&model.Profile{}).Where("email = ?", email).Update("password_hash", passwordHash).Error
}

// ============================================================
// CATEGORIES
// ============================================================

func (r *repository) GetCategories() ([]model.Category, error) {
	var categories []model.Category
	err := r.db.Find(&categories).Error
	return categories, err
}

// ============================================================
// REPORTS
// ============================================================

func (r *repository) CreateReport(report *model.Report) error {
	query := `INSERT INTO reports (profile_id, category_id, image_url, description, phone_number, latitude, longitude, location_detail) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, created_at, updated_at`
	return r.db.Raw(query, report.ProfileID, report.CategoryID, report.ImageURL, report.Description, report.PhoneNumber, report.Latitude, report.Longitude, report.LocationDetail).Scan(report).Error
}

func (r *repository) GetReportsByProfileID(profileID uuid.UUID) ([]model.Report, error) {
	var reports []model.Report
	query := `SELECT id, profile_id, category_id, image_url, description, phone_number, 
			  latitude, longitude, location_detail, vote_count, comment_count,
			  status, created_at, updated_at 
			  FROM reports WHERE profile_id = $1 ORDER BY created_at DESC`
	err := r.db.Raw(query, profileID).Scan(&reports).Error
	return reports, err
}

func (r *repository) GetAllReports(statusFilter, sortBy string) ([]model.Report, error) {
	var reports []model.Report
	
	orderClause := "created_at DESC"
	switch sortBy {
	case "votes":
		orderClause = "vote_count DESC, created_at DESC"
	case "comments":
		orderClause = "comment_count DESC, created_at DESC"
	case "category":
		orderClause = "category_id ASC, created_at DESC"
	}

	if statusFilter != "" {
		query := `SELECT id, profile_id, category_id, image_url, description, phone_number, 
				  latitude, longitude, location_detail, vote_count, comment_count,
				  status, created_at, updated_at 
				  FROM reports WHERE status = $1 ORDER BY ` + orderClause
		err := r.db.Raw(query, statusFilter).Scan(&reports).Error
		return reports, err
	}
	query := `SELECT id, profile_id, category_id, image_url, description, phone_number, 
			  latitude, longitude, location_detail, vote_count, comment_count,
			  status, created_at, updated_at 
			  FROM reports ORDER BY ` + orderClause
	err := r.db.Raw(query).Scan(&reports).Error
	return reports, err
}

func (r *repository) GetPublicReports(sortBy string) ([]model.Report, error) {
	var reports []model.Report
	
	orderClause := "created_at DESC"
	switch sortBy {
	case "votes":
		orderClause = "vote_count DESC, created_at DESC"
	case "comments":
		orderClause = "comment_count DESC, created_at DESC"
	case "category":
		orderClause = "category_id ASC, created_at DESC"
	}

	query := `SELECT id, profile_id, category_id, image_url, description, phone_number, 
			  latitude, longitude, location_detail, vote_count, comment_count,
			  status, created_at, updated_at 
			  FROM reports WHERE status IN ('VALID', 'RESOLVED') ORDER BY ` + orderClause
	err := r.db.Raw(query).Scan(&reports).Error
	return reports, err
}

func (r *repository) VerifyReport(reportID uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var report model.Report
		if err := tx.First(&report, "id = ?", reportID).Error; err != nil {
			return err
		}

		if report.Status != "PENDING" {
			return gorm.ErrInvalidData
		}

		if err := tx.Model(&report).Update("status", "VALID").Error; err != nil {
			return err
		}

		if err := tx.Model(&model.Profile{}).Where("id = ?", report.ProfileID).Update("points", gorm.Expr("points + ?", 10)).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *repository) ResolveReport(reportID uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var report model.Report
		if err := tx.First(&report, "id = ?", reportID).Error; err != nil {
			return err
		}

		if report.Status != "VALID" {
			return gorm.ErrInvalidData
		}

		if err := tx.Model(&report).Update("status", "RESOLVED").Error; err != nil {
			return err
		}

		if err := tx.Model(&model.Profile{}).Where("id = ?", report.ProfileID).Update("points", gorm.Expr("points + ?", 50)).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *repository) CancelReport(reportID uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var report model.Report
		if err := tx.First(&report, "id = ?", reportID).Error; err != nil {
			return err
		}

		if report.Status == "PENDING" {
			return gorm.ErrInvalidData
		}

		var pointDiff int
		if report.Status == "VALID" {
			pointDiff = -10
		} else if report.Status == "RESOLVED" {
			pointDiff = -60 // 10 for VALID + 50 for RESOLVED
		}

		if err := tx.Model(&report).Update("status", "PENDING").Error; err != nil {
			return err
		}

		if err := tx.Model(&model.Profile{}).Where("id = ?", report.ProfileID).Update("points", gorm.Expr("points + ?", pointDiff)).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *repository) GetReportStats(profileID uuid.UUID) (*model.ReportStats, error) {
	var stats model.ReportStats
	err := r.db.Model(&model.Report{}).Where("profile_id = ?", profileID).Count(&stats.Total).Error
	if err != nil { return nil, err }
	
	err = r.db.Model(&model.Report{}).Where("profile_id = ? AND status = ?", profileID, "PENDING").Count(&stats.Pending).Error
	if err != nil { return nil, err }
	
	err = r.db.Model(&model.Report{}).Where("profile_id = ? AND status = ?", profileID, "RESOLVED").Count(&stats.Resolved).Error
	return &stats, err
}

func (r *repository) DeleteReport(reportID, profileID uuid.UUID) error {
	res := r.db.Where("id = ? AND profile_id = ?", reportID, profileID).Delete(&model.Report{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ============================================================
// COMMENTS
// ============================================================

func (r *repository) CreateComment(comment *model.Comment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Insert comment
		query := `INSERT INTO comments (report_id, user_id, content) VALUES ($1, $2, $3) RETURNING id, created_at`
		if err := tx.Raw(query, comment.ReportID, comment.UserID, comment.Content).Scan(comment).Error; err != nil {
			return err
		}

		// Update comment count on report
		if err := tx.Model(&model.Report{}).Where("id = ?", comment.ReportID).UpdateColumn("comment_count", gorm.Expr("comment_count + ?", 1)).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *repository) GetCommentsByReportID(reportID uuid.UUID) ([]model.Comment, error) {
	var comments []model.Comment
	// We want to include user info ideally, but for now just the comments
	err := r.db.Preload("User").Where("report_id = ?", reportID).Order("created_at ASC").Find(&comments).Error
	return comments, err
}

// ============================================================
// VOTES
// ============================================================

func (r *repository) VoteReport(userID, reportID uuid.UUID, voteType int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var existingVote model.Vote
		err := tx.Where("user_id = ? AND report_id = ?", userID, reportID).First(&existingVote).Error

		voteDiff := 0
		if err == gorm.ErrRecordNotFound {
			// New vote
			newVote := model.Vote{UserID: userID, ReportID: reportID, VoteType: voteType}
			if err := tx.Create(&newVote).Error; err != nil {
				return err
			}
			voteDiff = voteType
		} else if err == nil {
			// Existing vote
			if existingVote.VoteType == voteType {
				// Remove vote
				if err := tx.Delete(&existingVote).Error; err != nil {
					return err
				}
				voteDiff = -voteType
			} else {
				// Change vote (e.g., from -1 to 1 means +2)
				if err := tx.Model(&existingVote).Update("vote_type", voteType).Error; err != nil {
					return err
				}
				voteDiff = voteType * 2
			}
		} else {
			return err
		}

		// Update report vote_count
		if err := tx.Model(&model.Report{}).Where("id = ?", reportID).UpdateColumn("vote_count", gorm.Expr("vote_count + ?", voteDiff)).Error; err != nil {
			return err
		}

		return nil
	})
}

// ============================================================
// LEADERBOARD
// ============================================================

func (r *repository) GetLeaderboard() ([]model.Profile, error) {
	var profiles []model.Profile
	err := r.db.Where("role = ? AND points > 0", "USER").Order("points desc").Limit(10).Find(&profiles).Error
	return profiles, err
}

// ============================================================
// OTP (Registration Handoff)
// ============================================================

func (r *repository) SaveOTP(email, code, name, passwordHash string) error {
	query := `INSERT INTO otps (email, code, name, password_hash, created_at) 
	          VALUES ($1, $2, $3, $4, NOW()) 
	          ON CONFLICT (email) DO UPDATE SET code = EXCLUDED.code, name = EXCLUDED.name, password_hash = EXCLUDED.password_hash, created_at = NOW()`
	return r.db.Exec(query, email, code, name, passwordHash).Error
}

func (r *repository) VerifyOTP(email, code string) (string, string, error) {
	var res struct {
		Name         string `gorm:"column:name"`
		PasswordHash string `gorm:"column:password_hash"`
	}
	err := r.db.Raw("SELECT name, password_hash FROM otps WHERE email = $1 AND code = $2 AND created_at > NOW() - INTERVAL '10 minutes'", email, code).Scan(&res).Error
	if err != nil {
		return "", "", err
	}
	if res.Name == "" {
		return "", "", gorm.ErrRecordNotFound
	}
	return res.Name, res.PasswordHash, nil
}

func (r *repository) DeleteOTP(email string) error {
	return r.db.Exec("DELETE FROM otps WHERE email = $1", email).Error
}

// ============================================================
// FORGOT PASSWORD OTP
// ============================================================

func (r *repository) SaveForgotPasswordOTP(email, code string) error {
	query := `INSERT INTO forgot_password_otps (email, code, created_at) 
	          VALUES ($1, $2, NOW()) 
	          ON CONFLICT (email) DO UPDATE SET code = EXCLUDED.code, created_at = NOW()`
	return r.db.Exec(query, email, code).Error
}

func (r *repository) VerifyForgotPasswordOTP(email, code string) error {
	var count int64
	err := r.db.Table("forgot_password_otps").Where("email = ? AND code = ? AND created_at > NOW() - INTERVAL '10 minutes'", email, code).Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *repository) DeleteForgotPasswordOTP(email string) error {
	return r.db.Exec("DELETE FROM forgot_password_otps WHERE email = $1", email).Error
}

// ============================================================
// ADMIN MANAGEMENT (For Super Admin)
// ============================================================

func (r *repository) GetAdmins() ([]model.Profile, error) {
	var admins []model.Profile
	err := r.db.Where("role = ?", "admin").Find(&admins).Error
	return admins, err
}

func (r *repository) CreateAdmin(profile *model.Profile) error {
	return r.db.Create(profile).Error
}

func (r *repository) UpdateAdmin(id uuid.UUID, fullName string, passwordHash string) error {
	updates := map[string]interface{}{"full_name": fullName}
	if passwordHash != "" {
		updates["password_hash"] = passwordHash
	}
	return r.db.Model(&model.Profile{}).Where("id = ?", id).Updates(updates).Error
}

func (r *repository) DeleteAdmin(id uuid.UUID) error {
	return r.db.Where("id = ? AND role = ?", id, "admin").Delete(&model.Profile{}).Error
}

// ============================================================
// CHAT SYSTEM
// ============================================================

func (r *repository) GetOrCreateConversation(participantID uuid.UUID) (*model.Conversation, error) {
	var conv model.Conversation
	err := r.db.Where("participant_id = ?", participantID).First(&conv).Error
	if err == gorm.ErrRecordNotFound {
		conv = model.Conversation{ParticipantID: participantID}
		if err := r.db.Create(&conv).Error; err != nil {
			return nil, err
		}
		return &conv, nil
	}
	return &conv, err
}

func (r *repository) SendMessage(msg *model.Message) error {
	return r.db.Create(msg).Error
}

func (r *repository) UpdateConversationLastMessage(conversationID uuid.UUID, content string, incrementUnread bool) error {
	updates := map[string]interface{}{
		"last_message":    content,
		"last_message_at": gorm.Expr("NOW()"),
	}
	if incrementUnread {
		updates["unread_count"] = gorm.Expr("unread_count + 1")
	}
	return r.db.Model(&model.Conversation{}).Where("id = ?", conversationID).Updates(updates).Error
}

func (r *repository) GetMessagesByConversationID(conversationID uuid.UUID) ([]model.Message, error) {
	var messages []model.Message
	err := r.db.Preload("Sender").Where("conversation_id = ?", conversationID).Order("created_at ASC").Find(&messages).Error
	return messages, err
}

func (r *repository) GetAllConversations(roleFilter string) ([]model.Conversation, error) {
	var convs []model.Conversation
	query := r.db.Preload("Participant").Order("last_message_at DESC")
	if roleFilter != "" {
		query = query.Joins("JOIN profiles ON profiles.id = conversations.participant_id").Where("profiles.role = ?", roleFilter)
	}
	err := query.Find(&convs).Error
	return convs, err
}

func (r *repository) MarkConversationAsRead(conversationID uuid.UUID) error {
	err := r.db.Model(&model.Message{}).Where("conversation_id = ? AND is_read = false", conversationID).Update("is_read", true).Error
	if err != nil {
		return err
	}
	return r.db.Model(&model.Conversation{}).Where("id = ?", conversationID).Update("unread_count", 0).Error
}

// ============================================================
// PROFILE LISTING
// ============================================================

func (r *repository) GetAllProfiles(search string, roleFilter string) ([]model.Profile, error) {
	var profiles []model.Profile
	query := r.db.Select("id, email, full_name, phone, avatar_url, points, role, created_at").Order("created_at DESC")
	
	if search != "" {
		query = query.Where("full_name ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if roleFilter != "" {
		query = query.Where("role = ?", roleFilter)
	}

	err := query.Find(&profiles).Error
	return profiles, err
}
