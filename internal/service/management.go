package service

import (
	models "ax-management/internal/model"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	"github.com/google/uuid"
)

// Ensure this method matches exactly what the Handler is calling
func (s *ManagementService) CreateTenant(name, slug string) (*models.Tenant, error) {
	tenant := &models.Tenant{
		Name: name,
		Slug: slug,
	}
	if err := s.repo.CreateTenant(tenant); err != nil {
		return nil, err
	}
	return tenant, nil
}

// Ensure this method matches exactly what the Handler is calling
func (s *ManagementService) GetTenantBySlug(slug string) (*models.Tenant, error) {
	return s.repo.GetTenantBySlug(slug)
}

// Ensure this method matches exactly what the Handler is calling
func (s *ManagementService) CreateApiKey(tenantID uuid.UUID, name string) (*models.ApiKey, string, error) {
	// 1. Generate a raw, secure 32-byte key
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return nil, "", err
	}

	// Format it for the user: ax_live_...
	rawKey := "ax_live_" + hex.EncodeToString(rawBytes)

	// 2. Hash it with SHA-256 for secure storage
	hash := sha256.Sum256([]byte(rawKey))
	hashedKey := hex.EncodeToString(hash[:])

	// 3. Create the Database model
	apiKey := &models.ApiKey{
		TenantID: tenantID,
		Name:     name,
		KeyHash:  hashedKey,
	}

	if err := s.repo.CreateApiKey(apiKey); err != nil {
		return nil, "", err
	}

	// Return both the DB model AND the raw string so the handler can show it to the user
	return apiKey, rawKey, nil
}

// -> ADD THIS BLOCK:
func (s *ManagementService) GetFirstTenant() (*models.Tenant, error) {
	return s.repo.GetFirstTenant()
}
