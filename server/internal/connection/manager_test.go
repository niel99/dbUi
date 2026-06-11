package connection

import (
	"context"
	"fmt"
	"testing"

	"github.com/niel/dbui/server/internal/adapter"
	"github.com/niel/dbui/server/internal/models"
	"github.com/niel/dbui/server/internal/registry"
)

type mockAdapter struct {
	connected bool
	pingErr   error
}

func (m *mockAdapter) Connect(ctx context.Context, conn models.Connection) error {
	m.connected = true
	return nil
}
func (m *mockAdapter) Disconnect(ctx context.Context) error {
	m.connected = false
	return nil
}
func (m *mockAdapter) Ping(ctx context.Context) error { return m.pingErr }
func (m *mockAdapter) ListTables(ctx context.Context) ([]models.TableInfo, error) {
	return nil, nil
}
func (m *mockAdapter) DescribeTable(ctx context.Context, table string) ([]models.Column, error) {
	return nil, nil
}
func (m *mockAdapter) Read(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	return nil, nil
}
func (m *mockAdapter) Create(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	return nil, nil
}
func (m *mockAdapter) Update(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	return nil, nil
}
func (m *mockAdapter) Delete(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	return nil, nil
}
func (m *mockAdapter) RawQuery(ctx context.Context, query string) (*models.QueryResult, error) {
	return nil, nil
}
func (m *mockAdapter) Type() models.DBType                                 { return models.DBTypePostgres }
func (m *mockAdapter) ListDatabases(ctx context.Context) ([]string, error) { return nil, nil }
func (m *mockAdapter) CreateTable(ctx context.Context, s models.TableSpec) error {
	return nil
}
func (m *mockAdapter) DropTable(ctx context.Context, table string) error { return nil }
func (m *mockAdapter) ListIndexes(ctx context.Context, table string) ([]models.IndexInfo, error) {
	return nil, nil
}

func setupManager() *Manager {
	reg := registry.New()
	reg.Register(models.DBTypePostgres, func() adapter.DatabaseAdapter {
		return &mockAdapter{}
	})
	return NewManager(reg)
}

func TestManager_SaveAndGet(t *testing.T) {
	m := setupManager()

	conn := models.Connection{
		Name:     "Test PG",
		Type:     models.DBTypePostgres,
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
	}

	saved, err := m.Save(conn)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if saved.ID == "" {
		t.Fatal("expected ID to be assigned")
	}
	if saved.Created.IsZero() {
		t.Fatal("expected Created to be set")
	}

	got, err := m.Get(saved.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Name != "Test PG" {
		t.Fatalf("expected name 'Test PG', got '%s'", got.Name)
	}
}

func TestManager_Get_NotFound(t *testing.T) {
	m := setupManager()

	_, err := m.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent connection")
	}
}

func TestManager_List_HidesPasswords(t *testing.T) {
	m := setupManager()

	m.Save(models.Connection{
		Name:     "PG1",
		Type:     models.DBTypePostgres,
		Password: "secret123",
	})

	list := m.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(list))
	}
	if list[0].Password != "" {
		t.Fatal("expected password to be hidden in list")
	}
}

func TestManager_Remove(t *testing.T) {
	m := setupManager()
	ctx := context.Background()

	saved, _ := m.Save(models.Connection{Name: "PG1", Type: models.DBTypePostgres})

	err := m.Remove(ctx, saved.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = m.Get(saved.ID)
	if err == nil {
		t.Fatal("expected error after removal")
	}
}

func TestManager_Remove_NotFound(t *testing.T) {
	m := setupManager()
	ctx := context.Background()

	err := m.Remove(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent connection")
	}
}

func TestManager_ConnectAndDisconnect(t *testing.T) {
	m := setupManager()
	ctx := context.Background()

	saved, _ := m.Save(models.Connection{
		Name: "PG1",
		Type: models.DBTypePostgres,
		Host: "localhost",
	})

	a, err := m.Connect(ctx, saved.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if a == nil {
		t.Fatal("expected adapter, got nil")
	}

	// Connect again should return same adapter (reuse)
	a2, err := m.Connect(ctx, saved.ID)
	if err != nil {
		t.Fatalf("expected no error on reconnect, got %v", err)
	}
	if a != a2 {
		t.Fatal("expected same adapter instance on reconnect")
	}

	err = m.Disconnect(ctx, saved.ID)
	if err != nil {
		t.Fatalf("expected no error on disconnect, got %v", err)
	}
}

func TestManager_Connect_NotFound(t *testing.T) {
	m := setupManager()
	ctx := context.Background()

	_, err := m.Connect(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent connection")
	}
}

func TestManager_Connect_StaleConnection(t *testing.T) {
	m := setupManager()
	ctx := context.Background()

	saved, _ := m.Save(models.Connection{
		Name: "PG1",
		Type: models.DBTypePostgres,
	})

	// First connect
	a1, _ := m.Connect(ctx, saved.ID)
	mock1 := a1.(*mockAdapter)

	// Simulate stale connection
	mock1.pingErr = fmt.Errorf("connection lost")

	// Reconnect should create new adapter
	a2, err := m.Connect(ctx, saved.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if a1 == a2 {
		t.Fatal("expected new adapter instance after stale connection")
	}
}

func TestManager_Disconnect_NoActive(t *testing.T) {
	m := setupManager()
	ctx := context.Background()

	err := m.Disconnect(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for no active connection")
	}
}

func TestManager_GetActive(t *testing.T) {
	m := setupManager()
	ctx := context.Background()

	saved, _ := m.Save(models.Connection{Name: "PG1", Type: models.DBTypePostgres})

	_, err := m.GetActive(saved.ID)
	if err == nil {
		t.Fatal("expected error before connecting")
	}

	m.Connect(ctx, saved.ID)

	a, err := m.GetActive(saved.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if a == nil {
		t.Fatal("expected adapter, got nil")
	}
}
