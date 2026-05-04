package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"sovraequitara-be/internal/config"
	"sovraequitara-be/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Protected validates JWT token.
// Supports both:
//   - HS256 tokens (admin custom tokens signed with JWT_SECRET)
//   - ES256 tokens (Supabase Auth tokens) validated via Supabase /auth/v1/user API
func Protected(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: Missing or invalid token format"})
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Step 1: Try parsing as HS256 (admin custom token)
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("not HMAC")
			}
			return []byte(cfg.SupabaseJWTSecret), nil
		})

		if err == nil && token.Valid {
			// HS256 token valid (admin token)
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: Invalid claims"})
			}
			userID, ok := claims["sub"].(string)
			if !ok {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: Missing user ID in token"})
			}
			c.Locals("userID", userID)
			return c.Next()
		}

		// Step 2: Not HS256 → validate via Supabase API (handles ES256, RS256, etc.)
		userID, apiErr := validateTokenViaSupabase(cfg.SupabaseURL, tokenString)
		if apiErr != nil {
			log.Printf("[AUTH ERROR] Supabase token validation failed: %v\n", apiErr)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized: Token tidak valid. Silakan login ulang.",
			})
		}

		c.Locals("userID", userID)
		return c.Next()
	}
}

// validateTokenViaSupabase calls Supabase /auth/v1/user with the access token.
// If Supabase returns the user, the token is valid.
func validateTokenViaSupabase(supabaseURL string, accessToken string) (string, error) {
	url := supabaseURL + "/auth/v1/user"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("apikey", accessToken) // Supabase also accepts this

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("supabase API call failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("supabase returned status %d: %s", resp.StatusCode, string(body))
	}

	var user struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &user); err != nil {
		return "", fmt.Errorf("failed to parse user response: %w", err)
	}

	if user.ID == "" {
		return "", fmt.Errorf("user ID empty in response")
	}

	return user.ID, nil
}

// AdminOnly checks if the authenticated user has 'admin' role in profiles table
func AdminOnly(repo repository.Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userIDStr := c.Locals("userID").(string)

		// Hardcoded Admin Bypass
		if userIDStr == "admin-ikhsan" {
			return c.Next()
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: Invalid User ID format"})
		}

		profile, err := repo.GetProfileByID(userID)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden: Profile not found"})
		}

		// Ensure strictly 'admin' role
		if profile.Role != "admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden: Admin privileges required"})
		}

		return c.Next()
	}
}
