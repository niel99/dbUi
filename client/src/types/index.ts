export type DBType = "postgres" | "mongodb" | "cassandra" | "scylladb";

export interface Connection {
  id: string;
  name: string;
  type: DBType;
  host: string;
  port: number;
  username: string;
  password?: string;
  database: string;
  options?: Record<string, string>;
  created: string;
  updated: string;
}

export interface TableInfo {
  name: string;
  schema?: string;
  keyspace?: string;
  rowCount?: number;
  tableType?: string;
}

export interface Column {
  name: string;
  dataType: string;
  nullable: boolean;
  primaryKey: boolean;
}

export interface QueryResult {
  columns?: string[];
  rows: Record<string, unknown>[];
  rowsAffected: number;
  error?: string;
}

export interface IndexInfo {
  name: string;
  columns?: string[];
  unique?: boolean;
  primary?: boolean;
  definition?: string;
}

export interface ColumnSpec {
  name: string;
  dataType: string;
  nullable: boolean;
  primaryKey: boolean;
  unique?: boolean;
  default?: string;
  minLength?: number;
  maxLength?: number;
  pattern?: string;
}

export interface TableSpec {
  name: string;
  schema?: string;
  columns: ColumnSpec[];
}

export interface CRUDRequest {
  table: string;
  schema?: string;
  keyspace?: string;
  data?: Record<string, unknown>;
  where?: Record<string, unknown>;
  limit?: number;
  offset?: number;
  orderBy?: string;
  orderDir?: "asc" | "desc";
}
