package repository

import (
	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/ignis-runtime/ignis-wasmtime/internal/models"
)

// RuntimeRepository defines the interface for runtime persistence operations
type RuntimeRepository interface {
	Create(runtime *models.Runtime) error
	FindByID(id uuid.UUID) (*models.Runtime, error)
	FindByHash(hash string) (*models.Runtime, error)
	GetAll() ([]*models.Runtime, error)
	Update(runtime *models.Runtime) error
	Delete(id uuid.UUID) error
}

// runtimeRepository implements the RuntimeRepository interface
type runtimeRepository struct {
	db *gorm.DB
}

// NewRuntimeRepository creates a new instance of RuntimeRepository
func NewRuntimeRepository(db *gorm.DB) RuntimeRepository {
	return &runtimeRepository{
		db: db,
	}
}

// Create persists a new runtime to the database
func (r *runtimeRepository) Create(runtime *models.Runtime) error {
	return r.db.Create(runtime).Error
}

// FindByID retrieves a runtime by its ID
func (r *runtimeRepository) FindByID(id uuid.UUID) (*models.Runtime, error) {
	var runtime models.Runtime
	err := r.db.Where("id = ?", id).First(&runtime).Error
	if err != nil {
		return nil, err
	}
	return &runtime, nil
}

// FindByHash retrieves a runtime by its hash
func (r *runtimeRepository) FindByHash(hash string) (*models.Runtime, error) {
	var runtime models.Runtime
	err := r.db.Where("hash = ?", hash).First(&runtime).Error
	if err != nil {
		return nil, err
	}
	return &runtime, nil
}

// Update updates an existing runtime in the database
func (r *runtimeRepository) Update(runtime *models.Runtime) error {
	return r.db.Save(runtime).Error
}

// GetAll retrieves all runtimes from the database
func (r *runtimeRepository) GetAll() ([]*models.Runtime, error) {
	var runtimes []*models.Runtime
	err := r.db.Find(&runtimes).Error
	if err != nil {
		return nil, err
	}
	return runtimes, nil
}

// Delete removes a runtime from the database
func (r *runtimeRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Runtime{}, "id = ?", id).Error
}