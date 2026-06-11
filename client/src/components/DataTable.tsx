import { useState, useEffect, useCallback } from "react";
import type { Column, QueryResult } from "@/types";
import { api } from "@/api/client";
import { ConfirmDialog } from "@/components/ConfirmDialog";

interface DataTableProps {
  connectionId: string;
  table: string;
}

interface FilterRow {
  col: string;
  val: string;
}

interface SortState {
  col: string;
  dir: "asc" | "desc";
}

function nextSort(current: SortState | null, col: string): SortState | null {
  if (current?.col !== col) return { col, dir: "asc" };
  if (current.dir === "asc") return { col, dir: "desc" };
  return null;
}

export function DataTable({ connectionId, table }: DataTableProps) {
  const [columns, setColumns] = useState<Column[]>([]);
  const [data, setData] = useState<QueryResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(0);
  const [editingRow, setEditingRow] = useState<Record<string, unknown> | null>(null);
  const [editingRaw, setEditingRaw] = useState<Record<string, string>>({});
  const [newRow, setNewRow] = useState<Record<string, unknown> | null>(null);
  const [rowToDelete, setRowToDelete] = useState<Record<string, unknown> | null>(null);
  const [showFilters, setShowFilters] = useState(false);
  const [filterRows, setFilterRows] = useState<FilterRow[]>([{ col: "", val: "" }]);
  const [appliedWhere, setAppliedWhere] = useState<Record<string, unknown>>({});
  const [sort, setSort] = useState<SortState | null>(null);
  const pageSize = 50;

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [cols, result] = await Promise.all([
        api.db.describe(connectionId, table),
        api.db.read(connectionId, table, {
          limit: pageSize,
          offset: page * pageSize,
          ...(Object.keys(appliedWhere).length > 0 ? { where: appliedWhere } : {}),
          ...(sort ? { orderBy: sort.col, orderDir: sort.dir } : {}),
        }),
      ]);
      setColumns(cols);
      setData(result);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, [connectionId, table, page, appliedWhere, sort]);

  useEffect(() => {
    setPage(0);
    setAppliedWhere({});
    setFilterRows([{ col: "", val: "" }]);
    setSort(null);
  }, [table]);

  const applyFilters = () => {
    const where: Record<string, unknown> = {};
    for (const { col, val } of filterRows) {
      if (col && val !== "") where[col] = val;
    }
    setAppliedWhere(where);
    setPage(0);
  };

  const clearFilters = () => {
    setFilterRows([{ col: "", val: "" }]);
    setAppliedWhere({});
    setPage(0);
  };

  const activeFilterCount = Object.keys(appliedWhere).length;

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleCreate = async () => {
    if (!newRow) return;
    try {
      await api.db.create(connectionId, table, newRow);
      setNewRow(null);
      loadData();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Create failed");
    }
  };

  const handleUpdate = async () => {
    if (!editingRow) return;
    const pkCols = columns.filter((c) => c.primaryKey);
    const where: Record<string, unknown> = {};
    const updateData: Record<string, unknown> = {};

    for (const col of pkCols) {
      where[col.name] = editingRow[col.name];
    }
    for (const [key, val] of Object.entries(editingRow)) {
      if (pkCols.some((c) => c.name === key)) continue;
      if (key in editingRaw) {
        try {
          updateData[key] = JSON.parse(editingRaw[key]!);
        } catch {
          setError(`Invalid JSON for field "${key}"`);
          return;
        }
      } else {
        updateData[key] = val;
      }
    }

    try {
      await api.db.update(connectionId, table, updateData, where);
      setEditingRow(null);
      setEditingRaw({});
      loadData();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Update failed");
    }
  };

  const confirmDelete = async () => {
    if (!rowToDelete) return;
    const pkCols = columns.filter((c) => c.primaryKey);
    const where: Record<string, unknown> = {};
    for (const col of pkCols) {
      where[col.name] = rowToDelete[col.name];
    }

    try {
      await api.db.delete(connectionId, table, where);
      setRowToDelete(null);
      loadData();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Delete failed");
      setRowToDelete(null);
    }
  };

  if (loading && !data) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-500">
        Loading...
      </div>
    );
  }

  if (error && !data) {
    return (
      <div className="flex-1 flex items-center justify-center text-red-400">
        {error}
      </div>
    );
  }

  const displayColumns = data?.columns || columns.map((c) => c.name);

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      {/* Toolbar */}
      <div className="flex items-center gap-2 px-4 py-2 bg-gray-900 border-b border-gray-800">
        <h3 className="text-sm font-medium">{table}</h3>
        <span className="text-xs text-gray-500">
          {data?.rowsAffected || 0} rows
        </span>
        <div className="flex-1" />
        <button
          onClick={() => setShowFilters((v) => !v)}
          className={`px-3 py-1 rounded text-xs font-medium transition-colors ${
            showFilters || activeFilterCount > 0
              ? "bg-blue-700 hover:bg-blue-600 text-white"
              : "bg-gray-700 hover:bg-gray-600"
          }`}
        >
          Filter{activeFilterCount > 0 ? ` (${activeFilterCount})` : ""}
        </button>
        <button
          onClick={() =>
            setNewRow(
              Object.fromEntries(columns.map((c) => [c.name, ""]))
            )
          }
          className="px-3 py-1 bg-green-700 hover:bg-green-600 rounded text-xs font-medium transition-colors"
        >
          + Insert Row
        </button>
        <button
          onClick={loadData}
          className="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-xs font-medium transition-colors"
        >
          Refresh
        </button>
      </div>

      {/* Filter panel */}
      {showFilters && (
        <div className="px-4 py-3 bg-gray-850 border-b border-gray-800 bg-gray-900/80">
          <div className="flex flex-col gap-2">
            {filterRows.map((row, i) => (
              <div key={i} className="flex items-center gap-2">
                <select
                  value={row.col}
                  onChange={(e) =>
                    setFilterRows((prev) =>
                      prev.map((r, j) => (j === i ? { ...r, col: e.target.value } : r))
                    )
                  }
                  className="px-2 py-1 bg-gray-800 border border-gray-700 rounded text-xs focus:outline-none focus:border-blue-500 min-w-32"
                >
                  <option value="">— column —</option>
                  {columns.map((c) => (
                    <option key={c.name} value={c.name}>
                      {c.name}
                    </option>
                  ))}
                </select>
                <span className="text-xs text-gray-600">=</span>
                <input
                  type="text"
                  placeholder="value"
                  value={row.val}
                  onChange={(e) =>
                    setFilterRows((prev) =>
                      prev.map((r, j) => (j === i ? { ...r, val: e.target.value } : r))
                    )
                  }
                  onKeyDown={(e) => e.key === "Enter" && applyFilters()}
                  className="px-2 py-1 bg-gray-800 border border-gray-700 rounded text-xs focus:outline-none focus:border-blue-500 w-48"
                />
                {filterRows.length > 1 && (
                  <button
                    onClick={() =>
                      setFilterRows((prev) => prev.filter((_, j) => j !== i))
                    }
                    className="text-gray-600 hover:text-red-400 text-xs"
                  >
                    ✕
                  </button>
                )}
              </div>
            ))}
            <div className="flex items-center gap-2 mt-1">
              <button
                onClick={() => setFilterRows((prev) => [...prev, { col: "", val: "" }])}
                className="text-xs text-gray-500 hover:text-gray-300"
              >
                + Add condition
              </button>
              <div className="flex-1" />
              <button
                onClick={clearFilters}
                className="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-xs"
              >
                Clear
              </button>
              <button
                onClick={applyFilters}
                className="px-3 py-1 bg-blue-700 hover:bg-blue-600 rounded text-xs font-medium"
              >
                Apply
              </button>
            </div>
          </div>
        </div>
      )}

      {error && (
        <div className="px-4 py-2 bg-red-900/50 text-red-300 text-xs">
          {error}
        </div>
      )}

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-sm">
          <thead className="bg-gray-900 sticky top-0">
            <tr>
              <th className="px-3 py-2 text-left text-xs text-gray-500 font-medium w-20">
                Actions
              </th>
              {displayColumns.map((col) => {
                const colInfo = columns.find((c) => c.name === col);
                const isActive = sort?.col === col;
                return (
                  <th
                    key={col}
                    className="px-3 py-2 text-left text-xs text-gray-500 font-medium cursor-pointer select-none hover:text-gray-300 group"
                    onClick={() => {
                      setSort((s) => nextSort(s, col));
                      setPage(0);
                    }}
                  >
                    <span className={isActive ? "text-blue-400" : ""}>{col}</span>
                    {colInfo?.primaryKey && (
                      <span className="ml-1 text-yellow-600">PK</span>
                    )}
                    {colInfo && (
                      <span className="ml-1 text-gray-700">
                        {colInfo.dataType}
                      </span>
                    )}
                    <span className="ml-1">
                      {isActive
                        ? sort!.dir === "asc" ? "↑" : "↓"
                        : <span className="opacity-0 group-hover:opacity-30">↑</span>}
                    </span>
                  </th>
                );
              })}
            </tr>
          </thead>
          <tbody>
            {/* New row form */}
            {newRow && (
              <tr className="bg-green-950/30">
                <td className="px-3 py-1">
                  <div className="flex gap-1">
                    <button
                      onClick={handleCreate}
                      className="text-green-400 hover:text-green-300 text-xs"
                    >
                      Save
                    </button>
                    <button
                      onClick={() => setNewRow(null)}
                      className="text-gray-500 hover:text-gray-300 text-xs"
                    >
                      Cancel
                    </button>
                  </div>
                </td>
                {displayColumns.map((col) => (
                  <td key={col} className="px-3 py-1">
                    <input
                      type="text"
                      value={String(newRow[col] ?? "")}
                      onChange={(e) =>
                        setNewRow((prev) => ({
                          ...prev!,
                          [col]: e.target.value,
                        }))
                      }
                      className="w-full px-2 py-1 bg-gray-800 border border-gray-700 rounded text-xs focus:outline-none focus:border-green-500"
                    />
                  </td>
                ))}
              </tr>
            )}

            {/* Data rows */}
            {data?.rows.map((row, idx) => {
              const isEditing =
                editingRow &&
                columns
                  .filter((c) => c.primaryKey)
                  .every((c) => editingRow[c.name] === row[c.name]);

              return (
                <tr
                  key={idx}
                  className="border-t border-gray-800/50 hover:bg-gray-800/30"
                >
                  <td className="px-3 py-1">
                    <div className="flex gap-1">
                      {isEditing ? (
                        <>
                          <button
                            onClick={handleUpdate}
                            className="text-blue-400 hover:text-blue-300 text-xs"
                          >
                            Save
                          </button>
                          <button
                            onClick={() => setEditingRow(null)}
                            className="text-gray-500 hover:text-gray-300 text-xs"
                          >
                            Cancel
                          </button>
                        </>
                      ) : (
                        <>
                          <button
                            onClick={() => {
                              const r = { ...row } as Record<string, unknown>;
                              setEditingRow(r);
                              const raw: Record<string, string> = {};
                              for (const [k, v] of Object.entries(r)) {
                                if (v !== null && typeof v === "object") {
                                  raw[k] = JSON.stringify(v, null, 2);
                                }
                              }
                              setEditingRaw(raw);
                            }}
                            className="text-gray-600 hover:text-blue-400 text-xs"
                          >
                            Edit
                          </button>
                          <button
                            onClick={() =>
                              setRowToDelete(row as Record<string, unknown>)
                            }
                            className="text-gray-600 hover:text-red-400 text-xs"
                          >
                            Del
                          </button>
                        </>
                      )}
                    </div>
                  </td>
                  {displayColumns.map((col) => (
                    <td key={col} className="px-3 py-1 text-xs">
                      {isEditing ? (
                        col in editingRaw ? (
                          <textarea
                            value={editingRaw[col]}
                            onChange={(e) =>
                              setEditingRaw((prev) => ({ ...prev, [col]: e.target.value }))
                            }
                            rows={3}
                            className="w-full px-2 py-1 bg-gray-800 border border-gray-700 rounded text-xs font-mono focus:outline-none focus:border-blue-500 resize-y"
                          />
                        ) : (
                        <input
                          type="text"
                          value={String(editingRow![col] ?? "")}
                          onChange={(e) =>
                            setEditingRow((prev) => ({
                              ...prev!,
                              [col]: e.target.value,
                            }))
                          }
                          className="w-full px-2 py-1 bg-gray-800 border border-gray-700 rounded text-xs focus:outline-none focus:border-blue-500"
                          disabled={columns.find((c) => c.name === col)?.primaryKey}
                        />
                        )
                      ) : (
                        <span className="text-gray-300">
                          {row[col] === null ? (
                            <span className="text-gray-600 italic">NULL</span>
                          ) : typeof row[col] === "object" ? (
                            JSON.stringify(row[col])
                          ) : (
                            String(row[col])
                          )}
                        </span>
                      )}
                    </td>
                  ))}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      <ConfirmDialog
        open={!!rowToDelete}
        title="Delete row?"
        message="This will permanently delete the row. This cannot be undone."
        confirmLabel="Delete"
        destructive
        onConfirm={confirmDelete}
        onCancel={() => setRowToDelete(null)}
      />

      {/* Pagination */}
      <div className="flex items-center gap-2 px-4 py-2 bg-gray-900 border-t border-gray-800">
        <button
          onClick={() => setPage((p) => Math.max(0, p - 1))}
          disabled={page === 0}
          className="px-3 py-1 bg-gray-800 hover:bg-gray-700 rounded text-xs disabled:opacity-30"
        >
          Previous
        </button>
        <span className="text-xs text-gray-500">
          Page {page + 1}
        </span>
        <button
          onClick={() => setPage((p) => p + 1)}
          disabled={(data?.rows.length || 0) < pageSize}
          className="px-3 py-1 bg-gray-800 hover:bg-gray-700 rounded text-xs disabled:opacity-30"
        >
          Next
        </button>
      </div>
    </div>
  );
}
