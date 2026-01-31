package models

import (
	"time"

	"github.com/google/uuid"
)

// Runtime represents a deployed runtime in the system
type Runtime struct {
	ID            uuid.UUID   `json:"id" gorm:"type:uuid;primaryKey"`
	RuntimeType   string      `json:"runtime_type" gorm:"not null"`
	Hash          string      `json:"hash" gorm:"not null;index"`
	S3FilePath    string      `json:"s3_file_path" gorm:"column:s3_file_path;not null"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}