package api

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware extracts the AX_SESSION cookie and injects the user ID into the context
func AuthMiddleware(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Get the HTTP-Only cookie
		tokenString := c.Cookies("AX_SESSION")

		// Fallback for API clients using Authorization header
		if tokenString == "" {
			authHeader := c.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: No session token found"})
		}

		// 2. Parse and Validate JWT
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: Invalid or expired session"})
		}

		// 3. Inject claims into Fiber context for downstream handlers
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: Invalid token claims"})
		}

		c.Locals("userID", claims["sub"])
		c.Locals("userEmail", claims["email"])

		return c.Next()
	}
}
