package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
	"github.com/niel/dbui/server/internal/models"
)

// Adapter implements DatabaseAdapter for PostgreSQL.
type Adapter struct {
	db *sql.DB
}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Type() models.DBType {
	return models.DBTypePostgres
}

func (a *Adapter) Connect(ctx context.Context, conn models.Connection) error {
	sslmode := conn.Options["sslmode"]
	if sslmode == "" {
		sslmode = "disable"
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		conn.Host, conn.Port, conn.Username, conn.Password, conn.Database, sslmode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return err
	}

	a.db = db
	return nil
}

func (a *Adapter) Disconnect(ctx context.Context) error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

func (a *Adapter) Ping(ctx context.Context) error {
	if a.db == nil {
		return fmt.Errorf("not connected")
	}
	return a.db.PingContext(ctx)
}

func (a *Adapter) ListDatabases(ctx context.Context) ([]string, error) {
	rows, err := a.db.QueryContext(ctx,
		`SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dbs := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		dbs = append(dbs, name)
	}
	return dbs, rows.Err()
}

func (a *Adapter) CreateTable(ctx context.Context, spec models.TableSpec) error {
	if spec.Name == "" || len(spec.Columns) == 0 {
		return fmt.Errorf("table name and at least one column are required")
	}
	schema := spec.Schema
	if schema == "" {
		schema = "public"
	}

	colDefs := make([]string, 0, len(spec.Columns))
	pkCols := make([]string, 0)
	checkConstraints := make([]string, 0)
	for _, c := range spec.Columns {
		if c.Name == "" || c.DataType == "" {
			return fmt.Errorf("column name and dataType are required")
		}
		def := fmt.Sprintf("%q %s", c.Name, c.DataType)
		if !c.Nullable {
			def += " NOT NULL"
		}
		if c.Unique {
			def += " UNIQUE"
		}
		if c.Default != "" {
			def += " DEFAULT " + c.Default
		}
		colDefs = append(colDefs, def)
		if c.PrimaryKey {
			pkCols = append(pkCols, fmt.Sprintf("%q", c.Name))
		}
		if c.MinLength != nil {
			checkConstraints = append(checkConstraints,
				fmt.Sprintf("length(%q) >= %d", c.Name, *c.MinLength))
		}
		if c.MaxLength != nil {
			checkConstraints = append(checkConstraints,
				fmt.Sprintf("length(%q) <= %d", c.Name, *c.MaxLength))
		}
		if c.Pattern != "" {
			checkConstraints = append(checkConstraints,
				fmt.Sprintf("%q ~ '%s'", c.Name, strings.ReplaceAll(c.Pattern, "'", "''")))
		}
	}
	if len(pkCols) > 0 {
		colDefs = append(colDefs, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pkCols, ", ")))
	}
	for _, chk := range checkConstraints {
		colDefs = append(colDefs, fmt.Sprintf("CHECK (%s)", chk))
	}

	query := fmt.Sprintf("CREATE TABLE %q.%q (%s)",
		schema, spec.Name, strings.Join(colDefs, ", "))
	_, err := a.db.ExecContext(ctx, query)
	return err
}

func (a *Adapter) DropTable(ctx context.Context, table string) error {
	schema, tableName := "public", table
	if parts := strings.SplitN(table, ".", 2); len(parts) == 2 {
		schema, tableName = parts[0], parts[1]
	}
	_, err := a.db.ExecContext(ctx, fmt.Sprintf("DROP TABLE %q.%q", schema, tableName))
	return err
}

func (a *Adapter) ListTables(ctx context.Context) ([]models.TableInfo, error) {
	query := `
		SELECT table_schema, table_name, table_type
		FROM information_schema.tables
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY table_schema, table_name`

	rows, err := a.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []models.TableInfo
	for rows.Next() {
		var t models.TableInfo
		var tableType string
		if err := rows.Scan(&t.Schema, &t.Name, &tableType); err != nil {
			return nil, err
		}
		if tableType == "VIEW" {
			t.TableType = "view"
		} else {
			t.TableType = "table"
		}
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

func (a *Adapter) DescribeTable(ctx context.Context, table string) ([]models.Column, error) {
	parts := strings.SplitN(table, ".", 2)
	schema, tableName := "public", table
	if len(parts) == 2 {
		schema, tableName = parts[0], parts[1]
	}

	query := `
		SELECT c.column_name, c.data_type, c.is_nullable,
			COALESCE(
				(SELECT true FROM information_schema.key_column_usage kcu
				 JOIN information_schema.table_constraints tc
				   ON tc.constraint_name = kcu.constraint_name
				   AND tc.table_schema = kcu.table_schema
				 WHERE tc.constraint_type = 'PRIMARY KEY'
				   AND kcu.table_schema = c.table_schema
				   AND kcu.table_name = c.table_name
				   AND kcu.column_name = c.column_name
				 LIMIT 1), false
			) as is_pk
		FROM information_schema.columns c
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position`

	rows, err := a.db.QueryContext(ctx, query, schema, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []models.Column
	for rows.Next() {
		var col models.Column
		var nullable string
		if err := rows.Scan(&col.Name, &col.DataType, &nullable, &col.PrimaryKey); err != nil {
			return nil, err
		}
		col.Nullable = nullable == "YES"
		columns = append(columns, col)
	}
	return columns, rows.Err()
}

func (a *Adapter) ListIndexes(ctx context.Context, table string) ([]models.IndexInfo, error) {
	schema, tableName := "public", table
	if parts := strings.SplitN(table, ".", 2); len(parts) == 2 {
		schema, tableName = parts[0], parts[1]
	}

	query := `
		SELECT i.relname, ix.indisunique, ix.indisprimary, pg_get_indexdef(ix.indexrelid)
		FROM pg_index ix
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_class t ON t.oid = ix.indrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = $1 AND t.relname = $2
		ORDER BY i.relname`

	rows, err := a.db.QueryContext(ctx, query, schema, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexes := make([]models.IndexInfo, 0)
	for rows.Next() {
		var ix models.IndexInfo
		if err := rows.Scan(&ix.Name, &ix.Unique, &ix.Primary, &ix.Definition); err != nil {
			return nil, err
		}
		indexes = append(indexes, ix)
	}
	return indexes, rows.Err()
}

func (a *Adapter) Read(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	schema := req.Schema
	if schema == "" {
		schema = "public"
	}
	table := fmt.Sprintf("%q.%q", schema, req.Table)

	query := fmt.Sprintf("SELECT * FROM %s", table)

	var args []interface{}
	if len(req.Where) > 0 {
		whereClauses, whereArgs := buildWhere(req.Where, 1)
		query += " WHERE " + strings.Join(whereClauses, " AND ")
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
	if req.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", req.Offset)
	}

	return a.execQuery(ctx, query, args...)
}

func (a *Adapter) Create(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	schema := req.Schema
	if schema == "" {
		schema = "public"
	}
	table := fmt.Sprintf("%q.%q", schema, req.Table)

	columns := make([]string, 0, len(req.Data))
	placeholders := make([]string, 0, len(req.Data))
	args := make([]interface{}, 0, len(req.Data))
	i := 1
	for col, val := range req.Data {
		columns = append(columns, fmt.Sprintf("%q", col))
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		args = append(args, val)
		i++
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING *",
		table, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	return a.execQuery(ctx, query, args...)
}

func (a *Adapter) Update(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	schema := req.Schema
	if schema == "" {
		schema = "public"
	}
	table := fmt.Sprintf("%q.%q", schema, req.Table)

	setClauses := make([]string, 0, len(req.Data))
	args := make([]interface{}, 0, len(req.Data)+len(req.Where))
	i := 1
	for col, val := range req.Data {
		setClauses = append(setClauses, fmt.Sprintf("%q = $%d", col, i))
		args = append(args, val)
		i++
	}

	query := fmt.Sprintf("UPDATE %s SET %s", table, strings.Join(setClauses, ", "))

	if len(req.Where) > 0 {
		whereClauses, whereArgs := buildWhere(req.Where, i)
		query += " WHERE " + strings.Join(whereClauses, " AND ")
		args = append(args, whereArgs...)
	}

	query += " RETURNING *"
	return a.execQuery(ctx, query, args...)
}

func (a *Adapter) Delete(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	schema := req.Schema
	if schema == "" {
		schema = "public"
	}
	table := fmt.Sprintf("%q.%q", schema, req.Table)

	query := fmt.Sprintf("DELETE FROM %s", table)
	var args []interface{}

	if len(req.Where) > 0 {
		whereClauses, whereArgs := buildWhere(req.Where, 1)
		query += " WHERE " + strings.Join(whereClauses, " AND ")
		args = append(args, whereArgs...)
	}

	query += " RETURNING *"
	return a.execQuery(ctx, query, args...)
}

func (a *Adapter) RenameColumn(ctx context.Context, table, oldName, newName string) error {
	schema, tableName := "public", table
	if parts := strings.SplitN(table, ".", 2); len(parts) == 2 {
		schema, tableName = parts[0], parts[1]
	}
	if oldName == "" || newName == "" {
		return fmt.Errorf("old and new column names are required")
	}
	query := fmt.Sprintf("ALTER TABLE %q.%q RENAME COLUMN %q TO %q", schema, tableName, oldName, newName)
	_, err := a.db.ExecContext(ctx, query)
	return err
}

func (a *Adapter) RawQuery(ctx context.Context, query string) (*models.QueryResult, error) {
	return a.execQuery(ctx, query)
}

func (a *Adapter) execQuery(ctx context.Context, query string, args ...interface{}) (*models.QueryResult, error) {
	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return &models.QueryResult{Error: err.Error()}, nil
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := &models.QueryResult{
		Columns: columns,
		Rows:    make([]map[string]interface{}, 0),
	}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		result.Rows = append(result.Rows, row)
		result.RowsAffected++
	}

	return result, rows.Err()
}

func buildWhere(where map[string]interface{}, startIdx int) ([]string, []interface{}) {
	clauses := make([]string, 0, len(where))
	args := make([]interface{}, 0, len(where))
	i := startIdx
	for col, val := range where {
		clauses = append(clauses, fmt.Sprintf("%q = $%d", col, i))
		args = append(args, val)
		i++
	}
	return clauses, args
}
