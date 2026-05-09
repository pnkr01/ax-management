package repository

import (
	models "ax-management/internal/model" // Matches your import
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// 1. ADDED GetTenantBySlug to the interface
type ManagementRepository interface {
	CreateTenant(tenant *models.Tenant) error
	GetTenantByID(id uuid.UUID) (*models.Tenant, error)
	GetTenantBySlug(slug string) (*models.Tenant, error)
	CreateApiKey(key *models.ApiKey) error
	GetFirstTenant() (*models.Tenant, error)
	GetKeysByTenant(tenantID uuid.UUID) ([]models.ApiKey, error)
	GetTeamMembers(tenantID uuid.UUID) ([]models.TeamMember, error)
	GetFunnelData(tenantID uuid.UUID, steps []string) ([]map[string]interface{}, error)
	GetOverviewKPIs(tenantID uuid.UUID) (map[string]interface{}, error)
	RevokeApiKey(tenantID, keyID uuid.UUID) (string, error)
}

type pgRepo struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) ManagementRepository {
	return &pgRepo{db: db}
}

func (r *pgRepo) CreateTenant(t *models.Tenant) error {
	return r.db.Create(t).Error
}

func (r *pgRepo) GetTenantByID(id uuid.UUID) (*models.Tenant, error) {
	var t models.Tenant
	err := r.db.First(&t, "id = ?", id).Error
	return &t, err
}

func (r *pgRepo) CreateApiKey(k *models.ApiKey) error {
	return r.db.Create(k).Error
}

// 2. FIXED: Changed receiver from PostgresRepository to pgRepo
func (r *pgRepo) GetTenantBySlug(slug string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.Where("slug = ?", slug).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *pgRepo) GetFirstTenant() (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *pgRepo) GetKeysByTenant(tenantID uuid.UUID) ([]models.ApiKey, error) {
	var keys []models.ApiKey
	err := r.db.Where("tenant_id = ?", tenantID).Order("created_at desc").Find(&keys).Error
	return keys, err
}

func (r *pgRepo) GetTeamMembers(tenantID uuid.UUID) ([]models.TeamMember, error) {
	var members []models.TeamMember
	err := r.db.Preload("User").Where("tenant_id = ?", tenantID).Order("created_at desc").Find(&members).Error
	return members, err
}

func (r *pgRepo) GetFunnelData(tenantID uuid.UUID, steps []string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := r.db.Model(&models.TelemetryEvent{}).
		Select("step, count(*) as count").
		Where("tenant_id = ? AND step IN ?", tenantID, steps).
		Group("step").
		Find(&results).Error
	return results, err
}

func (r *pgRepo) GetOverviewKPIs(tenantID uuid.UUID) (map[string]interface{}, error) {
	var totalInvocations int64
	var avgLatency float64
	var errorCount int64

	r.db.Model(&models.TelemetryEvent{}).Where("tenant_id = ?", tenantID).Count(&totalInvocations)
	r.db.Model(&models.TelemetryEvent{}).Where("tenant_id = ?", tenantID).Select("COALESCE(AVG(latency_ms), 0)").Scan(&avgLatency)
	r.db.Model(&models.TelemetryEvent{}).Where("tenant_id = ? AND is_error = ?", tenantID, true).Count(&errorCount)

	hallucinationRate := 0.0
	if totalInvocations > 0 {
		hallucinationRate = (float64(errorCount) / float64(totalInvocations)) * 100
	}

	return map[string]interface{}{
		"totalInvocations":  totalInvocations,
		"avgLatencyMs":      int(avgLatency),
		"hallucinationRate": fmt.Sprintf("%.2f", hallucinationRate),
		"trendData":         []map[string]interface{}{},
	}, nil
}

// UPDATED: Fetch the key first to grab the hash, then soft delete
func (r *pgRepo) RevokeApiKey(tenantID, keyID uuid.UUID) (string, error) {
	var apiKey models.ApiKey

	// 1. Fetch the key to get the KeyHash (required for Redis invalidation)
	if err := r.db.Where("id = ? AND tenant_id = ?", keyID, tenantID).First(&apiKey).Error; err != nil {
		return "", err
	}

	// 2. Soft delete: Keep record for audit logs but mark it inactive
	err := r.db.Model(&apiKey).Update("is_active", false).Error
	if err != nil {
		return "", err
	}

	return apiKey.KeyHash, nil
}
