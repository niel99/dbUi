import { useState } from "react";
import type { Connection, DBType } from "@/types";
import { api } from "@/api/client";

interface ConnectionFormProps {
  connection: Connection | null;
  onSaved: (conn: Connection) => void;
  onCancel: () => void;
}

const DB_TYPES: { value: DBType; label: string; defaultPort: number }[] = [
  { value: "postgres", label: "PostgreSQL", defaultPort: 5432 },
  { value: "mongodb", label: "MongoDB", defaultPort: 27017 },
  { value: "cassandra", label: "Cassandra", defaultPort: 9042 },
  { value: "scylladb", label: "ScyllaDB", defaultPort: 9042 },
];

export function ConnectionForm({
  connection,
  onSaved,
  onCancel,
}: ConnectionFormProps) {
  const [savedId, setSavedId] = useState<string | null>(connection?.id ?? null);
  const [form, setForm] = useState({
    name: connection?.name || "",
    type: (connection?.type || "postgres") as DBType,
    host: connection?.host || "localhost",
    port: connection?.port || 5432,
    username: connection?.username || "",
    password: connection?.password || "",
    database: connection?.database || "",
  });
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<{
    success: boolean;
    error?: string;
  } | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleTypeChange = (type: DBType) => {
    const dbType = DB_TYPES.find((t) => t.value === type);
    setForm((prev) => ({
      ...prev,
      type,
      port: dbType?.defaultPort || prev.port,
    }));
  };

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    try {
      let saved: Connection;
      if (savedId) {
        saved = await api.connections.update(savedId, form);
      } else {
        saved = await api.connections.create(form);
      }
      setSavedId(saved.id);
      onSaved(saved);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
    }
  };

  const handleTest = async () => {
    setTesting(true);
    setTestResult(null);
    try {
      // Save first if new, then test
      let connId = savedId;
      if (!connId) {
        const saved = await api.connections.create(form);
        connId = saved.id;
        setSavedId(connId);
      } else {
        await api.connections.update(connId, form);
      }
      const result = await api.connections.test(connId);
      setTestResult(result);
    } catch (e) {
      setTestResult({
        success: false,
        error: e instanceof Error ? e.message : "Test failed",
      });
    } finally {
      setTesting(false);
    }
  };

  return (
    <div className="flex-1 flex items-center justify-center p-8">
      <div className="w-full max-w-lg bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-800 p-6">
        <h2 className="text-xl font-semibold mb-6">
          {connection ? "Edit Connection" : "New Connection"}
        </h2>

        <div className="space-y-4">
          <div>
            <label className="block text-sm text-gray-500 dark:text-gray-400 mb-1">
              Connection Name
            </label>
            <input
              type="text"
              value={form.name}
              onChange={(e) =>
                setForm((prev) => ({ ...prev, name: e.target.value }))
              }
              className="w-full px-3 py-2 bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded text-sm focus:outline-none focus:border-blue-500"
              placeholder="My Database"
            />
          </div>

          <div>
            <label className="block text-sm text-gray-500 dark:text-gray-400 mb-1">
              Database Type
            </label>
            <div className="grid grid-cols-4 gap-2">
              {DB_TYPES.map((dbType) => (
                <button
                  key={dbType.value}
                  onClick={() => handleTypeChange(dbType.value)}
                  className={`px-3 py-2 rounded text-sm font-medium transition-colors ${
                    form.type === dbType.value
                      ? "bg-blue-600 text-white"
                      : "bg-gray-100 text-gray-500 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700"
                  }`}
                >
                  {dbType.label}
                </button>
              ))}
            </div>
          </div>

          <div className="grid grid-cols-3 gap-3">
            <div className="col-span-2">
              <label className="block text-sm text-gray-500 dark:text-gray-400 mb-1">Host</label>
              <input
                type="text"
                value={form.host}
                onChange={(e) =>
                  setForm((prev) => ({ ...prev, host: e.target.value }))
                }
                className="w-full px-3 py-2 bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded text-sm focus:outline-none focus:border-blue-500"
              />
            </div>
            <div>
              <label className="block text-sm text-gray-500 dark:text-gray-400 mb-1">Port</label>
              <input
                type="number"
                value={form.port}
                onChange={(e) =>
                  setForm((prev) => ({
                    ...prev,
                    port: parseInt(e.target.value) || 0,
                  }))
                }
                className="w-full px-3 py-2 bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded text-sm focus:outline-none focus:border-blue-500"
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm text-gray-500 dark:text-gray-400 mb-1">
                Username
              </label>
              <input
                type="text"
                value={form.username}
                onChange={(e) =>
                  setForm((prev) => ({ ...prev, username: e.target.value }))
                }
                className="w-full px-3 py-2 bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded text-sm focus:outline-none focus:border-blue-500"
              />
            </div>
            <div>
              <label className="block text-sm text-gray-500 dark:text-gray-400 mb-1">
                Password
              </label>
              <input
                type="password"
                value={form.password}
                onChange={(e) =>
                  setForm((prev) => ({ ...prev, password: e.target.value }))
                }
                className="w-full px-3 py-2 bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded text-sm focus:outline-none focus:border-blue-500"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm text-gray-500 dark:text-gray-400 mb-1">
              Database{form.type === "cassandra" || form.type === "scylladb"
                ? " / Keyspace"
                : ""}
            </label>
            <input
              type="text"
              value={form.database}
              onChange={(e) =>
                setForm((prev) => ({ ...prev, database: e.target.value }))
              }
              className="w-full px-3 py-2 bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded text-sm focus:outline-none focus:border-blue-500"
            />
          </div>

          {testResult && (
            <div
              className={`px-3 py-2 rounded text-sm ${
                testResult.success
                  ? "bg-green-100 dark:bg-green-900/50 border border-green-300 dark:border-green-800 text-green-700 dark:text-green-300"
                  : "bg-red-100 dark:bg-red-900/50 border border-red-300 dark:border-red-800 text-red-700 dark:text-red-300"
              }`}
            >
              {testResult.success
                ? "Connection successful!"
                : `Connection failed: ${testResult.error}`}
            </div>
          )}

          {error && (
            <div className="px-3 py-2 bg-red-100 dark:bg-red-900/50 border border-red-300 dark:border-red-800 rounded text-red-700 dark:text-red-300 text-sm">
              {error}
            </div>
          )}

          <div className="flex gap-3 pt-2">
            <button
              onClick={handleTest}
              disabled={testing}
              className="px-4 py-2 bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 rounded text-sm font-medium transition-colors disabled:opacity-50"
            >
              {testing ? "Testing..." : "Test Connection"}
            </button>
            <div className="flex-1" />
            <button
              onClick={onCancel}
              className="px-4 py-2 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 rounded text-sm font-medium transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleSave}
              disabled={saving || !form.name || !form.host}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded text-sm font-medium transition-colors disabled:opacity-50"
            >
              {saving ? "Saving..." : "Save & Connect"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
