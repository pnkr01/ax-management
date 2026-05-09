package repository

import (
	models "ax-management/internal/model" // Matches your import

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// 1. ADDED GetTenantBySlug to the interface
type ManagementRepository interface {
	CreateTenant(tenant *models.Tenant) error
	GetTenantByID(id uuid.UUID) (*models.Tenant, error)
	GetTenantBySlug(slug string) (*models.Tenant, error)
	CreateApiKey(key *models.ApiKey) error
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
