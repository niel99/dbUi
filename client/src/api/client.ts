import type { Connection, TableInfo, Column, QueryResult, CRUDRequest, TableSpec, IndexInfo } from "@/types";

const BASE = "/api";

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });

  if (res.status === 204) return undefined as T;

  const text = await res.text();
  let data: unknown = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    if (!res.ok) {
      throw new Error(
        `HTTP ${res.status}: ${text.slice(0, 200) || res.statusText}`
      );
    }
    throw new Error(`Invalid JSON response from ${path}: ${text.slice(0, 200)}`);
  }

  if (!res.ok) {
    const msg = (data as { error?: string })?.error ?? `HTTP ${res.status}`;
    throw new Error(msg);
  }
  return data as T;
}

// Connection endpoints
export const api = {
  connections: {
    list: () => request<Connection[]>("/connections"),
    get: (id: string) => request<Connection>(`/connections/${id}`),
    create: (conn: Partial<Connection>) =>
      request<Connection>("/connections", {
        method: "POST",
        body: JSON.stringify(conn),
      }),
    update: (id: string, conn: Partial<Connection>) =>
      request<Connection>(`/connections/${id}`, {
        method: "PUT",
        body: JSON.stringify(conn),
      }),
    delete: (id: string) =>
      request<void>(`/connections/${id}`, { method: "DELETE" }),
    connect: (id: string) =>
      request<{ status: string }>(`/connections/${id}/connect`, {
        method: "POST",
      }),
    disconnect: (id: string) =>
      request<{ status: string }>(`/connections/${id}/disconnect`, {
        method: "POST",
      }),
    test: (id: string) =>
      request<{ success: boolean; error?: string }>(
        `/connections/${id}/test`,
        { method: "POST" }
      ),
  },

  db: {
    databases: (connId: string) =>
      request<string[]>(`/db/${connId}/databases`),
    switchDatabase: (connId: string, database: string) =>
      request<{ status: string; database: string }>(`/db/${connId}/switch`, {
        method: "POST",
        body: JSON.stringify({ database }),
      }),
    createTable: (connId: string, spec: TableSpec) =>
      request<{ status: string; name: string }>(`/db/${connId}/tables`, {
        method: "POST",
        body: JSON.stringify(spec),
      }),
    dropTable: (connId: string, table: string) =>
      request<{ status: string; name: string }>(
        `/db/${connId}/tables/${encodeURIComponent(table)}`,
        { method: "DELETE" }
      ),
    tables: (connId: string) =>
      request<TableInfo[]>(`/db/${connId}/tables`),
    describe: (connId: string, table: string) =>
      request<Column[]>(`/db/${connId}/tables/${table}/describe`),
    renameColumn: (connId: string, table: string, oldName: string, newName: string) =>
      request<{ status: string; table: string; oldName: string; newName: string }>(
        `/db/${connId}/tables/${encodeURIComponent(table)}/rename-column`,
        { method: "POST", body: JSON.stringify({ oldName, newName }) }
      ),
    indexes: (connId: string, table: string) =>
      request<IndexInfo[]>(`/db/${connId}/tables/${table}/indexes`),
    read: (connId: string, table: string, req: Partial<CRUDRequest>) =>
      request<QueryResult>(`/db/${connId}/tables/${table}/read`, {
        method: "POST",
        body: JSON.stringify(req),
      }),
    create: (connId: string, table: string, data: Record<string, unknown>) =>
      request<QueryResult>(`/db/${connId}/tables/${table}/create`, {
        method: "POST",
        body: JSON.stringify({ data }),
      }),
    update: (
      connId: string,
      table: string,
      data: Record<string, unknown>,
      where: Record<string, unknown>
    ) =>
      request<QueryResult>(`/db/${connId}/tables/${table}/update`, {
        method: "POST",
        body: JSON.stringify({ data, where }),
      }),
    delete: (
      connId: string,
      table: string,
      where: Record<string, unknown>
    ) =>
      request<QueryResult>(`/db/${connId}/tables/${table}/delete`, {
        method: "POST",
        body: JSON.stringify({ where }),
      }),
    rawQuery: (connId: string, query: string) =>
      request<QueryResult>(`/db/${connId}/query`, {
        method: "POST",
        body: JSON.stringify({ query }),
      }),
  },
};
