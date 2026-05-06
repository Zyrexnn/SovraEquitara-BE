package main

import (
	"fmt"

	"sovraequitara-be/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

// Utility: verifies that your JWT_SECRET can correctly sign/verify tokens
func main() {
	cfg := config.LoadConfig()

	// Create a test token
	claims := jwt.MapClaims{
		"sub":  "test-user-id",
		"role": "USER",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		fmt.Printf("Failed to sign token: %v\n", err)
		return
	}

	fmt.Printf("Test token: %s\n\n", tokenString)

	// Verify the token
	parsed, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWTSecret), nil
	})

	if err != nil {
		fmt.Printf("Verification failed: %v\n", err)
	} else if parsed.Valid {
		fmt.Println("JWT_SECRET is working correctly!")
		fmt.Printf("Secret used: %s\n", cfg.JWTSecret)
	}
}
