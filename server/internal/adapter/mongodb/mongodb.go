package mongodb

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/niel/dbui/server/internal/models"
)

// Adapter implements DatabaseAdapter for MongoDB.
type Adapter struct {
	client *mongo.Client
	dbName string
}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Type() models.DBType {
	return models.DBTypeMongoDB
}

func (a *Adapter) Connect(ctx context.Context, conn models.Connection) error {
	if opts, ok := conn.Options["uri"]; ok {
		clientOpts := options.Client().ApplyURI(opts)
		client, err := mongo.Connect(clientOpts)
		if err != nil {
			return err
		}
		if err := client.Ping(ctx, nil); err != nil {
			client.Disconnect(ctx)
			return err
		}
		a.client = client
		a.dbName = conn.Database
		return nil
	}

	var uri string
	if conn.Username == "" {
		uri = fmt.Sprintf("mongodb://%s:%d/%s", conn.Host, conn.Port, conn.Database)
	} else {
		authSource := "admin"
		if as, ok := conn.Options["authSource"]; ok {
			authSource = as
		}
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?authSource=%s",
			conn.Username, conn.Password, conn.Host, conn.Port, conn.Database, authSource)
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}

	if err := client.Ping(ctx, nil); err != nil {
		client.Disconnect(ctx)
		return err
	}

	a.client = client
	a.dbName = conn.Database
	return nil
}

func (a *Adapter) Disconnect(ctx context.Context) error {
	if a.client != nil {
		return a.client.Disconnect(ctx)
	}
	return nil
}

func (a *Adapter) Ping(ctx context.Context) error {
	if a.client == nil {
		return fmt.Errorf("not connected")
	}
	return a.client.Ping(ctx, nil)
}

func (a *Adapter) ListDatabases(ctx context.Context) ([]string, error) {
	names, err := a.client.ListDatabaseNames(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	return names, nil
}

func (a *Adapter) CreateTable(ctx context.Context, spec models.TableSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("collection name is required")
	}
	return a.client.Database(a.dbName).CreateCollection(ctx, spec.Name)
}

func (a *Adapter) DropTable(ctx context.Context, table string) error {
	return a.client.Database(a.dbName).Collection(table).Drop(ctx)
}

func (a *Adapter) ListTables(ctx context.Context) ([]models.TableInfo, error) {
	db := a.client.Database(a.dbName)
	collections, err := db.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, err
	}

	tables := make([]models.TableInfo, len(collections))
	for i, name := range collections {
		tables[i] = models.TableInfo{
			Name:      name,
			TableType: "collection",
		}
	}
	return tables, nil
}

func (a *Adapter) DescribeTable(ctx context.Context, table string) ([]models.Column, error) {
	// MongoDB is schemaless; sample a document to infer fields
	coll := a.client.Database(a.dbName).Collection(table)

	var doc bson.M
	err := coll.FindOne(ctx, bson.D{}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return []models.Column{}, nil
		}
		return nil, err
	}

	columns := make([]models.Column, 0, len(doc))
	for key, val := range doc {
		col := models.Column{
			Name:     key,
			DataType: fmt.Sprintf("%T", val),
			Nullable: true,
		}
		if key == "_id" {
			col.PrimaryKey = true
		}
		columns = append(columns, col)
	}
	return columns, nil
}

func (a *Adapter) ListIndexes(ctx context.Context, table string) ([]models.IndexInfo, error) {
	coll := a.client.Database(a.dbName).Collection(table)
	cursor, err := coll.Indexes().List(ctx)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	indexes := make([]models.IndexInfo, 0)
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		ix := models.IndexInfo{}
		if name, ok := doc["name"].(string); ok {
			ix.Name = name
		}
		if unique, ok := doc["unique"].(bool); ok {
			ix.Unique = unique
		}
		if key, ok := doc["key"].(bson.M); ok {
			for k := range key {
				ix.Columns = append(ix.Columns, k)
			}
		}
		if ix.Name == "_id_" {
			ix.Primary = true
		}
		indexes = append(indexes, ix)
	}
	return indexes, cursor.Err()
}

func (a *Adapter) Read(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	coll := a.client.Database(a.dbName).Collection(req.Table)

	filter := bson.D{}
	if len(req.Where) > 0 {
		for k, v := range req.Where {
			filter = append(filter, bson.E{Key: k, Value: v})
		}
	}

	opts := options.Find()
	if req.Limit > 0 {
		opts.SetLimit(int64(req.Limit))
	}
	if req.Offset > 0 {
		opts.SetSkip(int64(req.Offset))
	}
	if req.OrderBy != "" {
		dir := 1
		if strings.ToLower(req.OrderDir) == "desc" {
			dir = -1
		}
		opts.SetSort(bson.D{{Key: req.OrderBy, Value: dir}})
	}

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return &models.QueryResult{Error: err.Error()}, nil
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	rows := make([]map[string]interface{}, len(results))
	for i, doc := range results {
		row := make(map[string]interface{})
		for k, v := range doc {
			row[k] = v
		}
		rows[i] = row
	}

	// Collect column names from first row
	var columns []string
	if len(rows) > 0 {
		for k := range rows[0] {
			columns = append(columns, k)
		}
	}

	return &models.QueryResult{
		Columns:      columns,
		Rows:         rows,
		RowsAffected: int64(len(rows)),
	}, nil
}

func (a *Adapter) Create(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	coll := a.client.Database(a.dbName).Collection(req.Table)

	doc := bson.D{}
	for k, v := range req.Data {
		doc = append(doc, bson.E{Key: k, Value: v})
	}

	result, err := coll.InsertOne(ctx, doc)
	if err != nil {
		return &models.QueryResult{Error: err.Error()}, nil
	}

	row := make(map[string]interface{})
	for k, v := range req.Data {
		row[k] = v
	}
	row["_id"] = result.InsertedID

	return &models.QueryResult{
		Rows:         []map[string]interface{}{row},
		RowsAffected: 1,
	}, nil
}

func (a *Adapter) Update(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	coll := a.client.Database(a.dbName).Collection(req.Table)

	filter := bson.D{}
	for k, v := range req.Where {
		filter = append(filter, bson.E{Key: k, Value: v})
	}

	update := bson.D{{Key: "$set", Value: req.Data}}

	result, err := coll.UpdateMany(ctx, filter, update)
	if err != nil {
		return &models.QueryResult{Error: err.Error()}, nil
	}

	return &models.QueryResult{
		RowsAffected: result.ModifiedCount,
	}, nil
}

func (a *Adapter) Delete(ctx context.Context, req models.CRUDRequest) (*models.QueryResult, error) {
	coll := a.client.Database(a.dbName).Collection(req.Table)

	filter := bson.D{}
	for k, v := range req.Where {
		filter = append(filter, bson.E{Key: k, Value: v})
	}

	result, err := coll.DeleteMany(ctx, filter)
	if err != nil {
		return &models.QueryResult{Error: err.Error()}, nil
	}

	return &models.QueryResult{
		RowsAffected: result.DeletedCount,
	}, nil
}

func (a *Adapter) RawQuery(ctx context.Context, query string) (*models.QueryResult, error) {
	// MongoDB doesn't have SQL queries — execute as a command
	db := a.client.Database(a.dbName)

	var cmdResult bson.M
	err := db.RunCommand(ctx, bson.D{{Key: "eval", Value: query}}).Decode(&cmdResult)
	if err != nil {
		return &models.QueryResult{Error: err.Error()}, nil
	}

	row := make(map[string]interface{})
	for k, v := range cmdResult {
		row[k] = v
	}

	return &models.QueryResult{
		Rows:         []map[string]interface{}{row},
		RowsAffected: 1,
	}, nil
}
