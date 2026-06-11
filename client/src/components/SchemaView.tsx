import { useEffect, useRef, useState } from "react";
import type { Column } from "@/types";
import { api } from "@/api/client";

interface SchemaViewProps {
  connectionId: string;
  table: string;
  dbType?: string;
}

export function SchemaView({ connectionId, table, dbType }: SchemaViewProps) {
  const [columns, setColumns] = useState<Column[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editingCol, setEditingCol] = useState<string | null>(null);
  const [editValue, setEditValue] = useState("");
  const [renameError, setRenameError] = useState<string | null>(null);
  const [renaming, setRenaming] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const committingRef = useRef(false);

  const canRename = dbType === "postgres" || dbType === undefined;

  useEffect(() => {
    setLoading(true);
    setError(null);
    api.db
      .describe(connectionId, table)
      .then((c) => setColumns(c ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : "Failed to load schema"))
      .finally(() => setLoading(false));
  }, [connectionId, table]);

  useEffect(() => {
    if (editingCol && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [editingCol]);

  function startEdit(name: string) {
    setEditingCol(name);
    setEditValue(name);
    setRenameError(null);
    committingRef.current = false;
  }

  function cancelEdit() {
    committingRef.current = false;
    setEditingCol(null);
    setEditValue("");
    setRenameError(null);
  }

  async function commitRename(col: string, newName: string) {
    if (committingRef.current) return;
    committingRef.current = true;
    const trimmed = newName.trim();
    if (!trimmed || trimmed === col) {
      cancelEdit();
      return;
    }
    setRenaming(true);
    setRenameError(null);
    try {
      await api.db.renameColumn(connectionId, table, col, trimmed);
      setColumns((cols) =>
        cols.map((c) => (c.name === col ? { ...c, name: trimmed } : c))
      );
      setEditingCol(null);
    } catch (e) {
      committingRef.current = false;
      setRenameError(e instanceof Error ? e.message : "Rename failed");
    } finally {
      setRenaming(false);
    }
  }

  function handleKeyDown(e: React.KeyboardEvent, col: string) {
    if (e.key === "Enter") { e.preventDefault(); commitRename(col, editValue); }
    if (e.key === "Escape") cancelEdit();
  }

  if (loading) {
    return <div className="flex-1 flex items-center justify-center text-gray-500">Loading schema…</div>;
  }
  if (error) {
    return <div className="flex-1 flex items-center justify-center text-red-400">{error}</div>;
  }
  if (columns.length === 0) {
    return <div className="flex-1 flex items-center justify-center text-gray-500">No columns.</div>;
  }

  return (
    <div className="flex-1 overflow-auto p-4">
      {renameError && (
        <div className="mb-3 px-3 py-2 text-xs text-red-400 bg-red-900/20 rounded border border-red-800/40">
          {renameError}
        </div>
      )}
      <table className="w-full text-sm">
        <thead className="bg-gray-900 sticky top-0">
          <tr>
            <th className="px-3 py-2 text-left text-xs text-gray-500 font-medium">#</th>
            <th className="px-3 py-2 text-left text-xs text-gray-500 font-medium">Name</th>
            <th className="px-3 py-2 text-left text-xs text-gray-500 font-medium">Type</th>
            <th className="px-3 py-2 text-left text-xs text-gray-500 font-medium">Nullable</th>
            <th className="px-3 py-2 text-left text-xs text-gray-500 font-medium">Primary key</th>
          </tr>
        </thead>
        <tbody>
          {columns.map((c, i) => (
            <tr key={c.name} className="border-t border-gray-800/50 hover:bg-gray-800/30 group">
              <td className="px-3 py-1.5 text-xs text-gray-600">{i + 1}</td>
              <td className="px-3 py-1.5 text-gray-200">
                {editingCol === c.name ? (
                  <input
                    ref={inputRef}
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    onKeyDown={(e) => handleKeyDown(e, c.name)}
                    onBlur={() => commitRename(c.name, editValue)}
                    disabled={renaming}
                    className="bg-gray-800 border border-blue-500 rounded px-1.5 py-0.5 text-sm text-gray-100 outline-none w-40"
                  />
                ) : (
                  <span className="flex items-center gap-1.5">
                    {c.name}
                    {canRename && (
                      <button
                        onClick={() => startEdit(c.name)}
                        className="opacity-0 group-hover:opacity-100 text-gray-500 hover:text-gray-300 transition-opacity"
                        title="Rename column"
                      >
                        <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                          <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                          <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                        </svg>
                      </button>
                    )}
                  </span>
                )}
              </td>
              <td className="px-3 py-1.5 text-gray-400 font-mono text-xs">{c.dataType}</td>
              <td className="px-3 py-1.5 text-xs">
                {c.nullable ? (
                  <span className="text-gray-500">YES</span>
                ) : (
                  <span className="text-yellow-500">NO</span>
                )}
              </td>
              <td className="px-3 py-1.5 text-xs">
                {c.primaryKey && <span className="text-yellow-500">PK</span>}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
