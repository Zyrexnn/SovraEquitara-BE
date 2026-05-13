package middleware

import (
	"fmt"
	"strings"

	"sovraequitara-be/internal/config"
	"sovraequitara-be/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Protected validates HS256 JWT tokens (self-issued, no Supabase dependency)
func Protected(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: Missing or invalid token format"})
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			secret := cfg.JWTSecret
			if len(secret) > 5 {
				fmt.Printf("JWT Debug - Verifying with secret: %s...\n", secret[:5])
			} else {
				fmt.Printf("JWT Debug - Verifying with short secret\n")
			}	
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			fmt.Printf("JWT Debug - Token: [%s], Error: %v\n", tokenString, err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized: Token tidak valid.",
				"debug_error": fmt.Sprintf("%v", err),
			})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: Invalid claims"})
		}

		userID, ok := claims["sub"].(string)
		if !ok || userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: Missing user ID in token"})
		}

		c.Locals("userID", userID)
		return c.Next()
	}
}

// AdminOnly checks if the authenticated user has 'admin' role in profiles table
func AdminOnly(repo repository.Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userIDStr := c.Locals("userID").(string)

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: Invalid User ID format"})
		}

		profile, err := repo.GetProfileByID(userID)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden: Profile not found"})
		}

		// Ensure strictly 'admin' or 'super_admin' role
		if profile.Role != "admin" && profile.Role != "super_admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden: Admin privileges required"})
		}

		return c.Next()
	}
}

// SuperAdminOnly checks if the authenticated user has 'super_admin' role in profiles table
func SuperAdminOnly(repo repository.Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userIDStr := c.Locals("userID").(string)

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: Invalid User ID format"})
		}

		profile, err := repo.GetProfileByID(userID)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden: Profile not found"})
		}

		// Ensure strictly 'super_admin' role
		if profile.Role != "super_admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden: Super Admin privileges required"})
		}

		return c.Next()
	}
}
