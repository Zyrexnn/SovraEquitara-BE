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

	// Reports
	CreateReport(profileID uuid.UUID, req model.CreateReportRequest) error
	GetReportsByProfileID(profileID uuid.UUID) ([]model.Report, error)
	GetAllReports(statusFilter string) ([]model.Report, error)
	VerifyReport(reportID uuid.UUID) error
	ResolveReport(reportID uuid.UUID) error

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

func (r *repository) UpdatePasswordByEmail(email, passwordHash string) error {
	return r.db.Model(&model.Profile{}).Where("email = ?", email).Update("password_hash", passwordHash).Error
}

// ============================================================
// REPORTS
// ============================================================

func (r *repository) CreateReport(profileID uuid.UUID, req model.CreateReportRequest) error {
	query := `INSERT INTO reports (profile_id, description, phone_number, latitude, longitude, location_detail) 
			  VALUES ($1, $2, $3, $4, $5, $6)`
	return r.db.Exec(query, profileID, req.Description, req.PhoneNumber, req.Lat, req.Lng, req.LocationDetail).Error
}

func (r *repository) GetReportsByProfileID(profileID uuid.UUID) ([]model.Report, error) {
	var reports []model.Report
	query := `SELECT id, profile_id, description, phone_number, 
			  latitude, longitude, location_detail,
			  status, created_at, updated_at 
			  FROM reports WHERE profile_id = $1 ORDER BY created_at DESC`
	err := r.db.Raw(query, profileID).Scan(&reports).Error
	return reports, err
}

func (r *repository) GetAllReports(statusFilter string) ([]model.Report, error) {
	var reports []model.Report
	if statusFilter != "" {
		query := `SELECT id, profile_id, description, phone_number, 
				  latitude, longitude, location_detail,
				  status, created_at, updated_at 
				  FROM reports WHERE status = $1 ORDER BY created_at DESC`
		err := r.db.Raw(query, statusFilter).Scan(&reports).Error
		return reports, err
	}
	query := `SELECT id, profile_id, description, phone_number, 
			  latitude, longitude, location_detail,
			  status, created_at, updated_at 
			  FROM reports ORDER BY created_at DESC`
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

// ============================================================
// LEADERBOARD
// ============================================================

func (r *repository) GetLeaderboard() ([]model.Profile, error) {
	var profiles []model.Profile
	err := r.db.Order("points desc").Limit(10).Find(&profiles).Error
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
