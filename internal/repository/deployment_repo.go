package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ignis-runtime/ignis-wasmtime/internal/models"
)

// DeploymentRepository defines the interface for runtime persistence operations
type DeploymentRepository interface {
	Create(ctx context.Context, runtime *models.Runtime) (*models.Runtime, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.Runtime, error)
	FindByHash(ctx context.Context, hash string) (*models.Runtime, error)
	GetAll(ctx context.Context) ([]*models.Runtime, error)
	Update(ctx context.Context, runtime *models.Runtime) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type deploymentRepository struct {
	db *gorm.DB
}

func NewDeploymentRepository(db *gorm.DB) DeploymentRepository {
	return &deploymentRepository{
		db: db,
	}
}

// Create persists a new runtime and returns the created record
func (r *deploymentRepository) Create(ctx context.Context, runtime *models.Runtime) (*models.Runtime, error) {
	// GORM's Create method updates the 'runtime' pointer with DB-generated fields
	err := r.db.WithContext(ctx).Create(runtime).Error
	if err != nil {
		return nil, err
	}
	return runtime, nil
}

func (r *deploymentRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Runtime, error) {
	var runtime models.Runtime
	err := r.db.WithContext(ctx).First(&runtime, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &runtime, nil
}

func (r *deploymentRepository) FindByHash(ctx context.Context, hash string) (*models.Runtime, error) {
	var runtime models.Runtime
	err := r.db.WithContext(ctx).First(&runtime, "hash = ?", hash).Error
	if err != nil {
		return nil, err
	}
	return &runtime, nil
}

func (r *deploymentRepository) Update(ctx context.Context, runtime *models.Runtime) error {
	return r.db.WithContext(ctx).Save(runtime).Error
}

func (r *deploymentRepository) GetAll(ctx context.Context) ([]*models.Runtime, error) {
	var runtimes []*models.Runtime
	err := r.db.WithContext(ctx).Find(&runtimes).Error
	if err != nil {
		return nil, err
	}
	return runtimes, nil
}

func (r *deploymentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.Runtime{}, "id = ?", id).Error
}
