package service

import (
	models "ax-management/internal/model"
	"ax-management/internal/repository"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type ManagementService struct {
	repo  repository.ManagementRepository
	cache *redis.Client
}

func NewManagementService(r repository.ManagementRepository, c *redis.Client) *ManagementService {
	return &ManagementService{repo: r, cache: c}
}

func (s *ManagementService) OnboardTenant(name, slug string) (*models.Tenant, error) {
	tenant := &models.Tenant{Name: name, Slug: slug}
	if err := s.repo.CreateTenant(tenant); err != nil {
		return nil, err
	}
	return tenant, nil
}

func (s *ManagementService) GenerateApiKey(tenantID uuid.UUID) (string, error) {
	tenant, err := s.repo.GetTenantByID(tenantID)
	if err != nil {
		return "", err
	}

	// 1. Generate Secure Key
	bytes := make([]byte, 32)
	rand.Read(bytes)
	rawKey := fmt.Sprintf("ax_live_%s", hex.EncodeToString(bytes))

	// 2. Hash for DB
	hash := sha256.Sum256([]byte(rawKey))
	hashedKey := hex.EncodeToString(hash[:])

	// 3. Persist to Postgres
	apiKey := &models.ApiKey{
		TenantID:  tenant.ID,
		KeyPrefix: "ax_live_",
		KeyHash:   hashedKey,
	}
	if err := s.repo.CreateApiKey(apiKey); err != nil {
		return "", err
	}

	// 4. Sync to Redis for the Data Plane
	redisKey := fmt.Sprintf("apikey:%s", rawKey)
	s.cache.Set(context.Background(), redisKey, tenant.Slug, 0)

	return rawKey, nil
}
