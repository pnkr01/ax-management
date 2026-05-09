package service

import (
	models "ax-management/internal/model"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

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

//// Ensure this method matches exactly what the Handler is calling
//func (s *ManagementService) CreateApiKey(tenantID uuid.UUID, name string) (*models.ApiKey, string, error) {
//	// 1. Generate a raw, secure 32-byte key
//	rawBytes := make([]byte, 32)
//	if _, err := rand.Read(rawBytes); err != nil {
//		return nil, "", err
//	}
//
//	// Format it for the user: ax_live_...
//	rawKey := "ax_live_" + hex.EncodeToString(rawBytes)
//
//	// 2. Hash it with SHA-256 for secure storage
//	hash := sha256.Sum256([]byte(rawKey))
//	hashedKey := hex.EncodeToString(hash[:])
//
//	// 3. Create the Database model
//	apiKey := &models.ApiKey{
//		TenantID: tenantID,
//		Name:     name,
//		KeyHash:  hashedKey,
//	}
//
//	if err := s.repo.CreateApiKey(apiKey); err != nil {
//		return nil, "", err
//	}
//
//	// Return both the DB model AND the raw string so the handler can show it to the user
//	return apiKey, rawKey, nil
//}

// -> ADD THIS BLOCK:
func (s *ManagementService) GetFirstTenant() (*models.Tenant, error) {
	return s.repo.GetFirstTenant()
}

// Add these at the bottom of the file

func (s *ManagementService) GetKeysByTenant(tenantID uuid.UUID) ([]models.ApiKey, error) {
	return s.repo.GetKeysByTenant(tenantID)
}

func (s *ManagementService) GetTeamMembers(tenantID uuid.UUID) ([]models.TeamMember, error) {
	return s.repo.GetTeamMembers(tenantID)
}

func (s *ManagementService) GetFunnelData(tenantID uuid.UUID, steps []string) ([]map[string]interface{}, error) {
	return s.repo.GetFunnelData(tenantID, steps)
}

func (s *ManagementService) GetOverviewKPIs(tenantID uuid.UUID) (map[string]interface{}, error) {
	return s.repo.GetOverviewKPIs(tenantID)
}

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

	// 4. NEW: Sync the valid key hash to Redis so the edge allows requests
	// Edge will check: GET apikey:{hashedKey}
	tenant, _ := s.repo.GetTenantByID(tenantID)
	redisKey := fmt.Sprintf("apikey:%s", hashedKey)

	// We use context.Background() here for the cache operation
	s.cache.Set(context.Background(), redisKey, tenant.Slug, 0)

	// Return both the DB model AND the raw string so the handler can show it to the user
	return apiKey, rawKey, nil
}

// UPDATED: Now takes context and applies the Redis tombstone
func (s *ManagementService) RevokeApiKey(ctx context.Context, tenantID, keyID uuid.UUID) error {
	// 1. Soft delete in DB and retrieve the hash
	keyHash, err := s.repo.RevokeApiKey(tenantID, keyID)
	if err != nil {
		return err
	}

	// 2. Apply the Tombstone in Redis
	redisKey := fmt.Sprintf("apikey:%s", keyHash)

	// Set to "revoked" with a 24-hour TTL. Edge nodes will see this and block requests instantly.
	err = s.cache.Set(ctx, redisKey, "revoked", 24*time.Hour).Err()
	if err != nil {
		// Log the error, but don't fail the API call since the DB was successfully updated
		fmt.Printf("Warning: Failed to invalidate redis cache for key %s: %v\n", keyID, err)
	}

	return nil
}
