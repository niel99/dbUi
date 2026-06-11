package registry

import (
	"context"
	"testing"

	"github.com/niel/dbui/server/internal/adapter"
	"github.com/niel/dbui/server/internal/models"
)

// mockAdapter implements adapter.DatabaseAdapter for testing.
type mockAdapter struct {
	dbType models.DBType
}

func (m *mockAdapter) Connect(ctx context.Context, conn models.Connection) error   { return nil }
func (m *mockAdapter) Disconnect(ctx context.Context) error                        { return nil }
func (m *mockAdapter) Ping(ctx context.Context) error                              { return nil }
func (m *mockAdapter) ListTables(ctx context.Context) ([]models.TableInfo, error)  { return nil, nil }
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
func (m *mockAdapter) Type() models.DBType                                 { return m.dbType }
func (m *mockAdapter) ListDatabases(ctx context.Context) ([]string, error) { return nil, nil }
func (m *mockAdapter) CreateTable(ctx context.Context, s models.TableSpec) error {
	return nil
}
func (m *mockAdapter) DropTable(ctx context.Context, table string) error { return nil }
func (m *mockAdapter) ListIndexes(ctx context.Context, table string) ([]models.IndexInfo, error) {
	return nil, nil
}

func newMockFactory(dbType models.DBType) AdapterFactory {
	return func() adapter.DatabaseAdapter {
		return &mockAdapter{dbType: dbType}
	}
}

func TestRegistry_Register(t *testing.T) {
	r := New()

	err := r.Register(models.DBTypePostgres, newMockFactory(models.DBTypePostgres))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Registering same type again should fail
	err = r.Register(models.DBTypePostgres, newMockFactory(models.DBTypePostgres))
	if err == nil {
		t.Fatal("expected error for duplicate registration, got nil")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := New()
	r.Register(models.DBTypePostgres, newMockFactory(models.DBTypePostgres))

	a, err := r.Get(models.DBTypePostgres)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if a.Type() != models.DBTypePostgres {
		t.Fatalf("expected type %s, got %s", models.DBTypePostgres, a.Type())
	}

	// Getting unregistered type should fail
	_, err = r.Get(models.DBTypeMongoDB)
	if err == nil {
		t.Fatal("expected error for unregistered type, got nil")
	}
}

func TestRegistry_ListTypes(t *testing.T) {
	r := New()
	r.Register(models.DBTypePostgres, newMockFactory(models.DBTypePostgres))
	r.Register(models.DBTypeMongoDB, newMockFactory(models.DBTypeMongoDB))
	r.Register(models.DBTypeCassandra, newMockFactory(models.DBTypeCassandra))

	types := r.ListTypes()
	if len(types) != 3 {
		t.Fatalf("expected 3 types, got %d", len(types))
	}

	typeSet := make(map[models.DBType]bool)
	for _, typ := range types {
		typeSet[typ] = true
	}

	for _, expected := range []models.DBType{models.DBTypePostgres, models.DBTypeMongoDB, models.DBTypeCassandra} {
		if !typeSet[expected] {
			t.Errorf("expected type %s in list", expected)
		}
	}
}

func TestRegistry_Get_ReturnsNewInstance(t *testing.T) {
	r := New()
	r.Register(models.DBTypePostgres, newMockFactory(models.DBTypePostgres))

	a1, _ := r.Get(models.DBTypePostgres)
	a2, _ := r.Get(models.DBTypePostgres)

	if a1 == a2 {
		t.Fatal("expected different instances from factory, got same pointer")
	}
}
