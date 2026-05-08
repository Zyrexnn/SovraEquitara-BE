package handler

import (
	"fmt"
	"log"
	"math/rand"
	"net/smtp"
	"time"

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
	return token.SignedString([]byte(h.Config.JWTSecret))
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
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) AdminLogin(c *fiber.Ctx) error {
	var req AdminLoginReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	if req.Username == "ikhsan" && req.Password == "0721" {
		tokenString, err := h.generateJWT("admin-ikhsan", "admin")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not login admin"})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message":      "Admin login successful",
			"access_token": tokenString,
			"user": map[string]interface{}{
				"id":    "admin-ikhsan",
				"email": "ikhsan@admin.com",
				"role":  "admin",
			},
		})
	}

	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Username atau Password Admin salah!"})
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
