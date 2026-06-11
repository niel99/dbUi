package connection

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/niel/dbui/server/internal/adapter"
	"github.com/niel/dbui/server/internal/models"
	"github.com/niel/dbui/server/internal/registry"
)

// Manager manages saved connections and active database sessions.
type Manager struct {
	mu          sync.RWMutex
	connections map[string]models.Connection
	active      map[string]adapter.DatabaseAdapter
	registry    *registry.Registry
	nextID      int
}

// NewManager creates a new connection manager.
func NewManager(reg *registry.Registry) *Manager {
	return &Manager{
		connections: make(map[string]models.Connection),
		active:      make(map[string]adapter.DatabaseAdapter),
		registry:    reg,
	}
}

// Save stores a connection configuration.
func (m *Manager) Save(conn models.Connection) (models.Connection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	if conn.ID == "" {
		m.nextID++
		conn.ID = fmt.Sprintf("conn_%d", m.nextID)
		conn.Created = now
	}
	conn.Updated = now
	m.connections[conn.ID] = conn
	return conn, nil
}

// Get returns a saved connection by ID.
func (m *Manager) Get(id string) (models.Connection, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, exists := m.connections[id]
	if !exists {
		return models.Connection{}, fmt.Errorf("connection not found: %s", id)
	}
	return conn, nil
}

// List returns all saved connections.
func (m *Manager) List() []models.Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]models.Connection, 0, len(m.connections))
	for _, conn := range m.connections {
		c := conn
		c.Password = "" // never expose passwords in list
		list = append(list, c)
	}
	return list
}

// Remove deletes a saved connection and disconnects if active.
func (m *Manager) Remove(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.connections[id]; !exists {
		return fmt.Errorf("connection not found: %s", id)
	}

	if a, active := m.active[id]; active {
		a.Disconnect(ctx)
		delete(m.active, id)
	}
	delete(m.connections, id)
	return nil
}

// Connect establishes a live database connection.
func (m *Manager) Connect(ctx context.Context, id string) (adapter.DatabaseAdapter, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, exists := m.connections[id]
	if !exists {
		return nil, fmt.Errorf("connection not found: %s", id)
	}

	// Return existing active connection
	if a, active := m.active[id]; active {
		if err := a.Ping(ctx); err == nil {
			return a, nil
		}
		// Stale connection, clean up
		a.Disconnect(ctx)
		delete(m.active, id)
	}

	a, err := m.registry.Get(conn.Type)
	if err != nil {
		return nil, err
	}

	if err := a.Connect(ctx, conn); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	m.active[id] = a
	return a, nil
}

// Disconnect closes an active connection.
func (m *Manager) Disconnect(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	a, active := m.active[id]
	if !active {
		return fmt.Errorf("no active connection: %s", id)
	}

	err := a.Disconnect(ctx)
	delete(m.active, id)
	return err
}

// SwitchDatabase updates the active connection to use a different database,
// reconnecting under the hood. The saved connection's Database field is updated.
func (m *Manager) SwitchDatabase(ctx context.Context, id, dbName string) (adapter.DatabaseAdapter, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, exists := m.connections[id]
	if !exists {
		return nil, fmt.Errorf("connection not found: %s", id)
	}

	if a, active := m.active[id]; active {
		a.Disconnect(ctx)
		delete(m.active, id)
	}

	conn.Database = dbName
	conn.Updated = time.Now()
	m.connections[id] = conn

	a, err := m.registry.Get(conn.Type)
	if err != nil {
		return nil, err
	}
	if err := a.Connect(ctx, conn); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	m.active[id] = a
	return a, nil
}

// GetActive returns the active adapter for a connection ID.
func (m *Manager) GetActive(id string) (adapter.DatabaseAdapter, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	a, active := m.active[id]
	if !active {
		return nil, fmt.Errorf("no active connection: %s", id)
	}
	return a, nil
}
