package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/smtp"
	"os"
	"sync"
	"time"
	"strings"

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
	Hub    *SSEHub
}

func NewHandler(repo repository.Repository, cfg *config.Config) *Handler {
	return &Handler{
		Repo:   repo,
		Config: cfg,
		Hub:    NewSSEHub(),
	}
}

// ============================================================
// SSE HUB — Real-time broadcast to all connected clients
// ============================================================

// SSEHub manages all active SSE client connections.
type SSEHub struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

// NewSSEHub creates and returns a new SSEHub.
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[chan []byte]struct{}),
	}
}

// addClient registers a new SSE client channel.
func (h *SSEHub) addClient(ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[ch] = struct{}{}
}

// removeClient deregisters an SSE client channel.
func (h *SSEHub) removeClient(ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, ch)
}

// Broadcast sends an SSEEvent as JSON to all connected clients.
func (h *SSEHub) Broadcast(event model.SSEEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("[SSE] Failed to marshal event: %v", err)
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- payload:
		default:
			// Drop if client channel is full (slow consumer)
		}
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
			"full_name":   profile.FullName,
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
		// Log silently for security audits and debugging, but return generic success to user
		log.Printf("[AUTH] ForgotPassword requested for non-registered email: %s\n", req.Email)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Kode OTP reset password telah dikirim ke email Anda jika terdaftar"})
	}

	otpCode := generateOTP()
	if err := h.Repo.SaveForgotPasswordOTP(req.Email, otpCode); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memproses reset password"})
	}

	go h.sendForgotPasswordEmail(req.Email, otpCode)
	log.Printf("[AUTH] FORGOT PASSWORD OTP UNTUK %s: %s\n", req.Email, otpCode)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Kode OTP reset password telah dikirim ke email Anda jika terdaftar"})
}

type VerifyForgotPasswordOTPRequest struct {
	Email string `json:"email" validate:"required"`
	Token string `json:"token" validate:"required"`
}

func (h *Handler) VerifyForgotPasswordOTP(c *fiber.Ctx) error {
	var req VerifyForgotPasswordOTPRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Data tidak valid"})
	}

	if err := h.Repo.VerifyForgotPasswordOTP(req.Email, req.Token); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Kode OTP salah atau sudah kadaluarsa"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Kode OTP valid"})
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

// UpdateProfilePassword — change password without OTP for any authenticated user (especially Admin/Super Admin).
func (h *Handler) UpdateProfilePassword(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	profileID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Data tidak valid"})
	}
	if len(req.CurrentPassword) == 0 || len(req.NewPassword) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Kata sandi baru minimal 6 karakter"})
	}

	// Fetch the user's current hashed password from the database
	profile, err := h.Repo.GetProfileByID(profileID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat profil"})
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(profile.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Kata sandi saat ini tidak benar"})
	}

	// Hash new password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memproses kata sandi baru"})
	}

	if err := h.Repo.UpdateProfilePassword(profileID, string(hashed)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui kata sandi"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Kata sandi berhasil diperbarui"})
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
	form, err := c.MultipartForm()
	var imageUrls []string
	if err == nil && form != nil {
		files := form.File["images"]
		if len(files) > 3 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Maksimal 3 gambar yang diperbolehkan"})
		}
		for _, file := range files {
			if file.Size > 2*1024*1024 {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Ukuran gambar maksimal 2MB per file"})
			}
			filename := fmt.Sprintf("%d-%s", time.Now().Unix(), file.Filename)
			filepath := fmt.Sprintf("./uploads/%s", filename)
			if err := c.SaveFile(file, filepath); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan gambar"})
			}
			imageUrls = append(imageUrls, fmt.Sprintf("/uploads/%s", filename))
		}
	}

	report := &model.Report{
		ProfileID:      profileID,
		CategoryID:     req.CategoryID,
		ImageURLs:      imageUrls,
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

func (h *Handler) GetReportByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	reportID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID Laporan tidak valid"})
	}
	report, err := h.Repo.GetReportByID(reportID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Laporan tidak ditemukan"})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": report})
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

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Laporan berhasil ditandai menunggu persetujuan pelapor"})
}

func (h *Handler) ApproveReportResolution(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	profileID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	reportIDStr := c.Params("id")
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request"})
	}

	if err := h.Repo.ApproveResolution(reportID, profileID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyetujui penyelesaian laporan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Penyelesaian laporan disetujui, laporan telah Selesai"})
}

func (h *Handler) RejectReportResolution(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	profileID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	reportIDStr := c.Params("id")
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request"})
	}

	if err := h.Repo.RejectResolution(reportID, profileID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menolak penyelesaian laporan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Penyelesaian laporan ditolak, status dikembalikan ke Diproses"})
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

	// Broadcast updated comment count to all SSE clients
	commentCount, _ := h.Repo.GetCommentCount(reportID)
	h.Hub.Broadcast(model.SSEEvent{
		EventType:    "comment_update",
		ReportID:     reportID,
		CommentCount: int(commentCount),
	})

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

	// Broadcast updated vote count to all SSE clients
	voteCount, _ := h.Repo.GetVoteCount(reportID)
	h.Hub.Broadcast(model.SSEEvent{
		EventType: "vote_update",
		ReportID:  reportID,
		VoteCount: int(voteCount),
	})

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Vote berhasil disimpan", "vote_count": voteCount})
}

// GetVoteStatus returns the current user's vote type for a given report (1, -1, or 0 = not voted).
func (h *Handler) GetVoteStatus(c *fiber.Ctx) error {
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

	voteType, err := h.Repo.GetUserVoteForReport(userID, reportID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil status vote"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"vote_type": voteType})
}

// SSEHandler handles Server-Sent Events connections.
// Clients connect to GET /api/events and receive real-time vote/comment updates.
func (h *Handler) SSEHandler(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Access-Control-Allow-Origin", "*")
	c.Set("X-Accel-Buffering", "no") // Disable Nginx buffering if proxied

	// Create a buffered channel for this client
	ch := make(chan []byte, 32)
	h.Hub.addClient(ch)

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		// Send initial keep-alive comment
		fmt.Fprintf(w, ": connected\n\n")
		w.Flush()

		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		defer h.Hub.removeClient(ch)

		for {
			select {
			case payload, ok := <-ch:
				if !ok {
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", payload)
				if err := w.Flush(); err != nil {
					return // Client disconnected
				}
			case <-ticker.C:
				// Heartbeat to keep connection alive
				fmt.Fprintf(w, ": heartbeat\n\n")
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})
	return nil
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
	userIDStr := c.Locals("userID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

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
	systemPrompt := `Anda adalah Asisten AI untuk dashboard admin platform SovraEquitara (atau SuperAI Assistant jika pengguna adalah Super Admin).
SovraEquitara adalah platform pelaporan warga yang bertujuan menyelesaikan berbagai masalah kota secara efisien.
Tugas Anda adalah membantu admin/super admin mengelola platform, menganalisis laporan, mencari data laporan, dan memberikan saran tindakan berdasarkan data laporan yang disediakan di bawah ini.

ATURAN UTAMA:
1. BATASAN KONTEKS (STRICT SCOPE): Anda hanya boleh menjawab pertanyaan yang berkaitan dengan SovraEquitara, laporan-laporan warga yang disediakan, analisis data laporan, moderasi, atau tugas administratif di platform ini. Jika pengguna menanyakan hal yang sama sekali tidak relevan (seperti resep makanan, pemrograman umum diluar platform, sejarah dunia, obrolan santai yang tidak berhubungan, dll.), tolak secara sopan dengan kalimat persis: "Maaf, saya adalah Asisten AI Administrasi SovraEquitara dan hanya dapat membantu Anda terkait manajemen laporan, aduan kota, moderasi staf, dan data platform."
Permintaan untuk mencari, memfilter, meringkas, atau menampilkan laporan dengan status tertentu (seperti "pending", "tertunda", "valid", "resolved", "laporan masuk") adalah sepenuhnya relevan dengan manajemen laporan dan HARUS dijawab dengan baik menggunakan data laporan yang tersedia!
2. JANGAN BERHALUSINASI: Jangan pernah membuat-buat data laporan baru, UUID palsu, statistik karangan, kategori khayalan, atau informasi lain yang tidak ada dalam data konteks di bawah ini. Semua informasi harus sepenuhnya berbasis data nyata yang disediakan. Jika data tidak ditemukan, sampaikan secara jujur bahwa laporan dengan kriteria tersebut tidak ada dalam data saat ini.
3. TOMBOL DETAIL LAPORAN: Saat Anda menyebutkan atau mereferensikan sebuah laporan dalam jawaban Anda, Anda HARUS menyertakan tombol detail dengan format persis seperti ini: [DETAIL_BTN:the-report-id] di mana "the-report-id" adalah UUID laporan tersebut dari data di bawah ini. Ini sangat penting agar admin bisa langsung meninjau laporan tersebut.
4. GRAFIK DINAMIS (DYNAMIC CHARTS): Jika pengguna meminta analisis data, perbandingan kategori, statistik laporan, atau visualisasi tren, Anda dapat menampilkan grafik interaktif secara otomatis. Gunakan format persis seperti ini: [CHART:tipe:label1,nilai1|label2,nilai2|label3,nilai3|...]
   Pilihan tipe: "bar" (grafik batang), "line" (grafik garis), atau "pie" (grafik lingkaran/kue).
   Contoh: [CHART:bar:Pending,5|Valid,12|Selesai,8]

Berikut adalah daftar laporan warga terbaru yang tersedia di database saat ini:
` + contextStr

	// Resolve or dynamically create thread ID
	threadID := req.ThreadID
	if threadID == uuid.Nil {
		title := req.Query
		if len(title) > 30 {
			title = title[:27] + "..."
		}
		newThread, err := h.Repo.CreateAIThread(userID, title)
		if err == nil {
			threadID = newThread.ID
		}
	}

	// Fetch past messages in thread if threadID is valid
	var pastMessages []model.AIMessage
	if threadID != uuid.Nil {
		pastMessages, _ = h.Repo.GetAIMessagesByThreadID(threadID)
	}

	var responseText string

	if req.Model == "local" {
		// LM Studio
		payload := LocalChatRequest{
			Model: "qwen2.5-vl-3b-instruct",
			Messages: []LocalMessage{
				{Role: "system", Content: systemPrompt},
			},
		}

		// Append past conversation history
		for _, m := range pastMessages {
			payload.Messages = append(payload.Messages, LocalMessage{
				Role:    m.Role,
				Content: m.Content,
			})
		}

		// Append current query
		payload.Messages = append(payload.Messages, LocalMessage{
			Role:    "user",
			Content: req.Query,
		})

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
		responseText = message["content"].(string)

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
			Contents: []GeminiContent{},
		}

		// Append past conversation history
		for _, m := range pastMessages {
			role := m.Role
			if role == "assistant" {
				role = "model"
			}
			payload.Contents = append(payload.Contents, GeminiContent{
				Role:  role,
				Parts: []GeminiPart{{Text: m.Content}},
			})
		}

		// Append current query
		payload.Contents = append(payload.Contents, GeminiContent{
			Role:  "user",
			Parts: []GeminiPart{{Text: req.Query}},
		})

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
		responseText = firstPart["text"].(string)
	}

	// Save messages dynamically under the active thread
	if threadID != uuid.Nil {
		_ = h.Repo.AddAIMessage(threadID, "user", req.Query)
		_ = h.Repo.AddAIMessage(threadID, "assistant", responseText)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"response":  responseText,
		"thread_id": threadID,
	})
}

// ============================================================
// AI CHAT HISTORY HANDLERS (Admin Exclusive)
// ============================================================

func (h *Handler) CreateAIThread(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var body struct {
		Title string `json:"title"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}
	if body.Title == "" {
		body.Title = "Obrolan Baru"
	}

	thread, err := h.Repo.CreateAIThread(userID, body.Title)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat sesi obrolan"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Sesi obrolan berhasil dibuat",
		"data":    thread,
	})
}

func (h *Handler) GetAIThreads(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	threads, err := h.Repo.GetAIThreadsByUserID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat riwayat obrolan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": threads,
	})
}

func (h *Handler) GetAIThreadMessages(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	_, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	threadIDStr := c.Params("id")
	threadID, err := uuid.Parse(threadIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID Sesi tidak valid"})
	}

	messages, err := h.Repo.GetAIMessagesByThreadID(threadID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat pesan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": messages,
	})
}

func (h *Handler) DeleteAIThread(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	threadIDStr := c.Params("id")
	threadID, err := uuid.Parse(threadIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID Sesi tidak valid"})
	}

	err = h.Repo.DeleteAIThread(threadID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus sesi obrolan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Sesi obrolan berhasil dihapus",
	})
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
	userIDStr := c.Locals("userID").(string)
	userID, _ := uuid.Parse(userIDStr)
	
	notifs, err := h.Repo.GetNotifications(userID, role)
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

	targetRole := strings.ToUpper(req.TargetRole)
	if targetRole == "CITIZEN" {
		targetRole = "USER"
	} else if targetRole == "SUPER_ADMIN" {
		targetRole = "SUPERADMIN"
	}

	notif := &model.Notification{
		Title:      req.Title,
		Message:    req.Message,
		Type:       req.Type,
		TargetRole: targetRole,
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

	targetRoleUpdate := strings.ToUpper(req.TargetRole)
	if targetRoleUpdate == "CITIZEN" {
		targetRoleUpdate = "USER"
	} else if targetRoleUpdate == "SUPER_ADMIN" {
		targetRoleUpdate = "SUPERADMIN"
	}

	if err := h.Repo.UpdateNotification(notifID, req.Title, req.Message, req.Type, targetRoleUpdate); err != nil {
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

// ============================================================
// USER AI ASSISTANT
// ============================================================

func (h *Handler) UserAIAssistant(c *fiber.Ctx) error {
	var req model.AIAssistantRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	userIDStr := c.Locals("userID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	
	// Fetch some context from DB specific to this user
	reports, _ := h.Repo.GetReportsByProfileID(userID)
	var contextStr string
	for i, r := range reports {
		if i >= 10 { // Limit context to latest 10 reports
			break
		}
		contextStr += fmt.Sprintf("Report ID: %s, Category: %v, Description: %s, Status: %s, Location: %s\n", r.ID, r.CategoryID, r.Description, r.Status, r.LocationDetail)
	}
	systemPrompt := `Anda adalah Asisten AI yang ramah untuk warga yang menggunakan platform SovraEquitara.
SovraEquitara adalah platform pelaporan warga yang bertujuan menyelesaikan berbagai masalah kota secara efisien.
Tugas Anda adalah membantu warga memahami cara membuat laporan baru, melacak status laporan mereka, memberikan panduan pelaporan, dan memberikan saran/informasi yang bermanfaat berdasarkan data laporan warga tersebut yang disediakan di bawah ini.

ATURAN UTAMA:
1. BATASAN KONTEKS (STRICT SCOPE): Anda hanya boleh menjawab pertanyaan yang berkaitan dengan SovraEquitara, pembuatan laporan, pengaduan kota, status laporan warga tersebut, atau fitur-fitur di platform ini. Jika pengguna menanyakan hal yang tidak relevan (seperti resep makanan, pemrograman umum, trivia umum, atau topik lainnya), Anda harus menolak secara sopan dengan kalimat persis: "Maaf, saya adalah Asisten AI SovraEquitara dan hanya dapat membantu Anda terkait dengan aduan kota atau fitur pada platform ini."
Permintaan melacak status laporan, mencari laporan dengan kata kunci tertentu, atau panduan melaporkan masalah kota adalah sepenuhnya relevan dan HARUS dilayani dengan ramah menggunakan data yang tersedia.
2. JANGAN BERHALUSINASI: Jangan pernah mengarang, memalsukan, atau berasumsi tentang status, ID, kategori, atau detail laporan yang tidak ada pada konteks data di bawah ini. Jika warga belum memiliki laporan dalam daftar di bawah, sampaikan secara ramah bahwa mereka belum membuat laporan dan tawarkan bantuan untuk memandu cara membuat laporan.
3. RAMAH & SOLUTIF: Selalu gunakan Bahasa Indonesia yang ramah, sopan, hangat, dan solutif.

Berikut adalah daftar laporan terbaru milik warga ini (jika ada):
` + contextStr

	// User AI always uses local model
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
}

// ============================================================
// SUPER ADMIN — DATABASE BACKUP
// ============================================================

func (h *Handler) BackupDatabase(c *fiber.Ctx) error {
	// 1. Get SQL dump string from Repository
	sqlContent, err := h.Repo.BackupDatabase()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghasilkan data cadangan database: " + err.Error()})
	}

	// 2. Ensure the backups directory exists
	backupsDir := "./backups"
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyiapkan folder penyimpanan cadangan: " + err.Error()})
	}

	// 3. Generate filename with timestamp
	timestamp := time.Now().Format("2006-01-02_150405")
	filename := fmt.Sprintf("sovra_db_backup_%s.sql", timestamp)
	filepath := fmt.Sprintf("%s/%s", backupsDir, filename)

	// 4. Write SQL string to file
	if err := os.WriteFile(filepath, []byte(sqlContent), 0644); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan berkas cadangan database ke disk: " + err.Error()})
	}

	// 5. Return success and download URL
	downloadURL := fmt.Sprintf("/backups/%s", filename)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":      "Backup database berhasil dibuat",
		"filename":     filename,
		"download_url": downloadURL,
	})
}
