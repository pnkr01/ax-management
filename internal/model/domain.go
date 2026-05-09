package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Tenant struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name      string    `gorm:"type:varchar(255);not null"`
	Slug      string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	PlanTier  string    `gorm:"type:varchar(50);default:'free'"`
	CreatedAt time.Time
}

type ApiKey struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TenantID  uuid.UUID `gorm:"type:uuid;index;not null"`
	KeyPrefix string    `gorm:"type:varchar(10)"`
	KeyHash   string    `gorm:"type:text;not null"`
	Name      string    `gorm:"type:varchar(100)"`
	IsActive  bool      `gorm:"default:true"`
	CreatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type CreateTenantRequest struct {
	Name string `json:"name" validate:"required"`
	Slug string `json:"slug" validate:"required"`
}

// Add to internal/models/domain.go
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	FullName     string    `gorm:"type:varchar(255);not null"`
	Email        string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash string    `gorm:"type:text;not null"` // Never store plain text
	CreatedAt    time.Time
}

// Request payloads from your Next.js forms
type RegisterRequest struct {
	FullName string `json:"fullName" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// Add this to your existing internal/model/domain.go

type TeamMember struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TenantID  uuid.UUID `gorm:"type:uuid;index;not null"`
	UserID    uuid.UUID `gorm:"type:uuid;index;not null"`
	Role      string    `gorm:"type:varchar(50);default:'VIEWER'"` // OWNER, ADMIN, VIEWER
	Status    string    `gorm:"type:varchar(50);default:'ACTIVE'"`
	CreatedAt time.Time

	// Relationships
	User User `gorm:"foreignKey:UserID"`
}

// TelemetryEvent represents a single span/event ingested from your AI Agents
type TelemetryEvent struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TenantID  uuid.UUID `gorm:"type:uuid;index;not null"`
	TraceID   string    `gorm:"type:varchar(255);index"`
	Step      string    `gorm:"type:varchar(255);index;not null"` // e.g., "voice_input_received"
	LatencyMs int       `gorm:"type:int;default:0"`
	IsError   bool      `gorm:"default:false"`
	CreatedAt time.Time `gorm:"index"`
}
