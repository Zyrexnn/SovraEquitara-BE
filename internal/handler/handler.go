package handler

import (
	"sovraequitara-be/internal/config"
	"sovraequitara-be/internal/model"
	"sovraequitara-be/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nedpals/supabase-go"
)

type Handler struct {
	Repo     repository.Repository
	Config   *config.Config
	Supabase *supabase.Client
}

func NewHandler(repo repository.Repository, cfg *config.Config) *Handler {
	supabaseClient := supabase.CreateClient(cfg.SupabaseURL, cfg.SupabaseKey)
	return &Handler{
		Repo:     repo,
		Config:   cfg,
		Supabase: supabaseClient,
	}
}

// ----------------- AUTH -----------------

func (h *Handler) AuthOTP(c *fiber.Ctx) error {
	var req model.OTPRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request: Invalid request body"})
	}

	err := h.Supabase.Auth.SendMagicLink(c.Context(), req.Email)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Bad Request: Failed to send OTP. Please wait 60 seconds before trying again.",
			"detail": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "OTP sent successfully"})
}

func (h *Handler) AuthVerify(c *fiber.Ctx) error {
	var req model.VerifyOTPRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request: Invalid request body"})
	}

	res, err := h.Supabase.Auth.VerifyOtp(c.Context(), supabase.VerifyEmailOtpCredentials{
		Email: req.Email,
		Token: req.Token,
		Type:  supabase.EmailOtpType("magiclink"),
	})
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: Invalid or expired OTP"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":      "Authentication successful",
		"access_token": res.AccessToken,
		"user":         res.User,
	})
}

// ----------------- USER REPORTS -----------------

func (h *Handler) CreateReport(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	profileID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: Invalid profile ID"})
	}

	var req model.CreateReportRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request: Invalid request payload"})
	}

	if req.Description == "" || req.Lat == 0 || req.Lng == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request: Description, latitude, and longitude are required"})
	}

	if err := h.Repo.CreateReport(profileID, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal Server Error: Failed to create report"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Report submitted successfully"})
}

func (h *Handler) GetMyReports(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	profileID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: Invalid profile ID"})
	}

	reports, err := h.Repo.GetReportsByProfileID(profileID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal Server Error: Failed to fetch reports"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": reports})
}

// ----------------- ADMIN -----------------

func (h *Handler) GetAllReports(c *fiber.Ctx) error {
	statusFilter := c.Query("status") // optional filter
	reports, err := h.Repo.GetAllReports(statusFilter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal Server Error: Failed to fetch reports"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": reports})
}

func (h *Handler) VerifyReport(c *fiber.Ctx) error {
	reportIDStr := c.Params("id")
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request: Invalid report ID format"})
	}

	if err := h.Repo.VerifyReport(reportID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal Server Error: Failed to verify report", "detail": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Report verified successfully. 10 points awarded to reporter."})
}

func (h *Handler) ResolveReport(c *fiber.Ctx) error {
	reportIDStr := c.Params("id")
	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request: Invalid report ID format"})
	}

	if err := h.Repo.ResolveReport(reportID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal Server Error: Failed to resolve report", "detail": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Report resolved successfully. 50 bonus points awarded to reporter."})
}

// ----------------- LEADERBOARD -----------------

func (h *Handler) GetLeaderboard(c *fiber.Ctx) error {
	profiles, err := h.Repo.GetLeaderboard()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal Server Error: Failed to fetch leaderboard"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": profiles})
}
