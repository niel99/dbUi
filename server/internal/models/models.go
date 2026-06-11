package models

import "time"

// DBType represents a supported database type.
type DBType string

const (
	DBTypePostgres  DBType = "postgres"
	DBTypeMongoDB   DBType = "mongodb"
	DBTypeCassandra DBType = "cassandra"
	DBTypeScyllaDB  DBType = "scylladb"
)

// Connection holds the configuration for a database connection.
type Connection struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Type     DBType            `json:"type"`
	Host     string            `json:"host"`
	Port     int               `json:"port"`
	Username string            `json:"username"`
	Password string            `json:"password,omitempty"`
	Database string            `json:"database"`
	Options  map[string]string `json:"options,omitempty"`
	Created  time.Time         `json:"created"`
	Updated  time.Time         `json:"updated"`
}

// TableInfo describes a table or collection.
type TableInfo struct {
	Name      string `json:"name"`
	Schema    string `json:"schema,omitempty"`    // SQL databases
	Keyspace  string `json:"keyspace,omitempty"`  // Cassandra/ScyllaDB
	RowCount  int64  `json:"rowCount,omitempty"`
	TableType string `json:"tableType,omitempty"` // "table", "view", "collection"
}

// Column describes a column or field.
type Column struct {
	Name       string `json:"name"`
	DataType   string `json:"dataType"`
	Nullable   bool   `json:"nullable"`
	PrimaryKey bool   `json:"primaryKey"`
}

// QueryResult holds the result of a query or CRUD operation.
type QueryResult struct {
	Columns      []string                 `json:"columns,omitempty"`
	Rows         []map[string]interface{} `json:"rows"`
	RowsAffected int64                    `json:"rowsAffected"`
	Error        string                   `json:"error,omitempty"`
}

// CRUDRequest represents a create/update/delete request.
type CRUDRequest struct {
	Table    string                 `json:"table"`
	Schema   string                 `json:"schema,omitempty"`
	Keyspace string                 `json:"keyspace,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Where    map[string]interface{} `json:"where,omitempty"`
	Limit    int                    `json:"limit,omitempty"`
	Offset   int                    `json:"offset,omitempty"`
	OrderBy  string                 `json:"orderBy,omitempty"`
	OrderDir string                 `json:"orderDir,omitempty"` // "asc" or "desc"
}

// ColumnSpec describes a column to be created.
type ColumnSpec struct {
	Name       string  `json:"name"`
	DataType   string  `json:"dataType"`
	Nullable   bool    `json:"nullable"`
	PrimaryKey bool    `json:"primaryKey"`
	Unique     bool    `json:"unique,omitempty"`
	Default    string  `json:"default,omitempty"`
	MinLength  *int    `json:"minLength,omitempty"`
	MaxLength  *int    `json:"maxLength,omitempty"`
	Pattern    string  `json:"pattern,omitempty"`
}

// IndexInfo describes an index on a table/collection.
type IndexInfo struct {
	Name       string   `json:"name"`
	Columns    []string `json:"columns,omitempty"`
	Unique     bool     `json:"unique,omitempty"`
	Primary    bool     `json:"primary,omitempty"`
	Definition string   `json:"definition,omitempty"`
}

// TableSpec describes a table/collection to be created.
type TableSpec struct {
	Name    string       `json:"name"`
	Schema  string       `json:"schema,omitempty"`
	Columns []ColumnSpec `json:"columns,omitempty"`
}
