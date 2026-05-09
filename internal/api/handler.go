package api

import (
	models "ax-management/internal/model"
	"ax-management/internal/service"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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

// ==========================================
// REAL DATA HANDLERS (Paste at bottom of handler.go)
// ==========================================

func (h *Handler) GetMe(c *fiber.Ctx) error {
	// Extract the user ID injected by the JWT Middleware
	userIDStr := c.Locals("userID").(string)

	// Note: If you don't have GetUserByID in your auth service yet,
	// you can mock this response for now or add the DB lookup to authSvc.
	return c.JSON(fiber.Map{
		"id":    userIDStr,
		"name":  "Enterprise User", // Fallback if DB lookup isn't wired yet
		"email": c.Locals("userEmail"),
		"role":  "OWNER",
	})
}

func (h *Handler) GetKeys(c *fiber.Ctx) error {
	slug := c.Params("slug")
	tenant, err := h.svc.GetTenantBySlug(slug)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Tenant not found"})
	}

	keys, err := h.svc.GetKeysByTenant(tenant.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch keys"})
	}

	// Format for UI
	var formattedKeys []fiber.Map
	for _, k := range keys {
		status := "active"
		if !k.IsActive {
			status = "revoked"
		}
		formattedKeys = append(formattedKeys, fiber.Map{
			"id":           k.ID,
			"name":         k.Name,
			"prefix":       k.KeyPrefix,
			"created_at":   k.CreatedAt.Format(time.RFC3339),
			"status":       status,
			"last_used_at": nil,
		})
	}

	return c.JSON(fiber.Map{"keys": formattedKeys})
}

func (h *Handler) GetTeamMembers(c *fiber.Ctx) error {
	slug := c.Params("slug")
	tenant, err := h.svc.GetTenantBySlug(slug)
	if err != nil || tenant == nil {
		return c.Status(404).JSON(fiber.Map{"error": "Tenant not found"})
	}

	members, err := h.svc.GetTeamMembers(tenant.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch team"})
	}

	var formattedMembers []fiber.Map
	for _, m := range members {
		formattedMembers = append(formattedMembers, fiber.Map{
			"id":        m.ID,
			"name":      m.User.FullName,
			"email":     m.User.Email,
			"role":      m.Role,
			"status":    m.Status,
			"joined_at": m.CreatedAt.Format(time.RFC3339),
		})
	}

	return c.JSON(fiber.Map{"members": formattedMembers})
}

func (h *Handler) GetAnalyticsOverview(c *fiber.Ctx) error {
	slug := c.Params("slug")
	tenant, _ := h.svc.GetTenantBySlug(slug)

	kpis, err := h.svc.GetOverviewKPIs(tenant.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to calculate KPIs"})
	}

	return c.JSON(kpis)
}

func (h *Handler) RunFunnelQuery(c *fiber.Ctx) error {
	slug := c.Params("slug")
	tenant, _ := h.svc.GetTenantBySlug(slug)

	var req struct {
		Steps []string `json:"steps"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	results, err := h.svc.GetFunnelData(tenant.ID, req.Steps)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to run query"})
	}

	// Apply UI colors based on step
	colors := []string{"#3b82f6", "#60a5fa", "#93c5fd", "#bfdbfe"}
	for i, r := range results {
		colorIdx := i % len(colors)
		r["color"] = colors[colorIdx]
	}

	return c.JSON(fiber.Map{"results": results})
}

// UPDATED: Pass c.Context() to the service layer
func (h *Handler) RevokeKey(c *fiber.Ctx) error {
	slug := c.Params("slug")
	keyIDStr := c.Params("keyId")

	// Parse the UUID from the URL
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid Key ID format"})
	}

	// Validate the Tenant
	tenant, err := h.svc.GetTenantBySlug(slug)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Tenant not found"})
	}

	// Execute Revocation (Passing Fiber's context into the service)
	if err := h.svc.RevokeApiKey(c.Context(), tenant.ID, keyID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to revoke key"})
	}

	return c.JSON(fiber.Map{"message": "Key revoked successfully"})
}
