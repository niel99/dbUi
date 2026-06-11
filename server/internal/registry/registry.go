package registry

import (
	"fmt"
	"sync"

	"github.com/niel/dbui/server/internal/adapter"
	"github.com/niel/dbui/server/internal/models"
)

// AdapterFactory is a function that creates a new adapter instance.
type AdapterFactory func() adapter.DatabaseAdapter

// Registry manages available database adapter plugins.
type Registry struct {
	mu       sync.RWMutex
	adapters map[models.DBType]AdapterFactory
}

// New creates a new plugin registry.
func New() *Registry {
	return &Registry{
		adapters: make(map[models.DBType]AdapterFactory),
	}
}

// Register adds an adapter factory for a database type.
func (r *Registry) Register(dbType models.DBType, factory AdapterFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.adapters[dbType]; exists {
		return fmt.Errorf("adapter already registered for type: %s", dbType)
	}

	r.adapters[dbType] = factory
	return nil
}

// Get returns a new adapter instance for the given database type.
func (r *Registry) Get(dbType models.DBType) (adapter.DatabaseAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.adapters[dbType]
	if !exists {
		return nil, fmt.Errorf("no adapter registered for type: %s", dbType)
	}

	return factory(), nil
}

// ListTypes returns all registered database types.
func (r *Registry) ListTypes() []models.DBType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]models.DBType, 0, len(r.adapters))
	for t := range r.adapters {
		types = append(types, t)
	}
	return types
}
