package handler

import (
	"fmt"
	"log"
	"math/rand"
	"net/smtp"
	"time"
	"bytes"
	"encoding/json"
	"net/http"

	"sovraequitara-be/internal/config"
	"sovraequitara-be/internal/model"
	"sovraequitara-be/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	Repo   repository.Repository
	Config *config.Config
}

func NewHandler(repo repository.Repository, cfg *config.Config) *Handler {
	return &Handler{
		Repo:   repo,
		Config: cfg,
	}
}

// ============================================================
// HELPERS
// ============================================================

// hashPassword hashes a plaintext password using bcrypt
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// checkPassword compares a plaintext password against a bcrypt hash
func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// generateJWT creates a signed HS256 JWT token for a user
func (h *Handler) generateJWT(userID string, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(72 * time.Hour).Unix(), // 3 days
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := h.Config.JWTSecret
	if len(secret) > 5 {
		fmt.Printf("JWT Debug - Signing with secret: %s...\n", secret[:5])
	} else {
		fmt.Printf("JWT Debug - Signing with short secret\n")
	}
	return token.SignedString([]byte(secret))
}

func generateOTP() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func (h *Handler) sendOTPEmail(to string, code string) error {
	if h.Config.SMTPHost == "" {
		log.Printf("[WARNING] SMTP_HOST tidak dikonfigurasi. OTP untuk %s adalah: %s\n", to, code)
		return nil
	}

	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: Kode OTP SovraEquitara\r\n"+
		"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"+
		"<html><body>"+
		"<h2>Halo!</h2>"+
		"<p>Kode OTP Anda untuk mendaftar di SovraEquitara adalah:</p>"+
		"<h1 style='color: #0ea5e9;'>%s</h1>"+
		"<p>Kode ini berlaku selama 10 menit.</p>"+
		"</body></html>", to, code))

	addr := fmt.Sprintf("%s:%s", h.Config.SMTPHost, h.Config.SMTPPort)

	var auth smtp.Auth
	if h.Config.SMTPUser != "" && h.Config.SMTPPass != "" {
		auth = smtp.PlainAuth("", h.Config.SMTPUser, h.Config.SMTPPass, h.Config.SMTPHost)
	}

	from := h.Config.SMTPFrom
	if from == "" {
		from = "onboarding@sovraequitara.my.id"
	}

	err := smtp.SendMail(addr, auth, from, []string{to}, msg)
	if err != nil {
		log.Printf("[SMTP ERROR] Gagal mengirim email ke %s: %v\n", to, err)
		return err
	}
	return nil
}

func (h *Handler) sendForgotPasswordEmail(to string, code string) error {
	if h.Config.SMTPHost == "" {
		log.Printf("[WARNING] SMTP_HOST tidak dikonfigurasi. OTP Lupa Password untuk %s adalah: %s\n", to, code)
		return nil
	}

	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: Reset Password SovraEquitara\r\n"+
		"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"+
		"<html><body>"+
		"<h2>Reset Password</h2>"+
		"<p>Kode OTP Anda untuk mereset password di SovraEquitara adalah:</p>"+
		"<h1 style='color: #f43f5e;'>%s</h1>"+
		"<p>Kode ini berlaku selama 10 menit. Jangan berikan kode ini kepada siapapun.</p>"+
		"</body></html>", to, code))

	addr := fmt.Sprintf("%s:%s", h.Config.SMTPHost, h.Config.SMTPPort)

	var auth smtp.Auth
	if h.Config.SMTPUser != "" && h.Config.SMTPPass != "" {
		auth = smtp.PlainAuth("", h.Config.SMTPUser, h.Config.SMTPPass, h.Config.SMTPHost)
	}

	from := h.Config.SMTPFrom
	if from == "" {
		from = "onboarding@sovraequitara.my.id"
	}

	err := smtp.SendMail(addr, auth, from, []string{to}, msg)
	if err != nil {
		log.Printf("[SMTP ERROR] Gagal mengirim email reset ke %s: %v\n", to, err)
		return err
	}
	return nil
}

// ============================================================
// AUTH — REGISTER (Step 1: save OTP + hashed password)
// ============================================================

func (h *Handler) AuthRegister(c *fiber.Ctx) error {
	var req model.AuthRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request: Data tidak valid"})
	}

	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email dan password wajib diisi"})
	}

	if len(req.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password minimal 6 karakter"})
	}

	exists, _ := h.Repo.CheckEmailExists(req.Email)
	if exists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email sudah terdaftar. Silakan login."})
	}

	// Hash password before storing in OTP table
	hashedPw, err := hashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memproses pendaftaran"})
	}

	otpCode := generateOTP()
	err = h.Repo.SaveOTP(req.Email, otpCode, req.Name, hashedPw)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memproses pendaftaran"})
	}

	go h.sendOTPEmail(req.Email, otpCode)

	log.Printf("\n[AUTH] OTP UNTUK %s: %s\n", req.Email, otpCode)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Kode OTP telah dikirim ke email Anda.",
		"email":   req.Email,
	})
}

// ============================================================
// AUTH — VERIFY OTP (Step 2: create profile + issue JWT)
// ============================================================

func (h *Handler) AuthVerify(c *fiber.Ctx) error {
	var req model.VerifyOTPRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request: Data tidak valid"})
	}

	name, passwordHash, err := h.Repo.VerifyOTP(req.Email, req.Token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Kode OTP salah atau sudah kadaluarsa"})
	}

	// Create profile in database (native, no Supabase)
	profile := &model.Profile{
		Email:        req.Email,
		PasswordHash: passwordHash,
		FullName:     name,
		Role:         "USER",
	}

	if err := h.Repo.CreateProfile(profile); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat akun"})
	}

	// Clean up OTP
	h.Repo.DeleteOTP(req.Email)

	// Issue JWT
	tokenString, err := h.generateJWT(profile.ID.String(), profile.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat sesi login"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":      "Verifikasi berhasil!",
		"access_token": tokenString,
		"user": map[string]interface{}{
			"id":    profile.ID.String(),
			"email": req.Email,
			"name":  name,
			"role":  "USER",
		},
	})
}

// ============================================================
// AUTH — LOGIN (bcrypt password check + JWT)
// ============================================================

func (h *Handler) AuthLogin(c *fiber.Ctx) error {
	var req model.AuthRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request"})
	}

	profile, err := h.Repo.GetProfileByEmail(req.Email)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Email atau password salah"})
	}

	if !checkPassword(req.Password, profile.PasswordHash) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Email atau password salah"})
	}

	tokenString, err := h.generateJWT(profile.ID.String(), profile.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat sesi login"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token": tokenString,
		"user": map[string]interface{}{
			"id":     profile.ID.String(),
			"email":  profile.Email,
			"name":   profile.FullName,
			"role":   profile.Role,
			"points": profile.Points,
		},
	})
}

// ============================================================
// AUTH — FORGOT PASSWORD
// ============================================================

func (h *Handler) ForgotPassword(c *fiber.Ctx) error {
	var req model.ForgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Data tidak valid"})
	}

	exists, _ := h.Repo.CheckEmailExists(req.Email)
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Email tidak ditemukan"})
	}

	otpCode := generateOTP()
	if err := h.Repo.SaveForgotPasswordOTP(req.Email, otpCode); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memproses reset password"})
	}

	go h.sendForgotPasswordEmail(req.Email, otpCode)
	log.Printf("[AUTH] FORGOT PASSWORD OTP UNTUK %s: %s\n", req.Email, otpCode)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Kode OTP reset password telah dikirim ke email Anda"})
}

// ============================================================
// AUTH — RESET PASSWORD
// ============================================================

func (h *Handler) ResetPassword(c *fiber.Ctx) error {
	var req model.ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Data tidak valid"})
	}

	if len(req.NewPassword) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password baru minimal 6 karakter"})
	}

	if err := h.Repo.VerifyForgotPasswordOTP(req.Email, req.Token); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Kode OTP salah atau sudah kadaluarsa"})
	}

	// Hash new password and update in database
	hashedPw, err := hashPassword(req.NewPassword)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memproses reset password"})
	}

	if err := h.Repo.UpdatePasswordByEmail(req.Email, hashedPw); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengubah password"})
	}

	h.Repo.DeleteForgotPasswordOTP(req.Email)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Password berhasil direset. Silakan login kembali."})
}

// ============================================================
// AUTH — ADMIN LOGIN (hardcoded credentials)
// ============================================================

type AdminLoginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) AdminLogin(c *fiber.Ctx) error {
	var req AdminLoginReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	profile, err := h.Repo.GetProfileByEmail(req.Email)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Email atau password admin salah!"})
	}

	if profile.Role != "admin" && profile.Role != "super_admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akses ditolak: Anda bukan admin!"})
	}

	if !checkPassword(req.Password, profile.PasswordHash) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Email atau password admin salah!"})
	}

	tokenString, err := h.generateJWT(profile.ID.String(), profile.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat sesi login admin"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":      "Admin login successful",
		"access_token": tokenString,
		"user": map[string]interface{}{
			"id":        profile.ID.String(),
			"email":     profile.Email,
			"full_name": profile.FullName,
			"role":      profile.Role,
		},
	})
}

// ============================================================
// PROFILE
// ============================================================

func (h *Handler) GetProfile(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	profileID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	profile, err := h.Repo.GetProfileByID(profileID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data profil"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": profile})
}

func (h *Handler) UpdateProfile(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	profileID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req model.UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Data tidak valid"})
	}

	if err := h.Repo.UpdateProfile(profileID, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui profil"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Profil berhasil diperbarui"})
}

// ============================================================
// CATEGORIES
// ============================================================

func (h *Handler) GetCategories(c *fiber.Ctx) error {
	categories, err := h.Repo.GetCategories()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat kategori"})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": categories})
}

// ============================================================
// REPORTS
// ============================================================

func (h *Handler) CreateReport(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	profileID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req model.CreateReportRequest

	// Always parse manually for multipart form
	req.Description = c.FormValue("description")
	req.LocationDetail = c.FormValue("location_detail")

	catID := c.FormValue("category_id")
	if catID != "" {
		var id int
		fmt.Sscanf(catID, "%d", &id)
		req.CategoryID = &id
	}

	phone := c.FormValue("phone_number")
	if phone != "" {
		req.PhoneNumber = &phone
	}

	fmt.Sscanf(c.FormValue("latitude"), "%f", &req.Lat)
	fmt.Sscanf(c.FormValue("longitude"), "%f", &req.Lng)

	if req.Description == "" || req.Lat == 0 || req.Lng == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Description, latitude, and longitude are required"})
	}
	var imageURL *string
	file, err := c.FormFile("image")
	if err == nil {
		// Make sure it's an image
		if file.Size > 2*1024*1024 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Ukuran gambar maksimal 2MB"})
		}
		
		filename := fmt.Sprintf("%d-%s", time.Now().Unix(), file.Filename)
		filepath := fmt.Sprintf("./uploads/%s", filename)
		if err := c.SaveFile(file, filepath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan gambar"})
		}
		url := fmt.Sprintf("/uploads/%s", filename)
		imageURL = &url
	}

	report := &model.Report{
		ProfileID:      profileID,
		CategoryID:     req.CategoryID,
		ImageURL:       imageURL,
		Description:    req.Description,
		PhoneNumber:    req.PhoneNumber,
		Latitude:       req.Lat,
		Longitude:      req.Lng,
		LocationDetail: req.LocationDetail,
		Status:         "PENDING",
	}

	if err := h.Repo.CreateReport(report); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengirim laporan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Laporan berhasil dikirim!", "data": report})
}

func (h *Handler) GetMyReports(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	profileID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	reports, err := h.Repo.GetReportsByProfileID(profileID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat riwayat laporan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": reports})
}

func (h *Handler) DeleteReport(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	profileID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	reportIDStr := c.Params("id")
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID laporan tidak valid"})
	}

	err = h.Repo.DeleteReport(reportID, profileID)
	if err != nil {
		if err.Error() == "record not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Laporan tidak ditemukan atau Anda tidak memiliki akses"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus laporan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Laporan berhasil dihapus"})
}

func (h *Handler) GetAllReports(c *fiber.Ctx) error {
	statusFilter := c.Query("status")
	sortBy := c.Query("sort", "recent") // default to recent
	reports, err := h.Repo.GetAllReports(statusFilter, sortBy)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat laporan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": reports})
}

func (h *Handler) GetPublicReports(c *fiber.Ctx) error {
	sortBy := c.Query("sort", "recent") // default to recent
	reports, err := h.Repo.GetPublicReports(sortBy)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat laporan publik"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": reports})
}

// ============================================================
// SAVED REPORTS (For Admins)
// ============================================================

func (h *Handler) ToggleSaveReport(c *fiber.Ctx) error {
	adminIDStr := c.Locals("userID").(string)
	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Akses tidak valid"})
	}

	reportIDStr := c.Params("id")
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID laporan tidak valid"})
	}

	saved, err := h.Repo.ToggleSaveReport(adminID, reportID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan/menghapus laporan tersimpan"})
	}

	statusMsg := "Laporan dihapus dari penyimpanan"
	if saved {
		statusMsg = "Laporan berhasil disimpan"
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": statusMsg, "saved": saved})
}

func (h *Handler) GetSavedReports(c *fiber.Ctx) error {
	adminIDStr := c.Locals("userID").(string)
	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Akses tidak valid"})
	}

	reports, err := h.Repo.GetSavedReports(adminID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat laporan tersimpan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": reports})
}

func (h *Handler) VerifyReport(c *fiber.Ctx) error {
	reportIDStr := c.Params("id")
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request"})
	}

	if err := h.Repo.VerifyReport(reportID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memverifikasi laporan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Laporan berhasil diverifikasi"})
}

func (h *Handler) ResolveReport(c *fiber.Ctx) error {
	reportIDStr := c.Params("id")
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request"})
	}

	if err := h.Repo.ResolveReport(reportID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyelesaikan laporan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Laporan berhasil diselesaikan"})
}

func (h *Handler) CancelReport(c *fiber.Ctx) error {
	reportIDStr := c.Params("id")
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request"})
	}

	if err := h.Repo.CancelReport(reportID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membatalkan laporan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Laporan berhasil dibatalkan dan kembali ke status PENDING"})
}

// ============================================================
// COMMENTS & VOTES
// ============================================================

func (h *Handler) AddComment(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	reportIDStr := c.Params("id")
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request"})
	}

	var req model.CommentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Data tidak valid"})
	}

	comment := &model.Comment{
		ReportID: reportID,
		UserID:   userID,
		Content:  req.Content,
	}

	if err := h.Repo.CreateComment(comment); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menambahkan komentar"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Komentar berhasil ditambahkan", "data": comment})
}

func (h *Handler) GetComments(c *fiber.Ctx) error {
	reportIDStr := c.Params("id")
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request"})
	}

	comments, err := h.Repo.GetCommentsByReportID(reportID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat komentar"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": comments})
}

func (h *Handler) VoteReport(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	reportIDStr := c.Params("id")
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request"})
	}

	var req model.VoteRequest
	if err := c.BodyParser(&req); err != nil || (req.VoteType != 1 && req.VoteType != -1) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Vote tidak valid (harus 1 atau -1)"})
	}

	if err := h.Repo.VoteReport(userID, reportID, req.VoteType); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memberikan vote"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Vote berhasil disimpan"})
}

// ============================================================
// LEADERBOARD
// ============================================================

func (h *Handler) GetLeaderboard(c *fiber.Ctx) error {
	profiles, err := h.Repo.GetLeaderboard()
	if err != nil {
		log.Printf("[ERROR] Gagal memuat leaderboard: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat papan peringkat"})
	}

	log.Printf("[INFO] Leaderboard diakses, ditemukan %d warga\n", len(profiles))
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": profiles})
}

func (h *Handler) GetReportStats(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	profileID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	stats, err := h.Repo.GetReportStats(profileID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat statistik"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": stats})
}

// ============================================================
// AI ASSISTANT
// ============================================================

type GeminiRequest struct {
	Contents          []GeminiContent `json:"contents"`
	SystemInstruction *GeminiSystem   `json:"system_instruction,omitempty"`
}

type GeminiContent struct {
	Role  string       `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiSystem struct {
	Parts []GeminiPart `json:"parts"`
}

type LocalChatRequest struct {
	Model    string         `json:"model"`
	Messages []LocalMessage `json:"messages"`
}

type LocalMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (h *Handler) AIAssistant(c *fiber.Ctx) error {
	var req model.AIAssistantRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Fetch some context from DB
	reports, _ := h.Repo.GetAllReports("", "recent")
	var contextStr string
	for i, r := range reports {
		if i >= 10 { // Limit context to latest 10 reports
			break
		}
		contextStr += fmt.Sprintf("Report ID: %s, Category: %v, Description: %s, Status: %s, Location: %s\n", r.ID, r.CategoryID, r.Description, r.Status, r.LocationDetail)
	}
	systemPrompt := `You are an AI Assistant for the SovraEquitara platform admin dashboard. 
SovraEquitara is a citizen reporting platform that aims to resolve issues efficiently.
Your role is to help the admin manage the platform, analyze reports, and give actionable advice based on the provided data.
IMPORTANT: You must ONLY answer questions related to SovraEquitara, the provided reports, or administrative tasks on this platform. If the admin asks something entirely irrelevant, you MUST decline politely and state that you are the SovraEquitara AI Assistant and can only assist with platform-related matters.
When you reference a report, ALWAYS include a detail button using this exact format: [DETAIL_BTN:the-report-id].
Here are the recent reports:
` + contextStr

	if req.Model == "local" {
		// LM Studio
		payload := LocalChatRequest{
			Model: "qwen2.5-vl-3b-instruct",
			Messages: []LocalMessage{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: req.Query},
			},
		}
		body, _ := json.Marshal(payload)

		resp, err := http.Post("http://127.0.0.1:1234/v1/chat/completions", "application/json", bytes.NewBuffer(body))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Local AI is unreachable"})
		}
		defer resp.Body.Close()

		var res map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&res)

		choices, ok := res["choices"].([]interface{})
		if !ok || len(choices) == 0 {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Invalid local AI response"})
		}

		choice := choices[0].(map[string]interface{})
		message := choice["message"].(map[string]interface{})
		content := message["content"].(string)

		return c.Status(fiber.StatusOK).JSON(fiber.Map{"response": content})

	} else {
		// Gemini
		apiKey := h.Config.GeminiAPIKey
		if apiKey == "" {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gemini API key is not configured"})
		}
		url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + apiKey

		payload := GeminiRequest{
			SystemInstruction: &GeminiSystem{
				Parts: []GeminiPart{{Text: systemPrompt}},
			},
			Contents: []GeminiContent{
				{
					Role: "user",
					Parts: []GeminiPart{{Text: req.Query}},
				},
			},
		}

		body, _ := json.Marshal(payload)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gemini API is unreachable"})
		}
		defer resp.Body.Close()

		var res map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&res)

		candidates, ok := res["candidates"].([]interface{})
		if !ok || len(candidates) == 0 {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Invalid Gemini response"})
		}

		candidate := candidates[0].(map[string]interface{})
		contentObj := candidate["content"].(map[string]interface{})
		parts := contentObj["parts"].([]interface{})
		firstPart := parts[0].(map[string]interface{})

		return c.Status(fiber.StatusOK).JSON(fiber.Map{"response": firstPart["text"]})
	}
}

// ============================================================
// ADMIN MANAGEMENT (Super Admin)
// ============================================================

func (h *Handler) GetAdmins(c *fiber.Ctx) error {
	admins, err := h.Repo.GetAdmins()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar admin"})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": admins})
}

func (h *Handler) CreateAdmin(c *fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	exists, _ := h.Repo.CheckEmailExists(req.Email)
	if exists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email sudah terdaftar"})
	}

	hashedPw, err := hashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memproses password"})
	}

	profile := &model.Profile{
		Email:        req.Email,
		PasswordHash: hashedPw,
		FullName:     req.FullName,
		Role:         "admin",
	}

	if err := h.Repo.CreateAdmin(profile); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat admin"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Admin berhasil dibuat"})
}

func (h *Handler) UpdateAdmin(c *fiber.Ctx) error {
	adminIDStr := c.Params("id")
	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request"})
	}

	var req struct {
		FullName string `json:"full_name"`
		Password string `json:"password"` // optional
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	var hashedPw string
	if req.Password != "" {
		hashedPw, err = hashPassword(req.Password)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memproses password"})
		}
	}

	if err := h.Repo.UpdateAdmin(adminID, req.FullName, hashedPw); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengupdate admin"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Admin berhasil diupdate"})
}

func (h *Handler) DeleteAdmin(c *fiber.Ctx) error {
	adminIDStr := c.Params("id")
	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request"})
	}

	if err := h.Repo.DeleteAdmin(adminID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus admin"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Admin berhasil dihapus"})
}

func (h *Handler) GetUserStats(c *fiber.Ctx) error {
	profileIDStr := c.Params("id")
	profileID, err := uuid.Parse(profileIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid profile ID"})
	}

	stats, err := h.Repo.GetReportStats(profileID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat statistik pengguna"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": stats})
}

// ============================================================
// CHAT SYSTEM
// ============================================================

// SendChatMessage — any authenticated user sends a message to super admin
func (h *Handler) SendChatMessage(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req model.SendMessageRequest
	if err := c.BodyParser(&req); err != nil || req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Pesan tidak boleh kosong"})
	}

	// Get or create conversation for this user
	conv, err := h.Repo.GetOrCreateConversation(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat percakapan"})
	}

	msg := &model.Message{
		ConversationID: conv.ID,
		SenderID:       userID,
		Content:        req.Content,
	}
	if err := h.Repo.SendMessage(msg); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengirim pesan"})
	}

	// Update conversation metadata (unread for super admin)
	truncated := req.Content
	if len(truncated) > 100 {
		truncated = truncated[:100] + "..."
	}
	h.Repo.UpdateConversationLastMessage(conv.ID, truncated, true)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Pesan terkirim", "data": msg})
}

// GetMyMessages — user gets their own conversation messages
func (h *Handler) GetMyMessages(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	conv, err := h.Repo.GetOrCreateConversation(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat percakapan"})
	}

	messages, err := h.Repo.GetMessagesByConversationID(conv.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat pesan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": messages, "conversation_id": conv.ID})
}

// GetAllConversations — super admin lists all conversations with optional role filter
func (h *Handler) GetAllConversations(c *fiber.Ctx) error {
	roleFilter := c.Query("role", "")
	convs, err := h.Repo.GetAllConversations(roleFilter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat percakapan"})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": convs})
}

// GetConversationMessages — super admin gets messages of a specific conversation
func (h *Handler) GetConversationMessages(c *fiber.Ctx) error {
	convIDStr := c.Params("id")
	convID, err := uuid.Parse(convIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid conversation ID"})
	}

	messages, err := h.Repo.GetMessagesByConversationID(convID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat pesan"})
	}

	// Auto-mark as read when super admin opens
	h.Repo.MarkConversationAsRead(convID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": messages})
}

// ReplyChatMessage — super admin replies to a conversation
func (h *Handler) ReplyChatMessage(c *fiber.Ctx) error {
	convIDStr := c.Params("id")
	convID, err := uuid.Parse(convIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid conversation ID"})
	}

	userIDStr := c.Locals("userID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req model.SendMessageRequest
	if err := c.BodyParser(&req); err != nil || req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Pesan tidak boleh kosong"})
	}

	msg := &model.Message{
		ConversationID: convID,
		SenderID:       userID,
		Content:        req.Content,
	}
	if err := h.Repo.SendMessage(msg); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengirim balasan"})
	}

	truncated := req.Content
	if len(truncated) > 100 {
		truncated = truncated[:100] + "..."
	}
	h.Repo.UpdateConversationLastMessage(convID, truncated, false)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Balasan terkirim", "data": msg})
}

// ============================================================
// NOTIFICATIONS SYSTEM
// ============================================================

func (h *Handler) GetNotifications(c *fiber.Ctx) error {
	role := c.Locals("role").(string)
	
	notifs, err := h.Repo.GetNotifications(role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat notifikasi"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": notifs})
}

func (h *Handler) CreateNotification(c *fiber.Ctx) error {
	creatorIDStr := c.Locals("userID").(string)
	creatorID, err := uuid.Parse(creatorIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req model.CreateNotificationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Input tidak valid"})
	}

	notif := &model.Notification{
		Title:      req.Title,
		Message:    req.Message,
		Type:       req.Type,
		TargetRole: req.TargetRole,
		CreatedBy:  &creatorID,
	}

	if err := h.Repo.CreateNotification(notif); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat notifikasi"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Notifikasi/Pesan Darurat berhasil dibuat", "data": notif})
}

func (h *Handler) UpdateNotification(c *fiber.Ctx) error {
	notifIDStr := c.Params("id")
	notifID, err := uuid.Parse(notifIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID tidak valid"})
	}

	var req model.CreateNotificationRequest // Reusing struct for simplicity
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Input tidak valid"})
	}

	if err := h.Repo.UpdateNotification(notifID, req.Title, req.Message, req.Type, req.TargetRole); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui notifikasi"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Notifikasi berhasil diperbarui"})
}

func (h *Handler) DeleteNotification(c *fiber.Ctx) error {
	notifIDStr := c.Params("id")
	notifID, err := uuid.Parse(notifIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID tidak valid"})
	}

	if err := h.Repo.DeleteNotification(notifID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus notifikasi"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Notifikasi berhasil dihapus"})
}

// ============================================================
// PROFILE LISTING (Public)
// ============================================================

// GetAllProfiles
func (h *Handler) GetAllProfiles(c *fiber.Ctx) error {
	search := c.Query("search", "")
	role := c.Query("role", "")
	profiles, err := h.Repo.GetAllProfiles(search, role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat data profil"})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": profiles})
}

// UploadAvatar
func (h *Handler) UploadAvatar(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	profileID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Pilih gambar profil terlebih dahulu"})
	}

	if file.Size > 2*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Ukuran gambar maksimal 2MB"})
	}

	filename := fmt.Sprintf("avatar-%s-%d-%s", profileID.String(), time.Now().Unix(), file.Filename)
	filepath := fmt.Sprintf("./uploads/%s", filename)
	if err := c.SaveFile(file, filepath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan gambar"})
	}
	url := fmt.Sprintf("/uploads/%s", filename)

	if err := h.Repo.UpdateAvatar(profileID, url); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui avatar"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Foto profil berhasil diperbarui", "avatar_url": url})
}

func (h *Handler) GetProfileByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID"})
	}
	
	profile, err := h.Repo.GetProfileByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Profil tidak ditemukan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": fiber.Map{
		"id":         profile.ID,
		"email":      profile.Email,
		"full_name":  profile.FullName,
		"phone":      profile.Phone,
		"avatar_url": profile.AvatarURL,
		"points":     profile.Points,
		"role":       profile.Role,
		"created_at": profile.CreatedAt,
	}})
}
