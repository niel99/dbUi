package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/niel/dbui/server/internal/adapter"
	"github.com/niel/dbui/server/internal/connection"
	"github.com/niel/dbui/server/internal/models"
	"github.com/niel/dbui/server/internal/registry"
)

type mockAdapter struct {
	tables  []models.TableInfo
	columns []models.Column
	result  *models.QueryResult
}

func (m *mockAdapter) Connect(ctx context.Context, conn models.Connection) error { return nil }
func (m *mockAdapter) Disconnect(ctx context.Context) error                      { return nil }
func (m *mockAdapter) Ping(ctx context.Context) error                            { return nil }
func (m *mockAdapter) ListTables(ctx context.Context) ([]models.TableInfo, error) {
	return m.tables, nil
}
func (m *mockAdapter) DescribeTable(ctx context.Context, table string) ([]models.Column, error) {
	return m.columns, nil
}
func (m *mockAdapter) Read(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	return m.result, nil
}
func (m *mockAdapter) Create(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	return m.result, nil
}
func (m *mockAdapter) Update(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	return m.result, nil
}
func (m *mockAdapter) Delete(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	return m.result, nil
}
func (m *mockAdapter) RawQuery(ctx context.Context, query string) (*models.QueryResult, error) {
	return m.result, nil
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

var testMock = &mockAdapter{
	tables: []models.TableInfo{
		{Name: "users", Schema: "public", TableType: "table"},
	},
	columns: []models.Column{
		{Name: "id", DataType: "integer", PrimaryKey: true},
		{Name: "name", DataType: "text", Nullable: true},
	},
	result: &models.QueryResult{
		Columns: []string{"id", "name"},
		Rows: []map[string]interface{}{
			{"id": 1, "name": "Alice"},
		},
		RowsAffected: 1,
	},
}

func setupTestHandler() (*Handler, http.Handler) {
	reg := registry.New()
	reg.Register(models.DBTypePostgres, func() adapter.DatabaseAdapter {
		return testMock
	})
	mgr := connection.NewManager(reg)
	h := New(mgr)
	return h, h.Routes()
}

func TestCreateConnection(t *testing.T) {
	_, router := setupTestHandler()

	body, _ := json.Marshal(models.Connection{
		Name: "Test PG",
		Type: models.DBTypePostgres,
		Host: "localhost",
		Port: 5432,
	})

	req := httptest.NewRequest("POST", "/api/connections", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var conn models.Connection
	json.NewDecoder(w.Body).Decode(&conn)
	if conn.ID == "" {
		t.Fatal("expected ID to be assigned")
	}
	if conn.Name != "Test PG" {
		t.Fatalf("expected name 'Test PG', got '%s'", conn.Name)
	}
}

func TestCreateConnection_Validation(t *testing.T) {
	_, router := setupTestHandler()

	// Missing required fields
	body, _ := json.Marshal(models.Connection{Name: "Test"})
	req := httptest.NewRequest("POST", "/api/connections", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListConnections(t *testing.T) {
	_, router := setupTestHandler()

	// Create a connection first
	body, _ := json.Marshal(models.Connection{
		Name: "Test PG",
		Type: models.DBTypePostgres,
		Host: "localhost",
	})
	req := httptest.NewRequest("POST", "/api/connections", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// List connections
	req = httptest.NewRequest("GET", "/api/connections", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var conns []models.Connection
	json.NewDecoder(w.Body).Decode(&conns)
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
}

func TestGetConnection(t *testing.T) {
	_, router := setupTestHandler()

	// Create
	body, _ := json.Marshal(models.Connection{
		Name: "Test PG",
		Type: models.DBTypePostgres,
		Host: "localhost",
	})
	req := httptest.NewRequest("POST", "/api/connections", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var created models.Connection
	json.NewDecoder(w.Body).Decode(&created)

	// Get
	req = httptest.NewRequest("GET", "/api/connections/"+created.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetConnection_NotFound(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest("GET", "/api/connections/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeleteConnection(t *testing.T) {
	_, router := setupTestHandler()

	// Create
	body, _ := json.Marshal(models.Connection{
		Name: "Test PG",
		Type: models.DBTypePostgres,
		Host: "localhost",
	})
	req := httptest.NewRequest("POST", "/api/connections", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var created models.Connection
	json.NewDecoder(w.Body).Decode(&created)

	// Delete
	req = httptest.NewRequest("DELETE", "/api/connections/"+created.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	// Verify gone
	req = httptest.NewRequest("GET", "/api/connections/"+created.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestConnectAndListTables(t *testing.T) {
	_, router := setupTestHandler()

	// Create connection
	body, _ := json.Marshal(models.Connection{
		Name: "Test PG",
		Type: models.DBTypePostgres,
		Host: "localhost",
	})
	req := httptest.NewRequest("POST", "/api/connections", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var created models.Connection
	json.NewDecoder(w.Body).Decode(&created)

	// Connect
	req = httptest.NewRequest("POST", "/api/connections/"+created.ID+"/connect", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// List tables
	req = httptest.NewRequest("GET", "/api/db/"+created.ID+"/tables", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var tables []models.TableInfo
	json.NewDecoder(w.Body).Decode(&tables)
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	if tables[0].Name != "users" {
		t.Fatalf("expected table 'users', got '%s'", tables[0].Name)
	}
}

func TestDescribeTable(t *testing.T) {
	_, router := setupTestHandler()

	body, _ := json.Marshal(models.Connection{Name: "PG", Type: models.DBTypePostgres, Host: "localhost"})
	req := httptest.NewRequest("POST", "/api/connections", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var c models.Connection
	json.NewDecoder(w.Body).Decode(&c)

	req = httptest.NewRequest("POST", "/api/connections/"+c.ID+"/connect", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	req = httptest.NewRequest("GET", "/api/db/"+c.ID+"/tables/users/describe", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var cols []models.Column
	json.NewDecoder(w.Body).Decode(&cols)
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(cols))
	}
}

func TestCRUD_Read(t *testing.T) {
	_, router := setupTestHandler()

	body, _ := json.Marshal(models.Connection{Name: "PG", Type: models.DBTypePostgres, Host: "localhost"})
	req := httptest.NewRequest("POST", "/api/connections", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var c models.Connection
	json.NewDecoder(w.Body).Decode(&c)

	req = httptest.NewRequest("POST", "/api/connections/"+c.ID+"/connect", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	readBody, _ := json.Marshal(models.CRUDRequest{Limit: 10})
	req = httptest.NewRequest("POST", "/api/db/"+c.ID+"/tables/users/read", bytes.NewReader(readBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result models.QueryResult
	json.NewDecoder(w.Body).Decode(&result)
	if len(result.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result.Rows))
	}
}

func TestRawQuery(t *testing.T) {
	_, router := setupTestHandler()

	body, _ := json.Marshal(models.Connection{Name: "PG", Type: models.DBTypePostgres, Host: "localhost"})
	req := httptest.NewRequest("POST", "/api/connections", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var c models.Connection
	json.NewDecoder(w.Body).Decode(&c)

	req = httptest.NewRequest("POST", "/api/connections/"+c.ID+"/connect", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	qBody, _ := json.Marshal(map[string]string{"query": "SELECT * FROM users"})
	req = httptest.NewRequest("POST", "/api/db/"+c.ID+"/query", bytes.NewReader(qBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRawQuery_EmptyBody(t *testing.T) {
	_, router := setupTestHandler()

	body, _ := json.Marshal(models.Connection{Name: "PG", Type: models.DBTypePostgres, Host: "localhost"})
	req := httptest.NewRequest("POST", "/api/connections", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var c models.Connection
	json.NewDecoder(w.Body).Decode(&c)

	req = httptest.NewRequest("POST", "/api/connections/"+c.ID+"/connect", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	qBody, _ := json.Marshal(map[string]string{"query": ""})
	req = httptest.NewRequest("POST", "/api/db/"+c.ID+"/query", bytes.NewReader(qBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTestConnection(t *testing.T) {
	_, router := setupTestHandler()

	body, _ := json.Marshal(models.Connection{Name: "PG", Type: models.DBTypePostgres, Host: "localhost"})
	req := httptest.NewRequest("POST", "/api/connections", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var c models.Connection
	json.NewDecoder(w.Body).Decode(&c)

	req = httptest.NewRequest("POST", "/api/connections/"+c.ID+"/test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if result["success"] != true {
		t.Fatalf("expected success true, got %v", result["success"])
	}
}

func TestDBEndpoints_NoActiveConnection(t *testing.T) {
	_, router := setupTestHandler()

	req := httptest.NewRequest("GET", "/api/db/nonexistent/tables", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
