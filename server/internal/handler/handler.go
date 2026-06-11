package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/niel/dbui/server/internal/connection"
	"github.com/niel/dbui/server/internal/models"
)

// Handler provides HTTP handlers for the API.
type Handler struct {
	manager *connection.Manager
}

// New creates a new Handler.
func New(manager *connection.Manager) *Handler {
	return &Handler{manager: manager}
}

// Routes returns a chi router with all API routes mounted.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Route("/api/connections", func(r chi.Router) {
		r.Get("/", h.ListConnections)
		r.Post("/", h.CreateConnection)
		r.Get("/{id}", h.GetConnection)
		r.Put("/{id}", h.UpdateConnection)
		r.Delete("/{id}", h.DeleteConnection)
		r.Post("/{id}/connect", h.Connect)
		r.Post("/{id}/disconnect", h.Disconnect)
		r.Post("/{id}/test", h.TestConnection)
	})

	r.Route("/api/db/{connID}", func(r chi.Router) {
		r.Get("/databases", h.ListDatabases)
		r.Post("/switch", h.SwitchDatabase)
		r.Get("/tables", h.ListTables)
		r.Post("/tables", h.CreateTable)
		r.Delete("/tables/{table}", h.DropTable)
		r.Get("/tables/{table}/describe", h.DescribeTable)
		r.Post("/tables/{table}/rename-column", h.RenameColumn)
		r.Get("/tables/{table}/indexes", h.ListIndexes)
		r.Post("/tables/{table}/read", h.ReadRows)
		r.Post("/tables/{table}/create", h.CreateRow)
		r.Post("/tables/{table}/update", h.UpdateRows)
		r.Post("/tables/{table}/delete", h.DeleteRows)
		r.Post("/query", h.RawQuery)
	})

	return r
}

func (h *Handler) ListConnections(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.manager.List())
}

func (h *Handler) CreateConnection(w http.ResponseWriter, r *http.Request) {
	var conn models.Connection
	if err := json.NewDecoder(r.Body).Decode(&conn); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if conn.Name == "" || conn.Type == "" || conn.Host == "" {
		writeError(w, http.StatusBadRequest, "name, type, and host are required")
		return
	}

	saved, err := h.manager.Save(conn)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (h *Handler) GetConnection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	conn, err := h.manager.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, conn)
}

func (h *Handler) UpdateConnection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	existing, err := h.manager.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var update models.Connection
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	update.ID = existing.ID
	update.Created = existing.Created
	saved, err := h.manager.Save(update)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (h *Handler) DeleteConnection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.manager.Remove(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := h.manager.Connect(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "connected"})
}

func (h *Handler) Disconnect(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.manager.Disconnect(r.Context(), id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected"})
}

func (h *Handler) TestConnection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx := r.Context()

	a, err := h.manager.Connect(ctx, id)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if err := a.Ping(ctx); err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (h *Handler) ListDatabases(w http.ResponseWriter, r *http.Request) {
	a, err := h.getActiveAdapter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	dbs, err := a.ListDatabases(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dbs)
}

func (h *Handler) SwitchDatabase(w http.ResponseWriter, r *http.Request) {
	connID := chi.URLParam(r, "connID")
	var body struct {
		Database string `json:"database"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Database == "" {
		writeError(w, http.StatusBadRequest, "database is required")
		return
	}
	if _, err := h.manager.SwitchDatabase(r.Context(), connID, body.Database); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "switched", "database": body.Database})
}

func (h *Handler) CreateTable(w http.ResponseWriter, r *http.Request) {
	a, err := h.getActiveAdapter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var spec models.TableSpec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := a.CreateTable(r.Context(), spec); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "created", "name": spec.Name})
}

func (h *Handler) DropTable(w http.ResponseWriter, r *http.Request) {
	a, err := h.getActiveAdapter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	table := chi.URLParam(r, "table")
	if err := a.DropTable(r.Context(), table); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "dropped", "name": table})
}

func (h *Handler) ListTables(w http.ResponseWriter, r *http.Request) {
	a, err := h.getActiveAdapter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tables, err := a.ListTables(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tables)
}

func (h *Handler) DescribeTable(w http.ResponseWriter, r *http.Request) {
	a, err := h.getActiveAdapter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	table := chi.URLParam(r, "table")
	columns, err := a.DescribeTable(r.Context(), table)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, columns)
}

// columnRenamer is implemented only by adapters that support DDL column renames.
type columnRenamer interface {
	RenameColumn(ctx context.Context, table, oldName, newName string) error
}

func (h *Handler) RenameColumn(w http.ResponseWriter, r *http.Request) {
	a, err := h.getActiveAdapter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	renamer, ok := a.(columnRenamer)
	if !ok {
		writeError(w, http.StatusBadRequest, "rename column is not supported for this database type")
		return
	}

	table := chi.URLParam(r, "table")
	var body struct {
		OldName string `json:"oldName"`
		NewName string `json:"newName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.OldName == "" || body.NewName == "" {
		writeError(w, http.StatusBadRequest, "oldName and newName are required")
		return
	}

	if err := renamer.RenameColumn(r.Context(), table, body.OldName, body.NewName); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "renamed", "table": table, "oldName": body.OldName, "newName": body.NewName})
}

func (h *Handler) ListIndexes(w http.ResponseWriter, r *http.Request) {
	a, err := h.getActiveAdapter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	table := chi.URLParam(r, "table")
	indexes, err := a.ListIndexes(r.Context(), table)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, indexes)
}

func (h *Handler) ReadRows(w http.ResponseWriter, r *http.Request) {
	h.doCRUD(w, r, func(ctx context.Context, a crudOp, req models.CRUDRequest) (*models.QueryResult, error) {
		return a.Read(ctx, req)
	})
}

func (h *Handler) CreateRow(w http.ResponseWriter, r *http.Request) {
	h.doCRUD(w, r, func(ctx context.Context, a crudOp, req models.CRUDRequest) (*models.QueryResult, error) {
		return a.Create(ctx, req)
	})
}

func (h *Handler) UpdateRows(w http.ResponseWriter, r *http.Request) {
	h.doCRUD(w, r, func(ctx context.Context, a crudOp, req models.CRUDRequest) (*models.QueryResult, error) {
		return a.Update(ctx, req)
	})
}

func (h *Handler) DeleteRows(w http.ResponseWriter, r *http.Request) {
	h.doCRUD(w, r, func(ctx context.Context, a crudOp, req models.CRUDRequest) (*models.QueryResult, error) {
		return a.Delete(ctx, req)
	})
}

func (h *Handler) RawQuery(w http.ResponseWriter, r *http.Request) {
	a, err := h.getActiveAdapter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var body struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	result, err := a.RawQuery(r.Context(), body.Query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// crudOp is the interface for CRUD adapter methods used in doCRUD.
type crudOp interface {
	Read(context.Context, models.CRUDRequest) (*models.QueryResult, error)
	Create(context.Context, models.CRUDRequest) (*models.QueryResult, error)
	Update(context.Context, models.CRUDRequest) (*models.QueryResult, error)
	Delete(context.Context, models.CRUDRequest) (*models.QueryResult, error)
}

func (h *Handler) doCRUD(w http.ResponseWriter, r *http.Request, op func(context.Context, crudOp, models.CRUDRequest) (*models.QueryResult, error)) {
	a, err := h.getActiveAdapter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req models.CRUDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Table = chi.URLParam(r, "table")

	result, err := op(r.Context(), a, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getActiveAdapter(r *http.Request) (interface {
	ListDatabases(context.Context) ([]string, error)
	ListTables(context.Context) ([]models.TableInfo, error)
	CreateTable(context.Context, models.TableSpec) error
	DropTable(context.Context, string) error
	DescribeTable(context.Context, string) ([]models.Column, error)
	ListIndexes(context.Context, string) ([]models.IndexInfo, error)
	Read(context.Context, models.CRUDRequest) (*models.QueryResult, error)
	Create(context.Context, models.CRUDRequest) (*models.QueryResult, error)
	Update(context.Context, models.CRUDRequest) (*models.QueryResult, error)
	Delete(context.Context, models.CRUDRequest) (*models.QueryResult, error)
	RawQuery(context.Context, string) (*models.QueryResult, error)
	Ping(context.Context) error
}, error) {
	connID := chi.URLParam(r, "connID")
	return h.manager.GetActive(connID)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
