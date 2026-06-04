package repository

import (
	"fmt"
	"sovraequitara-be/internal/model"
	"strings"
	"time"

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
	UpdateProfilePassword(id uuid.UUID, passwordHash string) error

	// Categories
	GetCategories() ([]model.Category, error)

	// Reports
	CreateReport(report *model.Report) error
	GetReportByID(id uuid.UUID) (model.Report, error)
	GetReportsByProfileID(profileID uuid.UUID) ([]model.Report, error)
	GetAllReports(statusFilter, sortBy string) ([]model.Report, error)
	GetPublicReports(sortBy string) ([]model.Report, error)
	VerifyReport(reportID uuid.UUID) error
	ResolveReport(reportID uuid.UUID) error
	ApproveResolution(reportID, profileID uuid.UUID) error
	RejectResolution(reportID, profileID uuid.UUID) error
	CancelReport(reportID uuid.UUID) error
	GetReportStats(profileID uuid.UUID) (*model.ReportStats, error)
	GetSystemStats() (*model.SystemStats, error)
	SearchReports(keyword string) ([]model.Report, error)
	DeleteReport(reportID, profileID uuid.UUID) error

	// Saved Reports
	GetSavedReports(adminID uuid.UUID) ([]model.Report, error)
	ToggleSaveReport(adminID, reportID uuid.UUID) (bool, error)

	// Admin Management
	GetAdmins() ([]model.Profile, error)
	CreateAdmin(profile *model.Profile) error
	UpdateAdmin(id uuid.UUID, fullName string, passwordHash string) error
	DeleteAdmin(id uuid.UUID) error

	// Comments
	CreateComment(comment *model.Comment) error
	GetCommentsByReportID(reportID uuid.UUID) ([]model.Comment, error)
	GetCommentCount(reportID uuid.UUID) (int64, error)

	// Votes
	VoteReport(userID, reportID uuid.UUID, voteType int) error
	GetVoteCount(reportID uuid.UUID) (int64, error)
	GetUserVoteForReport(userID, reportID uuid.UUID) (int, error)

	// Leaderboard
	GetLeaderboard() ([]model.Profile, error)

	// OTP (Registration)
	SaveOTP(email, code, name, passwordHash string) error
	VerifyOTP(email, code string) (name, passwordHash string, err error)
	DeleteOTP(email string) error
	ResendOTP(email, code string) error

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

	// Notifications
	CreateNotification(notif *model.Notification) error
	GetNotifications(userID uuid.UUID, role string) ([]model.Notification, error)
	UpdateNotification(id uuid.UUID, title, message, notifType, targetRole string) error
	DeleteNotification(id uuid.UUID) error

	// Database Backup
	BackupDatabase() (string, error)

	// AI Chat History
	CreateAIThread(userID uuid.UUID, title string) (*model.AIThread, error)
	GetAIThreadsByUserID(userID uuid.UUID) ([]model.AIThread, error)
	GetAIThreadByID(threadID uuid.UUID) (*model.AIThread, error)
	DeleteAIThread(threadID uuid.UUID, userID uuid.UUID) error
	AddAIMessage(threadID uuid.UUID, role string, content string) error
	GetAIMessagesByThreadID(threadID uuid.UUID) ([]model.AIMessage, error)
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

func (r *repository) UpdateProfilePassword(id uuid.UUID, passwordHash string) error {
	tx := r.db.Model(&model.Profile{}).Where("id = ?", id).Update("password_hash", passwordHash)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *repository) UpdatePasswordByEmail(email, passwordHash string) error {
	tx := r.db.Model(&model.Profile{}).Where("email = ?", email).Update("password_hash", passwordHash)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
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
	query := `INSERT INTO reports (profile_id, category_id, image_urls, description, phone_number, latitude, longitude, location_detail)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, created_at, updated_at`
	return r.db.Raw(query, report.ProfileID, report.CategoryID, report.ImageURLs, report.Description, report.PhoneNumber, report.Latitude, report.Longitude, report.LocationDetail).Scan(report).Error
}

func (r *repository) GetReportByID(id uuid.UUID) (model.Report, error) {
	var report model.Report
	err := r.db.Preload("Profile").Preload("Category").First(&report, "id = ?", id).Error
	return report, err
}

func (r *repository) GetReportsByProfileID(profileID uuid.UUID) ([]model.Report, error) {
	var reports []model.Report
	err := r.db.Preload("Profile").Preload("Category").Where("profile_id = ?", profileID).Order("created_at DESC").Find(&reports).Error
	return reports, err
}

func (r *repository) GetAllReports(statusFilter, sortBy string) ([]model.Report, error) {
	var reports []model.Report
	
	orderClause := "CASE WHEN status = 'PENDING' THEN 0 ELSE 1 END ASC, created_at DESC"
	switch sortBy {
	case "votes":
		orderClause = "CASE WHEN status = 'PENDING' THEN 0 ELSE 1 END ASC, vote_count DESC, created_at DESC"
	case "comments":
		orderClause = "CASE WHEN status = 'PENDING' THEN 0 ELSE 1 END ASC, comment_count DESC, created_at DESC"
	case "category":
		orderClause = "CASE WHEN status = 'PENDING' THEN 0 ELSE 1 END ASC, category_id ASC, created_at DESC"
	}

	query := r.db.Preload("Profile").Preload("Category").Order(orderClause)
	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}
	err := query.Find(&reports).Error
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

	err := r.db.Preload("Profile").Preload("Category").Where("status IN ('PENDING', 'VALID', 'WAITING_APPROVAL', 'RESOLVED')").Order(orderClause).Find(&reports).Error
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

		actionURL := "/history?open=" + reportID.String()
		notif := model.Notification{
			Title:        "Laporan Diverifikasi",
			Message:      "Laporan Anda telah diverifikasi oleh petugas dan berstatus VALID.",
			Type:         "INFO",
			TargetRole:   "SPECIFIC_USER",
			TargetUserID: &report.ProfileID,
			ActionURL:    &actionURL,
		}
		if err := tx.Create(&notif).Error; err != nil {
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

		if err := tx.Model(&report).Update("status", "WAITING_APPROVAL").Error; err != nil {
			return err
		}

		actionURL := "/history?open=" + reportID.String()
		notif := model.Notification{
			Title:        "Laporan Ditangani",
			Message:      "Laporan Anda telah ditangani dan menunggu konfirmasi/persetujuan Anda.",
			Type:         "INFO",
			TargetRole:   "SPECIFIC_USER",
			TargetUserID: &report.ProfileID,
			ActionURL:    &actionURL,
		}
		if err := tx.Create(&notif).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *repository) ApproveResolution(reportID, profileID uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var report model.Report
		if err := tx.Where("id = ? AND profile_id = ?", reportID, profileID).First(&report).Error; err != nil {
			return err
		}

		if report.Status != "WAITING_APPROVAL" {
			return gorm.ErrInvalidData
		}

		if err := tx.Model(&report).Update("status", "RESOLVED").Error; err != nil {
			return err
		}

		if err := tx.Model(&model.Profile{}).Where("id = ?", report.ProfileID).Update("points", gorm.Expr("points + ?", 50)).Error; err != nil {
			return err
		}

		actionURL := "/history?open=" + reportID.String()
		notif := model.Notification{
			Title:        "Penyelesaian Laporan Disetujui",
			Message:      "Penyelesaian laporan Anda telah disetujui. Poin bonus keaktifan Anda bertambah +50!",
			Type:         "INFO",
			TargetRole:   "SPECIFIC_USER",
			TargetUserID: &report.ProfileID,
			ActionURL:    &actionURL,
		}
		if err := tx.Create(&notif).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *repository) RejectResolution(reportID, profileID uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var report model.Report
		if err := tx.Where("id = ? AND profile_id = ?", reportID, profileID).First(&report).Error; err != nil {
			return err
		}

		if report.Status != "WAITING_APPROVAL" {
			return gorm.ErrInvalidData
		}

		if err := tx.Model(&report).Update("status", "VALID").Error; err != nil {
			return err
		}

		actionURL := "/history?open=" + reportID.String()
		notif := model.Notification{
			Title:        "Penyelesaian Laporan Ditolak",
			Message:      "Anda telah menolak penyelesaian laporan. Status laporan dikembalikan ke Diproses (VALID).",
			Type:         "WARNING",
			TargetRole:   "SPECIFIC_USER",
			TargetUserID: &report.ProfileID,
			ActionURL:    &actionURL,
		}
		if err := tx.Create(&notif).Error; err != nil {
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
		if report.Status == "VALID" || report.Status == "WAITING_APPROVAL" {
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

		actionURL := "/history?open=" + reportID.String()
		notif := model.Notification{
			Title:        "Verifikasi Laporan Dibatalkan",
			Message:      "Verifikasi aduan Anda dibatalkan oleh Super Admin kembali ke status PENDING. Poin keaktifan disesuaikan.",
			Type:         "WARNING",
			TargetRole:   "SPECIFIC_USER",
			TargetUserID: &report.ProfileID,
			ActionURL:    &actionURL,
		}
		if err := tx.Create(&notif).Error; err != nil {
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

func (r *repository) GetSystemStats() (*model.SystemStats, error) {
	var stats model.SystemStats
	if err := r.db.Model(&model.Report{}).Count(&stats.TotalReports).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&model.Report{}).Where("status = ?", "PENDING").Count(&stats.PendingReports).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&model.Report{}).Where("status = ?", "VALID").Count(&stats.ValidReports).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&model.Report{}).Where("status = ?", "RESOLVED").Count(&stats.ResolvedReports).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&model.Profile{}).Where("role = ?", "USER").Count(&stats.TotalUsers).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&model.Profile{}).Where("role IN ('admin', 'super_admin')").Count(&stats.TotalAdmins).Error; err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *repository) SearchReports(keyword string) ([]model.Report, error) {
	var reports []model.Report
	err := r.db.Preload("Profile").Preload("Category").
		Where("description ILIKE ? OR location_detail ILIKE ?", "%"+keyword+"%", "%"+keyword+"%").
		Order("created_at DESC").
		Find(&reports).Error
	return reports, err
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
// SAVED REPORTS
// ============================================================

func (r *repository) ToggleSaveReport(adminID, reportID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&model.SavedReport{}).Where("admin_id = ? AND report_id = ?", adminID, reportID).Count(&count).Error
	if err != nil {
		return false, err
	}

	if count > 0 {
		// Unsave
		err = r.db.Where("admin_id = ? AND report_id = ?", adminID, reportID).Delete(&model.SavedReport{}).Error
		return false, err
	}

	// Save
	err = r.db.Create(&model.SavedReport{
		AdminID:  adminID,
		ReportID: reportID,
	}).Error
	return true, err
}

func (r *repository) GetSavedReports(adminID uuid.UUID) ([]model.Report, error) {
	var reports []model.Report
	err := r.db.Preload("Profile").Preload("Category").
		Joins("JOIN saved_reports sr ON reports.id = sr.report_id").
		Where("sr.admin_id = ?", adminID).
		Order("sr.created_at DESC").
		Find(&reports).Error
	return reports, err
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

		// Recalculate comment_count from COUNT(*) — guarantees accuracy vs cached +1
		if err := tx.Model(&model.Report{}).Where("id = ?", comment.ReportID).
			Update("comment_count", tx.Model(&model.Comment{}).Where("report_id = ?", comment.ReportID).Select("count(*)")).Error; err != nil {
			// Fallback: increment if subquery fails
			_ = tx.Model(&model.Report{}).Where("id = ?", comment.ReportID).UpdateColumn("comment_count", gorm.Expr("comment_count + ?", 1)).Error
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

// GetCommentCount returns the exact number of comments for a report from the DB.
func (r *repository) GetCommentCount(reportID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&model.Comment{}).Where("report_id = ?", reportID).Count(&count).Error
	return count, err
}

// ============================================================
// VOTES
// ============================================================

func (r *repository) VoteReport(userID, reportID uuid.UUID, voteType int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var existingVote model.Vote
		err := tx.Where("user_id = ? AND report_id = ?", userID, reportID).First(&existingVote).Error

		if err == gorm.ErrRecordNotFound {
			// New vote
			newVote := model.Vote{UserID: userID, ReportID: reportID, VoteType: voteType}
			if err := tx.Create(&newVote).Error; err != nil {
				return err
			}
		} else if err == nil {
			// Existing vote
			if existingVote.VoteType == voteType {
				// Remove vote (toggle off)
				if err := tx.Delete(&existingVote).Error; err != nil {
					return err
				}
			} else {
				// Change vote direction
				if err := tx.Model(&existingVote).Update("vote_type", voteType).Error; err != nil {
					return err
				}
			}
		} else {
			return err
		}

		// Recalculate vote_count from actual rows (upvotes - downvotes) — prevents drift
		var upvotes, downvotes int64
		if err := tx.Model(&model.Vote{}).Where("report_id = ? AND vote_type = 1", reportID).Count(&upvotes).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Vote{}).Where("report_id = ? AND vote_type = -1", reportID).Count(&downvotes).Error; err != nil {
			return err
		}
		voteCount := int(upvotes - downvotes)

		if err := tx.Model(&model.Report{}).Where("id = ?", reportID).Update("vote_count", voteCount).Error; err != nil {
			return err
		}

		return nil
	})
}

// GetVoteCount returns the net vote count (upvotes - downvotes) for a report.
func (r *repository) GetVoteCount(reportID uuid.UUID) (int64, error) {
	var upvotes, downvotes int64
	if err := r.db.Model(&model.Vote{}).Where("report_id = ? AND vote_type = 1", reportID).Count(&upvotes).Error; err != nil {
		return 0, err
	}
	if err := r.db.Model(&model.Vote{}).Where("report_id = ? AND vote_type = -1", reportID).Count(&downvotes).Error; err != nil {
		return 0, err
	}
	return upvotes - downvotes, nil
}

// GetUserVoteForReport returns the user's vote_type for a report (1, -1, or 0 if not voted).
func (r *repository) GetUserVoteForReport(userID, reportID uuid.UUID) (int, error) {
	var vote model.Vote
	err := r.db.Where("user_id = ? AND report_id = ?", userID, reportID).First(&vote).Error
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return vote.VoteType, nil
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
	// Check if currently blocked
	var blockedUntil *time.Time
	err := r.db.Raw("SELECT blocked_until FROM otps WHERE email = $1", email).Scan(&blockedUntil).Error
	if err == nil && blockedUntil != nil && blockedUntil.After(time.Now()) {
		return fmt.Errorf("BLOCKED:%d", int(time.Until(*blockedUntil).Seconds()))
	}

	query := `INSERT INTO otps (email, code, name, password_hash, failed_attempts, blocked_until, created_at) 
	          VALUES ($1, $2, $3, $4, 0, NULL, NOW()) 
	          ON CONFLICT (email) DO UPDATE SET code = EXCLUDED.code, name = EXCLUDED.name, password_hash = EXCLUDED.password_hash, failed_attempts = 0, blocked_until = NULL, created_at = NOW()`
	return r.db.Exec(query, email, code, name, passwordHash).Error
}

func (r *repository) VerifyOTP(email, code string) (string, string, error) {
	// 1. Fetch OTP record
	var otp struct {
		Code           *string    `gorm:"column:code"`
		Name           string     `gorm:"column:name"`
		PasswordHash   string     `gorm:"column:password_hash"`
		FailedAttempts int        `gorm:"column:failed_attempts"`
		BlockedUntil   *time.Time `gorm:"column:blocked_until"`
		CreatedAt      time.Time  `gorm:"column:created_at"`
	}
	err := r.db.Raw("SELECT code, name, password_hash, failed_attempts, blocked_until, created_at FROM otps WHERE email = $1", email).Scan(&otp).Error
	if err != nil {
		return "", "", err
	}
	if otp.Name == "" {
		return "", "", gorm.ErrRecordNotFound
	}

	// 2. Check if currently blocked
	if otp.BlockedUntil != nil && otp.BlockedUntil.After(time.Now()) {
		return "", "", fmt.Errorf("BLOCKED")
	}

	// 3. Check if OTP is expired (10 minutes)
	if time.Since(otp.CreatedAt) > 10*time.Minute {
		return "", "", fmt.Errorf("EXPIRED")
	}

	// 4. Check if code matches (and is not null)
	if otp.Code == nil || *otp.Code == "" || *otp.Code != code {
		newFailed := otp.FailedAttempts + 1
		if newFailed >= 4 {
			// Lockout! Block for 1 minute and clear code
			r.db.Exec("UPDATE otps SET failed_attempts = $1, blocked_until = NOW() + INTERVAL '1 minute', code = NULL WHERE email = $2", newFailed, email)
			return "", "", fmt.Errorf("LOCKOUT")
		} else {
			r.db.Exec("UPDATE otps SET failed_attempts = $1 WHERE email = $2", newFailed, email)
			return "", "", fmt.Errorf("WRONG_OTP:%d", 4-newFailed)
		}
	}

	return otp.Name, otp.PasswordHash, nil
}

func (r *repository) DeleteOTP(email string) error {
	return r.db.Exec("DELETE FROM otps WHERE email = $1", email).Error
}

func (r *repository) ResendOTP(email, code string) error {
	// Check if currently blocked
	var blockedUntil *time.Time
	err := r.db.Raw("SELECT blocked_until FROM otps WHERE email = $1", email).Scan(&blockedUntil).Error
	if err == nil && blockedUntil != nil && blockedUntil.After(time.Now()) {
		return fmt.Errorf("BLOCKED:%d", int(time.Until(*blockedUntil).Seconds()))
	}

	// First verify that the row actually exists
	var exists int64
	r.db.Table("otps").Where("email = ?", email).Count(&exists)
	if exists == 0 {
		return gorm.ErrRecordNotFound
	}

	// Update code and reset failed attempts
	err = r.db.Exec("UPDATE otps SET code = $1, failed_attempts = 0, blocked_until = NULL, created_at = NOW() WHERE email = $2", code, email).Error
	return err
}

// ============================================================
// FORGOT PASSWORD OTP
// ============================================================

func (r *repository) SaveForgotPasswordOTP(email, code string) error {
	// Check if currently blocked
	var blockedUntil *time.Time
	err := r.db.Raw("SELECT blocked_until FROM forgot_password_otps WHERE email = $1", email).Scan(&blockedUntil).Error
	if err == nil && blockedUntil != nil && blockedUntil.After(time.Now()) {
		return fmt.Errorf("BLOCKED:%d", int(time.Until(*blockedUntil).Seconds()))
	}

	// Auto Housekeeping: Delete expired OTPs older than 10 minutes (only if not blocked)
	_ = r.db.Exec("DELETE FROM forgot_password_otps WHERE created_at < NOW() - INTERVAL '10 minutes' AND (blocked_until IS NULL OR blocked_until < NOW())")

	query := `INSERT INTO forgot_password_otps (email, code, failed_attempts, blocked_until, created_at) 
	          VALUES ($1, $2, 0, NULL, NOW()) 
	          ON CONFLICT (email) DO UPDATE SET code = EXCLUDED.code, failed_attempts = 0, blocked_until = NULL, created_at = NOW()`
	return r.db.Exec(query, email, code).Error
}

func (r *repository) VerifyForgotPasswordOTP(email, code string) error {
	// 1. Fetch OTP record
	var otp struct {
		Code           *string    `gorm:"column:code"`
		FailedAttempts int        `gorm:"column:failed_attempts"`
		BlockedUntil   *time.Time `gorm:"column:blocked_until"`
		CreatedAt      time.Time  `gorm:"column:created_at"`
	}
	err := r.db.Raw("SELECT code, failed_attempts, blocked_until, created_at FROM forgot_password_otps WHERE email = $1", email).Scan(&otp).Error
	if err != nil {
		return err
	}
	if otp.CreatedAt.IsZero() {
		return gorm.ErrRecordNotFound
	}

	// 2. Check if currently blocked
	if otp.BlockedUntil != nil && otp.BlockedUntil.After(time.Now()) {
		return fmt.Errorf("BLOCKED")
	}

	// 3. Check if OTP is expired (10 minutes)
	if time.Since(otp.CreatedAt) > 10*time.Minute {
		return fmt.Errorf("EXPIRED")
	}

	// 4. Check if code matches (and is not null)
	if otp.Code == nil || *otp.Code == "" || *otp.Code != code {
		newFailed := otp.FailedAttempts + 1
		if newFailed >= 4 {
			// Lockout! Block for 1 minute and clear code
			r.db.Exec("UPDATE forgot_password_otps SET failed_attempts = $1, blocked_until = NOW() + INTERVAL '1 minute', code = NULL WHERE email = $2", newFailed, email)
			return fmt.Errorf("LOCKOUT")
		} else {
			r.db.Exec("UPDATE forgot_password_otps SET failed_attempts = $1 WHERE email = $2", newFailed, email)
			return fmt.Errorf("WRONG_OTP:%d", 4-newFailed)
		}
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

// ============================================================
// NOTIFICATIONS
// ============================================================

func (r *repository) CreateNotification(notif *model.Notification) error {
	return r.db.Create(notif).Error
}

func (r *repository) GetNotifications(userID uuid.UUID, role string) ([]model.Notification, error) {
	var notifs []model.Notification
	query := r.db.Preload("Creator").Order("created_at DESC").Limit(50)
	
	if role == "super_admin" || role == "admin" || role == "SUPERADMIN" || role == "ADMIN" {
		err := query.Where("target_role != 'SPECIFIC_USER' OR target_user_id = ?", userID).Find(&notifs).Error
		return notifs, err
	}

	err := query.Where("target_role IN (?) OR (target_role = 'SPECIFIC_USER' AND target_user_id = ?)", []string{"ALL", role}, userID).Find(&notifs).Error
	return notifs, err
}

func (r *repository) UpdateNotification(id uuid.UUID, title, message, notifType, targetRole string) error {
	return r.db.Model(&model.Notification{}).Where("id = ?", id).Updates(map[string]interface{}{
		"title":       title,
		"message":     message,
		"type":        notifType,
		"target_role": targetRole,
	}).Error
}

func (r *repository) DeleteNotification(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&model.Notification{}).Error
}

func (r *repository) BackupDatabase() (string, error) {
	var sqlBuilder strings.Builder

	// Header to disable constraints and triggers temporarily during import
	sqlBuilder.WriteString("-- SovraEquitara Database Backup\n")
	sqlBuilder.WriteString(fmt.Sprintf("-- Generated at: %s\n\n", time.Now().Format(time.RFC3339)))
	sqlBuilder.WriteString("SET session_replication_role = 'replica';\n\n")

	escapeStr := func(s string) string {
		return strings.ReplaceAll(s, "'", "''")
	}

	formatStrPtr := func(s *string) string {
		if s == nil {
			return "NULL"
		}
		return fmt.Sprintf("'%s'", escapeStr(*s))
	}

	formatIntPtr := func(i *int) string {
		if i == nil {
			return "NULL"
		}
		return fmt.Sprintf("%d", *i)
	}

	formatUUIDPtr := func(u *uuid.UUID) string {
		if u == nil {
			return "NULL"
		}
		return fmt.Sprintf("'%s'", u.String())
	}

	formatStrArray := func(arr []string) string {
		if arr == nil {
			return "NULL"
		}
		var escaped []string
		for _, s := range arr {
			escaped = append(escaped, fmt.Sprintf("'%s'", escapeStr(s)))
		}
		return fmt.Sprintf("ARRAY[%s]::TEXT[]", strings.Join(escaped, ", "))
	}

	formatTime := func(t time.Time) string {
		return fmt.Sprintf("'%s'", t.Format("2006-01-02 15:04:05.000000-07"))
	}

	// 1. PROFILES
	var profiles []model.Profile
	if err := r.db.Find(&profiles).Error; err != nil {
		return "", err
	}
	if len(profiles) > 0 {
		sqlBuilder.WriteString("-- TRUNCATE profiles;\n")
		sqlBuilder.WriteString("INSERT INTO profiles (id, email, password_hash, full_name, phone, avatar_url, points, role, created_at, updated_at) VALUES\n")
		for i, p := range profiles {
			comma := ","
			if i == len(profiles)-1 {
				comma = ";"
			}
			sqlBuilder.WriteString(fmt.Sprintf("('%s', '%s', '%s', '%s', '%s', %s, %d, '%s', %s, %s)%s\n",
				p.ID.String(), escapeStr(p.Email), escapeStr(p.PasswordHash), escapeStr(p.FullName), escapeStr(p.Phone),
				formatStrPtr(p.AvatarURL), p.Points, escapeStr(p.Role), formatTime(p.CreatedAt), formatTime(p.UpdatedAt), comma))
		}
		sqlBuilder.WriteString("\n")
	}

	// 2. CATEGORIES
	var categories []model.Category
	if err := r.db.Find(&categories).Error; err != nil {
		return "", err
	}
	if len(categories) > 0 {
		sqlBuilder.WriteString("-- TRUNCATE categories;\n")
		sqlBuilder.WriteString("INSERT INTO categories (id, name, slug) VALUES\n")
		for i, c := range categories {
			comma := ","
			if i == len(categories)-1 {
				comma = ";"
			}
			sqlBuilder.WriteString(fmt.Sprintf("(%d, '%s', '%s')%s\n",
				c.ID, escapeStr(c.Name), escapeStr(c.Slug), comma))
		}
		sqlBuilder.WriteString("\n")
	}

	// 3. REPORTS
	var reports []model.Report
	if err := r.db.Find(&reports).Error; err != nil {
		return "", err
	}
	if len(reports) > 0 {
		sqlBuilder.WriteString("-- TRUNCATE reports;\n")
		sqlBuilder.WriteString("INSERT INTO reports (id, profile_id, category_id, image_urls, description, phone_number, latitude, longitude, location_detail, vote_count, comment_count, status, created_at, updated_at) VALUES\n")
		for i, rp := range reports {
			comma := ","
			if i == len(reports)-1 {
				comma = ";"
			}
			sqlBuilder.WriteString(fmt.Sprintf("('%s', '%s', %s, %s, '%s', %s, %f, %f, '%s', %d, %d, '%s', %s, %s)%s\n",
				rp.ID.String(), rp.ProfileID.String(), formatIntPtr(rp.CategoryID), formatStrArray(rp.ImageURLs),
				escapeStr(rp.Description), formatStrPtr(rp.PhoneNumber), rp.Latitude, rp.Longitude, escapeStr(rp.LocationDetail),
				rp.VoteCount, rp.CommentCount, escapeStr(rp.Status), formatTime(rp.CreatedAt), formatTime(rp.UpdatedAt), comma))
		}
		sqlBuilder.WriteString("\n")
	}

	// 4. COMMENTS
	var comments []model.Comment
	if err := r.db.Find(&comments).Error; err != nil {
		return "", err
	}
	if len(comments) > 0 {
		sqlBuilder.WriteString("-- TRUNCATE comments;\n")
		sqlBuilder.WriteString("INSERT INTO comments (id, report_id, user_id, content, created_at) VALUES\n")
		for i, c := range comments {
			comma := ","
			if i == len(comments)-1 {
				comma = ";"
			}
			sqlBuilder.WriteString(fmt.Sprintf("('%s', '%s', '%s', '%s', %s)%s\n",
				c.ID.String(), c.ReportID.String(), c.UserID.String(), escapeStr(c.Content), formatTime(c.CreatedAt), comma))
		}
		sqlBuilder.WriteString("\n")
	}

	// 5. VOTES
	var votes []model.Vote
	if err := r.db.Find(&votes).Error; err != nil {
		return "", err
	}
	if len(votes) > 0 {
		sqlBuilder.WriteString("-- TRUNCATE votes;\n")
		sqlBuilder.WriteString("INSERT INTO votes (user_id, report_id, vote_type) VALUES\n")
		for i, v := range votes {
			comma := ","
			if i == len(votes)-1 {
				comma = ";"
			}
			sqlBuilder.WriteString(fmt.Sprintf("('%s', '%s', %d)%s\n",
				v.UserID.String(), v.ReportID.String(), v.VoteType, comma))
		}
		sqlBuilder.WriteString("\n")
	}

	// 6. SAVED REPORTS
	var savedReports []model.SavedReport
	if err := r.db.Find(&savedReports).Error; err != nil {
		return "", err
	}
	if len(savedReports) > 0 {
		sqlBuilder.WriteString("-- TRUNCATE saved_reports;\n")
		sqlBuilder.WriteString("INSERT INTO saved_reports (admin_id, report_id, created_at) VALUES\n")
		for i, sr := range savedReports {
			comma := ","
			if i == len(savedReports)-1 {
				comma = ";"
			}
			sqlBuilder.WriteString(fmt.Sprintf("('%s', '%s', %s)%s\n",
				sr.AdminID.String(), sr.ReportID.String(), formatTime(sr.CreatedAt), comma))
		}
		sqlBuilder.WriteString("\n")
	}

	// 7. CONVERSATIONS
	var conversations []model.Conversation
	if err := r.db.Find(&conversations).Error; err != nil {
		return "", err
	}
	if len(conversations) > 0 {
		sqlBuilder.WriteString("-- TRUNCATE conversations;\n")
		sqlBuilder.WriteString("INSERT INTO conversations (id, participant_id, last_message, last_message_at, unread_count, created_at, updated_at) VALUES\n")
		for i, cv := range conversations {
			comma := ","
			if i == len(conversations)-1 {
				comma = ";"
			}
			sqlBuilder.WriteString(fmt.Sprintf("('%s', '%s', '%s', %s, %d, %s, %s)%s\n",
				cv.ID.String(), cv.ParticipantID.String(), escapeStr(cv.LastMessage), formatTime(cv.LastMessageAt),
				cv.UnreadCount, formatTime(cv.CreatedAt), formatTime(cv.UpdatedAt), comma))
		}
		sqlBuilder.WriteString("\n")
	}

	// 8. MESSAGES
	var messages []model.Message
	if err := r.db.Find(&messages).Error; err != nil {
		return "", err
	}
	if len(messages) > 0 {
		sqlBuilder.WriteString("-- TRUNCATE messages;\n")
		sqlBuilder.WriteString("INSERT INTO messages (id, conversation_id, sender_id, content, is_read, created_at) VALUES\n")
		for i, m := range messages {
			comma := ","
			if i == len(messages)-1 {
				comma = ";"
			}
			sqlBuilder.WriteString(fmt.Sprintf("('%s', '%s', '%s', '%s', %t, %s)%s\n",
				m.ID.String(), m.ConversationID.String(), m.SenderID.String(), escapeStr(m.Content), m.IsRead, formatTime(m.CreatedAt), comma))
		}
		sqlBuilder.WriteString("\n")
	}

	// 9. NOTIFICATIONS
	var notifications []model.Notification
	if err := r.db.Find(&notifications).Error; err != nil {
		return "", err
	}
	if len(notifications) > 0 {
		sqlBuilder.WriteString("-- TRUNCATE notifications;\n")
		sqlBuilder.WriteString("INSERT INTO notifications (id, title, message, type, target_role, target_user_id, action_url, created_by, created_at) VALUES\n")
		for i, n := range notifications {
			comma := ","
			if i == len(notifications)-1 {
				comma = ";"
			}
			sqlBuilder.WriteString(fmt.Sprintf("('%s', '%s', '%s', '%s', '%s', %s, %s, %s, %s)%s\n",
				n.ID.String(), escapeStr(n.Title), escapeStr(n.Message), escapeStr(n.Type), escapeStr(n.TargetRole),
				formatUUIDPtr(n.TargetUserID), formatStrPtr(n.ActionURL), formatUUIDPtr(n.CreatedBy), formatTime(n.CreatedAt), comma))
		}
		sqlBuilder.WriteString("\n")
	}

	// Re-enable constraints and triggers
	sqlBuilder.WriteString("SET session_replication_role = 'origin';\n")

	return sqlBuilder.String(), nil
}

// ============================================================
// AI CHAT HISTORY IMPLEMENTATIONS
// ============================================================

func (r *repository) CreateAIThread(userID uuid.UUID, title string) (*model.AIThread, error) {
	thread := &model.AIThread{
		UserID: userID,
		Title:  title,
	}
	err := r.db.Create(thread).Error
	return thread, err
}

func (r *repository) GetAIThreadsByUserID(userID uuid.UUID) ([]model.AIThread, error) {
	var threads []model.AIThread
	err := r.db.Where("user_id = ?", userID).Order("updated_at DESC").Find(&threads).Error
	return threads, err
}

func (r *repository) GetAIThreadByID(threadID uuid.UUID) (*model.AIThread, error) {
	var thread model.AIThread
	err := r.db.Preload("Messages").First(&thread, "id = ?", threadID).Error
	if err != nil {
		return nil, err
	}
	return &thread, nil
}

func (r *repository) DeleteAIThread(threadID uuid.UUID, userID uuid.UUID) error {
	return r.db.Where("id = ? AND user_id = ?", threadID, userID).Delete(&model.AIThread{}).Error
}

func (r *repository) AddAIMessage(threadID uuid.UUID, role string, content string) error {
	msg := &model.AIMessage{
		ThreadID: threadID,
		Role:     role,
		Content:  content,
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(msg).Error; err != nil {
			return err
		}
		// Update the thread's updated_at timestamp
		return tx.Model(&model.AIThread{}).Where("id = ?", threadID).Update("updated_at", time.Now()).Error
	})	
}

func (r *repository) GetAIMessagesByThreadID(threadID uuid.UUID) ([]model.AIMessage, error) {
	var messages []model.AIMessage
	err := r.db.Where("thread_id = ?", threadID).Order("created_at ASC").Find(&messages).Error
	return messages, err
}
