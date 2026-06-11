import { useState, useEffect } from "react";
import type { Connection, TableInfo } from "@/types";
import { api } from "@/api/client";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { CreateTableDialog } from "@/components/CreateTableDialog";

interface SidebarProps {
  refreshKey?: number;
  activeSelection: { connectionId: string; table: string } | null;
  onSelectTable: (conn: Connection, table: string) => void;
  onNewConnection: () => void;
  onEditConnection: (conn: Connection) => void;
  onOpenQuery: (conn: Connection) => void;
}

const DB_ICONS: Record<string, string> = {
  postgres: "PG",
  mongodb: "MO",
  cassandra: "CA",
  scylladb: "SC",
};

type ConnState = {
  connected: boolean;
  expanded: boolean;
  databases: string[] | null;
  currentDb: string | null;
  tables: TableInfo[] | null;
  loading: boolean;
  error: string | null;
};

const initialState: ConnState = {
  connected: false,
  expanded: false,
  databases: null,
  currentDb: null,
  tables: null,
  loading: false,
  error: null,
};

export function Sidebar({
  refreshKey,
  activeSelection,
  onSelectTable,
  onNewConnection,
  onEditConnection,
  onOpenQuery,
}: SidebarProps) {
  const [connections, setConnections] = useState<Connection[]>([]);
  const [states, setStates] = useState<Record<string, ConnState>>({});
  const [error, setError] = useState<string | null>(null);

  const [confirmDeleteConn, setConfirmDeleteConn] = useState<Connection | null>(null);
  const [confirmDropTable, setConfirmDropTable] = useState<{
    conn: Connection;
    table: string;
  } | null>(null);
  const [createTableFor, setCreateTableFor] = useState<Connection | null>(null);
  const [confirmCreate, setConfirmCreate] = useState<{
    conn: Connection;
    spec: import("@/types").TableSpec;
  } | null>(null);

  const getState = (id: string): ConnState => states[id] ?? initialState;
  const setState = (id: string, patch: Partial<ConnState>) =>
    setStates((prev) => ({ ...prev, [id]: { ...(prev[id] ?? initialState), ...patch } }));

  const loadConnections = async () => {
    try {
      const conns = await api.connections.list();
      setConnections(conns);
      setError(null);
    } catch {
      setError("Failed to load connections");
    }
  };

  useEffect(() => {
    loadConnections();
  }, [refreshKey]);

  const toggleConnection = async (conn: Connection) => {
    const s = getState(conn.id);
    if (s.expanded) {
      setState(conn.id, { expanded: false });
      return;
    }
    setState(conn.id, { expanded: true, loading: true, error: null });
    try {
      if (!s.connected) {
        await api.connections.connect(conn.id);
      }
      const dbs = await api.db.databases(conn.id);
      setState(conn.id, {
        connected: true,
        databases: dbs ?? [],
        loading: false,
      });
    } catch (e) {
      setState(conn.id, {
        loading: false,
        error: e instanceof Error ? e.message : "Failed to load databases",
      });
    }
  };

  const toggleDatabase = async (conn: Connection, dbName: string) => {
    const s = getState(conn.id);
    if (s.currentDb === dbName && s.tables) {
      setState(conn.id, { currentDb: null, tables: null });
      return;
    }
    setState(conn.id, { loading: true, currentDb: dbName, error: null });
    try {
      await api.db.switchDatabase(conn.id, dbName);
      const tables = await api.db.tables(conn.id);
      setState(conn.id, { tables: tables ?? [], loading: false });
    } catch (e) {
      setState(conn.id, {
        loading: false,
        error: e instanceof Error ? e.message : "Failed to load tables",
      });
    }
  };

  const handleDisconnect = async (conn: Connection) => {
    try {
      await api.connections.disconnect(conn.id);
      setStates((prev) => ({ ...prev, [conn.id]: initialState }));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Disconnect failed");
    }
  };

  const handleDeleteConnection = async () => {
    if (!confirmDeleteConn) return;
    try {
      await api.connections.delete(confirmDeleteConn.id);
      setStates((prev) => {
        const next = { ...prev };
        delete next[confirmDeleteConn.id];
        return next;
      });
      setConfirmDeleteConn(null);
      loadConnections();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Delete failed");
      setConfirmDeleteConn(null);
    }
  };

  const handleDropTable = async () => {
    if (!confirmDropTable) return;
    const { conn, table } = confirmDropTable;
    try {
      await api.db.dropTable(conn.id, table);
      const tables = await api.db.tables(conn.id);
      setState(conn.id, { tables });
      setConfirmDropTable(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Drop table failed");
      setConfirmDropTable(null);
    }
  };

  const handleCreateTable = async () => {
    if (!confirmCreate) return;
    const { conn, spec } = confirmCreate;
    try {
      await api.db.createTable(conn.id, spec);
      const tables = await api.db.tables(conn.id);
      setState(conn.id, { tables });
      setConfirmCreate(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Create table failed");
      setConfirmCreate(null);
    }
  };

  return (
    <aside className="w-72 bg-gray-900 border-r border-gray-800 flex flex-col">
      <div className="p-4 border-b border-gray-800">
        <h1 className="text-lg font-bold text-white">dbUI</h1>
      </div>

      <div className="p-3">
        <button
          onClick={onNewConnection}
          className="w-full px-3 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded text-sm font-medium transition-colors"
        >
          + New Connection
        </button>
      </div>

      {error && (
        <div className="mx-3 mb-2 px-3 py-2 bg-red-900/50 border border-red-800 rounded text-red-300 text-xs">
          {error}
        </div>
      )}

      <div className="flex-1 overflow-y-auto pb-4">
        {connections.map((conn) => {
          const s = getState(conn.id);
          return (
            <div key={conn.id} className="mb-1">
              <div
                className={`group flex items-center px-2 py-1.5 mx-1 rounded cursor-pointer ${
                  activeSelection?.connectionId === conn.id
                    ? "bg-gray-800"
                    : "hover:bg-gray-800/60"
                }`}
                onClick={() => toggleConnection(conn)}
              >
                <span className="w-4 text-gray-500 text-xs">
                  {s.expanded ? "▾" : "▸"}
                </span>
                <span
                  className={`w-6 h-6 rounded flex items-center justify-center text-[10px] font-bold mr-2 ${
                    s.connected
                      ? "bg-green-900 text-green-300"
                      : "bg-gray-700 text-gray-400"
                  }`}
                >
                  {DB_ICONS[conn.type] || "DB"}
                </span>
                <div className="flex-1 min-w-0">
                  <p className="text-sm truncate">{conn.name}</p>
                  <p className="text-[10px] text-gray-600 truncate">
                    {conn.host}:{conn.port}
                  </p>
                </div>
                <div
                  className="hidden group-hover:flex gap-0.5"
                  onClick={(e) => e.stopPropagation()}
                >
                  {s.connected && (
                    <button
                      onClick={() => onOpenQuery(conn)}
                      className="p-1 text-gray-500 hover:text-white"
                      title="Query editor"
                    >
                      <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                      </svg>
                    </button>
                  )}
                  <button
                    onClick={() => onEditConnection(conn)}
                    className="p-1 text-gray-500 hover:text-white"
                    title="Edit"
                  >
                    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                    </svg>
                  </button>
                  {s.connected ? (
                    <button
                      onClick={() => handleDisconnect(conn)}
                      className="p-1 text-gray-500 hover:text-yellow-400"
                      title="Disconnect"
                    >
                      <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
                      </svg>
                    </button>
                  ) : (
                    <button
                      onClick={() => setConfirmDeleteConn(conn)}
                      className="p-1 text-gray-500 hover:text-red-400"
                      title="Delete"
                    >
                      <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  )}
                </div>
              </div>

              {s.expanded && (
                <div className="ml-4">
                  {s.loading && !s.databases && (
                    <div className="px-3 py-1 text-xs text-gray-600">Loading…</div>
                  )}
                  {s.error && (
                    <div className="px-3 py-1 text-xs text-red-400">{s.error}</div>
                  )}
                  {s.databases?.map((db) => {
                    const isOpen = s.currentDb === db;
                    return (
                      <div key={db}>
                        <div
                          className={`group flex items-center px-2 py-1 mx-1 rounded cursor-pointer text-sm ${
                            isOpen ? "bg-gray-800/60" : "hover:bg-gray-800/40"
                          }`}
                          onClick={() => toggleDatabase(conn, db)}
                        >
                          <span className="w-4 text-gray-600 text-xs">
                            {isOpen ? "▾" : "▸"}
                          </span>
                          <span className="text-gray-500 text-xs mr-2">DB</span>
                          <span className="flex-1 truncate text-gray-300">{db}</span>
                          {isOpen && (
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                setCreateTableFor(conn);
                              }}
                              className="hidden group-hover:block p-0.5 text-gray-500 hover:text-green-400"
                              title="Create table"
                            >
                              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                              </svg>
                            </button>
                          )}
                        </div>

                        {isOpen && (
                          <div className="ml-4">
                            {s.loading && !s.tables && (
                              <div className="px-3 py-1 text-xs text-gray-600">Loading…</div>
                            )}
                            {s.tables?.length === 0 && (
                              <div className="px-3 py-1 text-xs text-gray-600 italic">
                                No tables
                              </div>
                            )}
                            {s.tables?.map((t) => {
                              const isActive =
                                activeSelection?.connectionId === conn.id &&
                                activeSelection.table === t.name;
                              return (
                                <div
                                  key={t.name}
                                  className={`group flex items-center px-2 py-1 mx-1 rounded cursor-pointer text-xs ${
                                    isActive
                                      ? "bg-gray-700 text-white"
                                      : "text-gray-400 hover:bg-gray-800/40 hover:text-gray-200"
                                  }`}
                                  onClick={() => onSelectTable(conn, t.name)}
                                >
                                  <span className="w-4" />
                                  <span className="text-gray-600 mr-2">
                                    {t.tableType === "collection"
                                      ? "C"
                                      : t.tableType === "view"
                                      ? "V"
                                      : "T"}
                                  </span>
                                  <span className="flex-1 truncate">{t.name}</span>
                                  <button
                                    onClick={(e) => {
                                      e.stopPropagation();
                                      setConfirmDropTable({ conn, table: t.name });
                                    }}
                                    className="hidden group-hover:block p-0.5 text-gray-600 hover:text-red-400"
                                    title="Drop table"
                                  >
                                    <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                    </svg>
                                  </button>
                                </div>
                              );
                            })}
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          );
        })}
      </div>

      <ConfirmDialog
        open={!!confirmDeleteConn}
        title="Delete connection?"
        message={
          <>
            This will permanently remove the saved connection{" "}
            <strong className="text-white">{confirmDeleteConn?.name}</strong>. The
            database itself is not affected.
          </>
        }
        confirmLabel="Delete"
        destructive
        onConfirm={handleDeleteConnection}
        onCancel={() => setConfirmDeleteConn(null)}
      />

      <ConfirmDialog
        open={!!confirmDropTable}
        title="Drop table?"
        message={
          <>
            This will permanently drop{" "}
            <strong className="text-white">{confirmDropTable?.table}</strong> and all
            its data. This cannot be undone.
          </>
        }
        confirmLabel="Drop"
        destructive
        onConfirm={handleDropTable}
        onCancel={() => setConfirmDropTable(null)}
      />

      {createTableFor && (
        <CreateTableDialog
          open
          dbType={createTableFor.type}
          database={getState(createTableFor.id).currentDb ?? ""}
          onCancel={() => setCreateTableFor(null)}
          onSubmit={(spec) => {
            const conn = createTableFor;
            setCreateTableFor(null);
            setConfirmCreate({ conn, spec });
          }}
        />
      )}

      <ConfirmDialog
        open={!!confirmCreate}
        title="Create table?"
        message={
          <>
            Create{" "}
            <strong className="text-white">
              {confirmCreate?.spec.schema
                ? `${confirmCreate.spec.schema}.${confirmCreate.spec.name}`
                : confirmCreate?.spec.name}
            </strong>
            {confirmCreate?.spec.columns?.length
              ? ` with ${confirmCreate.spec.columns.length} column${
                  confirmCreate.spec.columns.length === 1 ? "" : "s"
                }?`
              : "?"}
          </>
        }
        confirmLabel="Create"
        onConfirm={handleCreateTable}
        onCancel={() => setConfirmCreate(null)}
      />
    </aside>
  );
}
