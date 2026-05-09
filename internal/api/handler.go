package api

import (
	models "ax-management/internal/model"
	"ax-management/internal/service"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	svc     *service.ManagementService
	authSvc *service.AuthService
	isProd  bool
}

func NewHandler(s *service.ManagementService, auth *service.AuthService, isProd bool) *Handler {
	return &Handler{svc: s, authSvc: auth, isProd: isProd}
}

// ==========================================
// AUTHENTICATION ROUTES
// ==========================================

func (h *Handler) Register(c *fiber.Ctx) error {
	var req models.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	user, err := h.authSvc.RegisterUser(req)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	token, err := h.authSvc.GenerateJWT(user.ID, user.Email)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate session"})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "AX_SESSION",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
		Secure:   h.isProd,
		SameSite: "Lax",
		Path:     "/",
	})

	return c.Status(201).JSON(fiber.Map{"message": "Registration successful", "user_id": user.ID})
}

//func (h *Handler) Login(c *fiber.Ctx) error {
//	var req models.LoginRequest
//	if err := c.BodyParser(&req); err != nil {
//		return c.Status(400).JSON(fiber.Map{"error": "Invalid request payload"})
//	}
//
//	user, err := h.authSvc.AuthenticateUser(req.Email, req.Password)
//	if err != nil {
//		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
//	}
//
//	token, err := h.authSvc.GenerateJWT(user.ID, user.Email)
//	if err != nil {
//		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate session"})
//	}
//
//	c.Cookie(&fiber.Cookie{
//		Name:     "AX_SESSION",
//		Value:    token,
//		Expires:  time.Now().Add(24 * time.Hour),
//		HTTPOnly: true,
//		Secure:   h.isProd,
//		SameSite: "Lax",
//		Path:     "/",
//	})
//
//	return c.Status(200).JSON(fiber.Map{"message": "Login successful", "user_id": user.ID})
//}

// ==========================================
// TENANT & WORKSPACE ROUTES
// ==========================================

func (h *Handler) CreateTenant(c *fiber.Ctx) error {
	// Maps to the Next.js Workspace Setup form
	var req struct {
		Name     string `json:"name"`
		Slug     string `json:"slug"`
		Industry string `json:"industry"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid payload"})
	}

	tenant, err := h.svc.CreateTenant(req.Name, req.Slug)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(tenant)
}

// ==========================================
// API KEY MANAGEMENT
// ==========================================

func (h *Handler) CreateKey(c *fiber.Ctx) error {
	slug := c.Params("slug")

	if slug == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "CRITICAL: Fiber failed to extract the slug from the URL.",
		})
	}

	// 1. Define the exact struct we expect from Next.js
	var req struct {
		Name string `json:"name"`
	}

	// 2. Parse the body and return EXACT error if it fails
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid payload format. Expected JSON: {\"name\": \"...\"}",
		})
	}

	// 3. Prevent empty names
	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "The API Key name cannot be empty. Check your frontend form fields.",
		})
	}

	// 4. Look up Tenant UUID using the slug
	tenant, err := h.svc.GetTenantBySlug(slug)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Workspace not found"})
	}

	// 5. Generate Key
	_, rawKey, err := h.svc.CreateApiKey(tenant.ID, req.Name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate API Key"})
	}

	return c.Status(201).JSON(fiber.Map{
		"message": "Key created successfully",
		"api_key": rawKey,
	})
}

// --- UPDATED LOGIN HANDLER ---
func (h *Handler) Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	user, err := h.authSvc.AuthenticateUser(req.Email, req.Password)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}

	// SMART ROUTING CHECK: Check if any workspace exists
	tenantSlug := ""
	tenant, err := h.svc.GetFirstTenant()
	if err == nil && tenant != nil {
		tenantSlug = tenant.Slug // Grab the slug (e.g., "fid")
	}

	token, err := h.authSvc.GenerateJWT(user.ID, user.Email)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate session"})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "AX_SESSION",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
		Secure:   h.isProd,
		SameSite: "Lax",
		Path:     "/",
	})

	// Return the slug back to Next.js
	return c.Status(200).JSON(fiber.Map{
		"message":     "Login successful",
		"tenant_slug": tenantSlug,
	})
}

// --- NEW LOGOUT HANDLER ---
func (h *Handler) Logout(c *fiber.Ctx) error {
	// Overwrite the cookie with an empty value and an expiration time in the past
	c.Cookie(&fiber.Cookie{
		Name:     "AX_SESSION",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour), // Expired!
		HTTPOnly: true,
		Secure:   h.isProd,
		SameSite: "Lax",
		Path:     "/",
	})

	return c.Status(200).JSON(fiber.Map{"message": "Logged out successfully"})
}
