package cassandra

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/niel/dbui/server/internal/models"
)

// Adapter implements DatabaseAdapter for Cassandra and ScyllaDB.
type Adapter struct {
	session  *gocql.Session
	keyspace string
	dbType   models.DBType
}

func New(dbType models.DBType) *Adapter {
	return &Adapter{dbType: dbType}
}

func (a *Adapter) Type() models.DBType {
	return a.dbType
}

func (a *Adapter) Connect(ctx context.Context, conn models.Connection) error {
	cluster := gocql.NewCluster(conn.Host)
	cluster.Port = conn.Port
	cluster.Keyspace = conn.Database
	cluster.Consistency = gocql.Quorum
	cluster.Timeout = 10 * time.Second

	if conn.Username != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: conn.Username,
			Password: conn.Password,
		}
	}

	session, err := cluster.CreateSession()
	if err != nil {
		return err
	}

	a.session = session
	a.keyspace = conn.Database
	return nil
}

func (a *Adapter) Disconnect(ctx context.Context) error {
	if a.session != nil {
		a.session.Close()
		a.session = nil
	}
	return nil
}

func (a *Adapter) Ping(ctx context.Context) error {
	if a.session == nil {
		return fmt.Errorf("not connected")
	}
	// Simple query to verify connection
	var now time.Time
	return a.session.Query("SELECT now() FROM system.local").WithContext(ctx).Scan(&now)
}

func (a *Adapter) ListDatabases(ctx context.Context) ([]string, error) {
	iter := a.session.Query(`SELECT keyspace_name FROM system_schema.keyspaces`).WithContext(ctx).Iter()
	keyspaces := make([]string, 0)
	var name string
	for iter.Scan(&name) {
		keyspaces = append(keyspaces, name)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return keyspaces, nil
}

func (a *Adapter) CreateTable(ctx context.Context, spec models.TableSpec) error {
	if spec.Name == "" || len(spec.Columns) == 0 {
		return fmt.Errorf("table name and at least one column are required")
	}

	colDefs := make([]string, 0, len(spec.Columns))
	pkCols := make([]string, 0)
	for _, c := range spec.Columns {
		if c.Name == "" || c.DataType == "" {
			return fmt.Errorf("column name and dataType are required")
		}
		colDefs = append(colDefs, fmt.Sprintf("%q %s", c.Name, c.DataType))
		if c.PrimaryKey {
			pkCols = append(pkCols, fmt.Sprintf("%q", c.Name))
		}
	}
	if len(pkCols) == 0 {
		return fmt.Errorf("at least one column must be a primary key")
	}
	colDefs = append(colDefs, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pkCols, ", ")))

	query := fmt.Sprintf("CREATE TABLE %q.%q (%s)",
		a.keyspace, spec.Name, strings.Join(colDefs, ", "))
	return a.session.Query(query).WithContext(ctx).Exec()
}

func (a *Adapter) DropTable(ctx context.Context, table string) error {
	return a.session.Query(fmt.Sprintf("DROP TABLE %q.%q", a.keyspace, table)).WithContext(ctx).Exec()
}

func (a *Adapter) ListTables(ctx context.Context) ([]models.TableInfo, error) {
	query := `SELECT table_name FROM system_schema.tables WHERE keyspace_name = ?`
	iter := a.session.Query(query, a.keyspace).WithContext(ctx).Iter()

	tables := make([]models.TableInfo, 0)
	var name string
	for iter.Scan(&name) {
		tables = append(tables, models.TableInfo{
			Name:      name,
			Keyspace:  a.keyspace,
			TableType: "table",
		})
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return tables, nil
}

func (a *Adapter) DescribeTable(ctx context.Context, table string) ([]models.Column, error) {
	query := `SELECT column_name, type, kind FROM system_schema.columns WHERE keyspace_name = ? AND table_name = ?`
	iter := a.session.Query(query, a.keyspace, table).WithContext(ctx).Iter()

	columns := make([]models.Column, 0)
	var name, dataType, kind string
	for iter.Scan(&name, &dataType, &kind) {
		columns = append(columns, models.Column{
			Name:       name,
			DataType:   dataType,
			PrimaryKey: kind == "partition_key" || kind == "clustering",
		})
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return columns, nil
}

func (a *Adapter) ListIndexes(ctx context.Context, table string) ([]models.IndexInfo, error) {
	query := `SELECT index_name, kind, options FROM system_schema.indexes WHERE keyspace_name = ? AND table_name = ?`
	iter := a.session.Query(query, a.keyspace, table).WithContext(ctx).Iter()

	indexes := make([]models.IndexInfo, 0)
	var name, kind string
	var options map[string]string
	for iter.Scan(&name, &kind, &options) {
		ix := models.IndexInfo{Name: name}
		if target, ok := options["target"]; ok && target != "" {
			ix.Columns = []string{target}
		}
		ix.Definition = kind
		indexes = append(indexes, ix)
		options = nil
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return indexes, nil
}

func (a *Adapter) Read(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	query := fmt.Sprintf("SELECT * FROM %q.%q", a.keyspace, req.Table)

	var args []interface{}
	if len(req.Where) > 0 {
		clauses, whereArgs := buildCQLWhere(req.Where)
		query += " WHERE " + strings.Join(clauses, " AND ")
		args = append(args, whereArgs...)
	}

	if req.OrderBy != "" {
		dir := "ASC"
		if strings.ToLower(req.OrderDir) == "desc" {
			dir = "DESC"
		}
		query += fmt.Sprintf(" ORDER BY %q %s", req.OrderBy, dir)
	}
	if req.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", req.Limit)
	}

	return a.execQuery(ctx, query, args...)
}

func (a *Adapter) Create(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	columns := make([]string, 0, len(req.Data))
	placeholders := make([]string, 0, len(req.Data))
	args := make([]interface{}, 0, len(req.Data))

	for col, val := range req.Data {
		columns = append(columns, fmt.Sprintf("%q", col))
		placeholders = append(placeholders, "?")
		args = append(args, val)
	}

	query := fmt.Sprintf("INSERT INTO %q.%q (%s) VALUES (%s)",
		a.keyspace, req.Table, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	if err := a.session.Query(query, args...).WithContext(ctx).Exec(); err != nil {
		return &models.QueryResult{Error: err.Error()}, nil
	}

	return &models.QueryResult{
		Rows:         []map[string]interface{}{req.Data},
		RowsAffected: 1,
	}, nil
}

func (a *Adapter) Update(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	setClauses := make([]string, 0, len(req.Data))
	args := make([]interface{}, 0, len(req.Data)+len(req.Where))

	for col, val := range req.Data {
		setClauses = append(setClauses, fmt.Sprintf("%q = ?", col))
		args = append(args, val)
	}

	query := fmt.Sprintf("UPDATE %q.%q SET %s",
		a.keyspace, req.Table, strings.Join(setClauses, ", "))

	if len(req.Where) > 0 {
		clauses, whereArgs := buildCQLWhere(req.Where)
		query += " WHERE " + strings.Join(clauses, " AND ")
		args = append(args, whereArgs...)
	}

	if err := a.session.Query(query, args...).WithContext(ctx).Exec(); err != nil {
		return &models.QueryResult{Error: err.Error()}, nil
	}

	return &models.QueryResult{RowsAffected: 1}, nil
}

func (a *Adapter) Delete(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	query := fmt.Sprintf("DELETE FROM %q.%q", a.keyspace, req.Table)
	var args []interface{}

	if len(req.Where) > 0 {
		clauses, whereArgs := buildCQLWhere(req.Where)
		query += " WHERE " + strings.Join(clauses, " AND ")
		args = append(args, whereArgs...)
	}

	if err := a.session.Query(query, args...).WithContext(ctx).Exec(); err != nil {
		return &models.QueryResult{Error: err.Error()}, nil
	}

	return &models.QueryResult{RowsAffected: 1}, nil
}

func (a *Adapter) RawQuery(ctx context.Context, query string) (*models.QueryResult, error) {
	return a.execQuery(ctx, query)
}

func (a *Adapter) execQuery(ctx context.Context, query string, args ...interface{}) (*models.QueryResult, error) {
	iter := a.session.Query(query, args...).WithContext(ctx).Iter()

	columns := make([]string, 0)
	for _, col := range iter.Columns() {
		columns = append(columns, col.Name)
	}

	result := &models.QueryResult{
		Columns: columns,
		Rows:    make([]map[string]interface{}, 0),
	}

	for {
		row := make(map[string]interface{})
		if !iter.MapScan(row) {
			break
		}
		result.Rows = append(result.Rows, row)
		result.RowsAffected++
	}

	if err := iter.Close(); err != nil {
		return &models.QueryResult{Error: err.Error()}, nil
	}

	return result, nil
}

func buildCQLWhere(where map[string]interface{}) ([]string, []interface{}) {
	clauses := make([]string, 0, len(where))
	args := make([]interface{}, 0, len(where))
	for col, val := range where {
		clauses = append(clauses, fmt.Sprintf("%q = ?", col))
		args = append(args, val)
	}
	return clauses, args
}
