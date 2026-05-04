package repository

import (
	"sovraequitara-be/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	CheckEmailExists(email string) (bool, error)
	CreateReport(profileID uuid.UUID, req model.CreateReportRequest) error
	GetReportsByProfileID(profileID uuid.UUID) ([]model.Report, error)
	GetAllReports(statusFilter string) ([]model.Report, error)
	VerifyReport(reportID uuid.UUID) error
	ResolveReport(reportID uuid.UUID) error
	GetLeaderboard() ([]model.Profile, error)
	GetProfileByID(id uuid.UUID) (*model.Profile, error)
	EnsureProfileExists(id uuid.UUID, email string, name string) error
	UpdateProfile(id uuid.UUID, req model.UpdateProfileRequest) error
	// OTP Methods
	SaveOTP(email, code, name, password string) error
	VerifyOTP(email, code string) (name, password string, err error)
	DeleteOTP(email string) error
	// Forgot Password Methods
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

func (r *repository) CheckEmailExists(email string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Profile{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (r *repository) CreateReport(profileID uuid.UUID, req model.CreateReportRequest) error {
	query := `INSERT INTO reports (profile_id, description, phone_number, location, latitude, longitude, location_detail) 
			  VALUES (?, ?, ?, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography, ?, ?, ?)`
	err := r.db.Exec(query, profileID, req.Description, req.PhoneNumber, req.Lng, req.Lat, req.Lat, req.Lng, req.LocationDetail).Error
	return err
}

func (r *repository) GetReportsByProfileID(profileID uuid.UUID) ([]model.Report, error) {
	var reports []model.Report
	query := `SELECT id, profile_id, description, phone_number, 
			  latitude, longitude, location_detail,
			  status, created_at, updated_at 
			  FROM reports WHERE profile_id = ? ORDER BY created_at DESC`
	err := r.db.Raw(query, profileID).Scan(&reports).Error
	return reports, err
}

func (r *repository) GetAllReports(statusFilter string) ([]model.Report, error) {
	var reports []model.Report
	query := `SELECT id, profile_id, description, phone_number, 
			  latitude, longitude, location_detail,
			  status, created_at, updated_at 
			  FROM reports`
	if statusFilter != "" {
		query += ` WHERE status = ?`
		query += ` ORDER BY created_at DESC`
		err := r.db.Raw(query, statusFilter).Scan(&reports).Error
		return reports, err
	}
	query += ` ORDER BY created_at DESC`
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

func (r *repository) GetLeaderboard() ([]model.Profile, error) {
	var profiles []model.Profile
	err := r.db.Order("points desc").Limit(10).Find(&profiles).Error
	return profiles, err
}

func (r *repository) GetProfileByID(id uuid.UUID) (*model.Profile, error) {
	var profile model.Profile
	err := r.db.First(&profile, "id = ?", id).Error
	return &profile, err
}

func (r *repository) EnsureProfileExists(id uuid.UUID, email string, name string) error {
	var count int64
	r.db.Model(&model.Profile{}).Where("id = ?", id).Count(&count)
	if count == 0 {
		profile := model.Profile{
			ID:       id,
			Email:    email,
			FullName: name,
			Role:     "USER",
		}
		return r.db.Create(&profile).Error
	}
	return nil
}

func (r *repository) UpdateProfile(id uuid.UUID, req model.UpdateProfileRequest) error {
	return r.db.Model(&model.Profile{}).Where("id = ?", id).Updates(map[string]interface{}{
		"full_name": req.FullName,
		"phone":     req.Phone,
	}).Error
}

func (r *repository) SaveOTP(email, code, name, password string) error {
	query := `INSERT INTO otps (email, code, name, password, created_at) 
	          VALUES (?, ?, ?, ?, NOW()) 
	          ON CONFLICT (email) DO UPDATE SET code = EXCLUDED.code, name = EXCLUDED.name, password = EXCLUDED.password, created_at = NOW()`
	return r.db.Exec(query, email, code, name, password).Error
}

func (r *repository) VerifyOTP(email, code string) (string, string, error) {
	var res struct {
		Name     string
		Password string
	}
	err := r.db.Raw("SELECT name, password FROM otps WHERE email = ? AND code = ?", email, code).Scan(&res).Error
	if err != nil {
		return "", "", err
	}
	if res.Name == "" {
		return "", "", gorm.ErrRecordNotFound
	}
	return res.Name, res.Password, nil
}

func (r *repository) DeleteOTP(email string) error {
	return r.db.Exec("DELETE FROM otps WHERE email = ?", email).Error
}

func (r *repository) SaveForgotPasswordOTP(email, code string) error {
	query := `INSERT INTO forgot_password_otps (email, code, created_at) 
	          VALUES (?, ?, NOW()) 
	          ON CONFLICT (email) DO UPDATE SET code = EXCLUDED.code, created_at = NOW()`
	return r.db.Exec(query, email, code).Error
}

func (r *repository) VerifyForgotPasswordOTP(email, code string) error {
	var count int64
	err := r.db.Table("forgot_password_otps").Where("email = ? AND code = ?", email, code).Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *repository) DeleteForgotPasswordOTP(email string) error {
	return r.db.Exec("DELETE FROM forgot_password_otps WHERE email = ?", email).Error
}
