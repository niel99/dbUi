package adapter

import (
	"context"

	"github.com/niel/dbui/server/internal/models"
)

// DatabaseAdapter is the interface that all database plugins must implement.
type DatabaseAdapter interface {
	// Connect establishes a connection to the database.
	Connect(ctx context.Context, conn models.Connection) error

	// Disconnect closes the database connection.
	Disconnect(ctx context.Context) error

	// Ping checks if the connection is alive.
	Ping(ctx context.Context) error

	// ListDatabases returns all databases/keyspaces on the server.
	ListDatabases(ctx context.Context) ([]string, error)

	// ListTables returns all tables/collections in the database.
	ListTables(ctx context.Context) ([]models.TableInfo, error)

	// CreateTable creates a new table/collection.
	CreateTable(ctx context.Context, spec models.TableSpec) error

	// DropTable drops a table/collection.
	DropTable(ctx context.Context, table string) error

	// DescribeTable returns column/field information for a table.
	DescribeTable(ctx context.Context, table string) ([]models.Column, error)

	// ListIndexes returns indexes defined on a table/collection.
	ListIndexes(ctx context.Context, table string) ([]models.IndexInfo, error)

	// Read fetches rows from a table with optional filtering.
	Read(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error)

	// Create inserts a new row/document.
	Create(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error)

	// Update modifies existing rows/documents matching the where clause.
	Update(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error)

	// Delete removes rows/documents matching the where clause.
	Delete(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error)

	// RawQuery executes a raw query string.
	RawQuery(ctx context.Context, query string) (*models.QueryResult, error)

	// Type returns the database type this adapter handles.
	Type() models.DBType
}
