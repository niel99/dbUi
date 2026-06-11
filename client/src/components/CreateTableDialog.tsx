import { Fragment, useState, useMemo } from "react";
import type { ColumnSpec, DBType, TableSpec } from "@/types";

interface CreateTableDialogProps {
  open: boolean;
  dbType: DBType;
  database: string;
  onCancel: () => void;
  onSubmit: (spec: TableSpec) => void;
}

const TYPE_SUGGESTIONS: Record<DBType, string[]> = {
  postgres: ["integer", "bigint", "text", "varchar(255)", "boolean", "timestamp", "uuid", "jsonb", "numeric"],
  mongodb: [],
  cassandra: ["int", "bigint", "text", "boolean", "timestamp", "uuid", "float", "double"],
  scylladb: ["int", "bigint", "text", "boolean", "timestamp", "uuid", "float", "double"],
};

function isTextType(dataType: string): boolean {
  const t = dataType.toLowerCase();
  return (
    t.startsWith("text") ||
    t.startsWith("varchar") ||
    t.startsWith("char") ||
    t.startsWith("citext")
  );
}

function emptyColumn(dbType: DBType): ColumnSpec {
  const types = TYPE_SUGGESTIONS[dbType];
  return {
    name: "",
    dataType: types[0] ?? "text",
    nullable: true,
    primaryKey: false,
  };
}

const IDENTIFIER_RE = /^[A-Za-z_][A-Za-z0-9_]*$/;

export function CreateTableDialog({
  open,
  dbType,
  database,
  onCancel,
  onSubmit,
}: CreateTableDialogProps) {
  const [name, setName] = useState("");
  const [schema, setSchema] = useState(dbType === "postgres" ? "public" : "");
  const [columns, setColumns] = useState<ColumnSpec[]>([
    { ...emptyColumn(dbType), name: "id", primaryKey: true, nullable: false },
  ]);
  const [expanded, setExpanded] = useState<Set<number>>(new Set());

  const isMongo = dbType === "mongodb";
  const isCql = dbType === "cassandra" || dbType === "scylladb";
  const typeOptions = TYPE_SUGGESTIONS[dbType];

  const errors = useMemo(() => {
    const list: string[] = [];
    if (!name.trim()) list.push("Table name is required.");
    else if (!IDENTIFIER_RE.test(name.trim()))
      list.push("Table name must start with a letter/underscore and contain only letters, digits, underscores.");

    if (!isMongo) {
      const seen = new Map<string, number>();
      let hasPk = false;
      columns.forEach((c, i) => {
        const n = c.name.trim();
        if (!n) list.push(`Column ${i + 1}: name is required.`);
        else if (!IDENTIFIER_RE.test(n))
          list.push(`Column "${n}": invalid identifier.`);
        else if (seen.has(n))
          list.push(`Duplicate column name "${n}".`);
        else seen.set(n, i);

        if (!c.dataType.trim()) list.push(`Column "${n || i + 1}": type is required.`);
        if (c.primaryKey) hasPk = true;

        if (c.minLength != null && c.maxLength != null && c.minLength > c.maxLength)
          list.push(`Column "${n}": minLength cannot exceed maxLength.`);

        if (c.pattern) {
          try {
            new RegExp(c.pattern);
          } catch {
            list.push(`Column "${n}": invalid regex pattern.`);
          }
        }
      });
      if (!hasPk) list.push("At least one column must be marked as primary key.");
    }
    return Array.from(new Set(list));
  }, [name, columns, isMongo]);

  if (!open) return null;

  const canSubmit = errors.length === 0;

  const updateColumn = (idx: number, patch: Partial<ColumnSpec>) => {
    setColumns((prev) => prev.map((c, i) => (i === idx ? { ...c, ...patch } : c)));
  };

  const addColumn = () => setColumns((prev) => [...prev, emptyColumn(dbType)]);

  const removeColumn = (idx: number) => {
    setColumns((prev) => prev.filter((_, i) => i !== idx));
    setExpanded((prev) => {
      const next = new Set<number>();
      prev.forEach((i) => {
        if (i < idx) next.add(i);
        else if (i > idx) next.add(i - 1);
      });
      return next;
    });
  };

  const toggleExpand = (idx: number) =>
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(idx)) next.delete(idx);
      else next.add(idx);
      return next;
    });

  const handleSubmit = () => {
    // Strip empty advanced fields so they're omitted from JSON.
    const cleaned = columns.map((c) => {
      const out: ColumnSpec = {
        name: c.name.trim(),
        dataType: c.dataType.trim(),
        nullable: c.nullable,
        primaryKey: c.primaryKey,
      };
      if (c.unique) out.unique = true;
      if (c.default?.trim()) out.default = c.default.trim();
      if (c.minLength != null) out.minLength = c.minLength;
      if (c.maxLength != null) out.maxLength = c.maxLength;
      if (c.pattern?.trim()) out.pattern = c.pattern.trim();
      return out;
    });
    onSubmit({
      name: name.trim(),
      schema: schema.trim() || undefined,
      columns: isMongo ? [] : cleaned,
    });
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
      onClick={onCancel}
    >
      <div
        className="w-full max-w-3xl max-h-[90vh] bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-800 p-6 shadow-xl flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <h3 className="text-lg font-semibold mb-1">
          Create {isMongo ? "collection" : "table"}
        </h3>
        <p className="text-xs text-gray-500 mb-4">
          in <span className="text-gray-700 dark:text-gray-300">{database}</span>
        </p>

        <div className="space-y-4 overflow-y-auto">
          <div className="grid grid-cols-3 gap-3">
            <div className={dbType === "postgres" ? "col-span-2" : "col-span-3"}>
              <label className="block text-sm text-gray-500 dark:text-gray-400 mb-1">Name</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full px-3 py-2 bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded text-sm focus:outline-none focus:border-blue-500"
                placeholder={isMongo ? "my_collection" : "my_table"}
                autoFocus
              />
            </div>
            {dbType === "postgres" && (
              <div>
                <label className="block text-sm text-gray-500 dark:text-gray-400 mb-1">Schema</label>
                <input
                  type="text"
                  value={schema}
                  onChange={(e) => setSchema(e.target.value)}
                  className="w-full px-3 py-2 bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded text-sm focus:outline-none focus:border-blue-500"
                />
              </div>
            )}
          </div>

          {isMongo ? (
            <p className="text-xs text-gray-500 italic">
              MongoDB collections are schemaless — columns aren't defined up front.
            </p>
          ) : (
            <>
              {isCql && (
                <div className="text-xs text-amber-700 dark:text-amber-400/80 bg-amber-50 dark:bg-amber-900/20 border border-amber-300 dark:border-amber-900/50 rounded px-3 py-2">
                  CQL ({dbType}) does not support <code>DEFAULT</code>,{" "}
                  <code>UNIQUE</code>, or <code>CHECK</code> constraints. Those
                  fields are disabled; only name, type, and primary key apply.
                </div>
              )}

              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="block text-sm text-gray-500 dark:text-gray-400">Columns</label>
                  <button
                    onClick={addColumn}
                    className="text-xs text-blue-400 hover:text-blue-300"
                  >
                    + Add column
                  </button>
                </div>

                <div className="border border-gray-200 dark:border-gray-800 rounded overflow-hidden">
                  <table className="w-full text-sm">
                    <thead className="bg-gray-100 dark:bg-gray-800/60">
                      <tr>
                        <th className="w-6" />
                        <th className="px-2 py-1.5 text-left text-xs text-gray-500 font-medium">Name</th>
                        <th className="px-2 py-1.5 text-left text-xs text-gray-500 font-medium">Type</th>
                        <th className="px-2 py-1.5 text-center text-xs text-gray-500 font-medium w-12">Null</th>
                        <th className="px-2 py-1.5 text-center text-xs text-gray-500 font-medium w-10">PK</th>
                        <th className="w-8" />
                      </tr>
                    </thead>
                    <tbody>
                      {columns.map((col, idx) => {
                        const isOpen = expanded.has(idx);
                        const pgTextType = dbType === "postgres" && isTextType(col.dataType);
                        return (
                          <Fragment key={idx}>
                            <tr className="border-t border-gray-200 dark:border-gray-800">
                              <td className="text-center">
                                <button
                                  onClick={() => toggleExpand(idx)}
                                  className="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 text-xs w-5"
                                  title="Advanced"
                                >
                                  {isOpen ? "▾" : "▸"}
                                </button>
                              </td>
                              <td className="px-2 py-1">
                                <input
                                  type="text"
                                  value={col.name}
                                  onChange={(e) => updateColumn(idx, { name: e.target.value })}
                                  className="w-full px-2 py-1 bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded text-xs focus:outline-none focus:border-blue-500"
                                  placeholder="column_name"
                                />
                              </td>
                              <td className="px-2 py-1">
                                <input
                                  type="text"
                                  value={col.dataType}
                                  onChange={(e) => updateColumn(idx, { dataType: e.target.value })}
                                  list={`types-${dbType}`}
                                  className="w-full px-2 py-1 bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded text-xs focus:outline-none focus:border-blue-500"
                                />
                                <datalist id={`types-${dbType}`}>
                                  {typeOptions.map((t) => (
                                    <option key={t} value={t} />
                                  ))}
                                </datalist>
                              </td>
                              <td className="px-2 py-1 text-center">
                                <input
                                  type="checkbox"
                                  checked={col.nullable}
                                  onChange={(e) => updateColumn(idx, { nullable: e.target.checked })}
                                />
                              </td>
                              <td className="px-2 py-1 text-center">
                                <input
                                  type="checkbox"
                                  checked={col.primaryKey}
                                  onChange={(e) => updateColumn(idx, { primaryKey: e.target.checked })}
                                />
                              </td>
                              <td className="px-1 py-1 text-center">
                                {columns.length > 1 && (
                                  <button
                                    onClick={() => removeColumn(idx)}
                                    className="text-gray-400 dark:text-gray-600 hover:text-red-400 text-xs"
                                    title="Remove"
                                  >
                                    ×
                                  </button>
                                )}
                              </td>
                            </tr>
                            {isOpen && (
                              <tr className="border-t border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-gray-900/40">
                                <td />
                                <td colSpan={5} className="px-3 py-3">
                                  <div className="grid grid-cols-2 gap-3">
                                    <div>
                                      <label className="block text-[11px] text-gray-500 mb-1">
                                        Default value
                                        {isCql && <span className="text-gray-400 dark:text-gray-700"> (unsupported)</span>}
                                      </label>
                                      <input
                                        type="text"
                                        value={col.default ?? ""}
                                        disabled={isCql}
                                        onChange={(e) => updateColumn(idx, { default: e.target.value })}
                                        placeholder="e.g. 0, now(), 'pending'"
                                        className="w-full px-2 py-1 bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded text-xs focus:outline-none focus:border-blue-500 disabled:opacity-40"
                                      />
                                      <p className="text-[10px] text-gray-400 dark:text-gray-600 mt-0.5">
                                        Raw SQL expression — quote strings.
                                      </p>
                                    </div>
                                    <div className="flex items-end">
                                      <label className="inline-flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                                        <input
                                          type="checkbox"
                                          checked={!!col.unique}
                                          disabled={isCql}
                                          onChange={(e) => updateColumn(idx, { unique: e.target.checked })}
                                        />
                                        Unique
                                        {isCql && <span className="text-gray-400 dark:text-gray-700">(unsupported)</span>}
                                      </label>
                                    </div>

                                    {pgTextType && (
                                      <>
                                        <div>
                                          <label className="block text-[11px] text-gray-500 mb-1">Min length</label>
                                          <input
                                            type="number"
                                            min={0}
                                            value={col.minLength ?? ""}
                                            onChange={(e) =>
                                              updateColumn(idx, {
                                                minLength:
                                                  e.target.value === ""
                                                    ? undefined
                                                    : Math.max(0, parseInt(e.target.value) || 0),
                                              })
                                            }
                                            className="w-full px-2 py-1 bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded text-xs focus:outline-none focus:border-blue-500"
                                          />
                                        </div>
                                        <div>
                                          <label className="block text-[11px] text-gray-500 mb-1">Max length</label>
                                          <input
                                            type="number"
                                            min={0}
                                            value={col.maxLength ?? ""}
                                            onChange={(e) =>
                                              updateColumn(idx, {
                                                maxLength:
                                                  e.target.value === ""
                                                    ? undefined
                                                    : Math.max(0, parseInt(e.target.value) || 0),
                                              })
                                            }
                                            className="w-full px-2 py-1 bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded text-xs focus:outline-none focus:border-blue-500"
                                          />
                                        </div>
                                        <div className="col-span-2">
                                          <label className="block text-[11px] text-gray-500 mb-1">
                                            Regex pattern (POSIX)
                                          </label>
                                          <input
                                            type="text"
                                            value={col.pattern ?? ""}
                                            onChange={(e) => updateColumn(idx, { pattern: e.target.value })}
                                            placeholder="e.g. ^[A-Za-z0-9_]+$"
                                            className="w-full px-2 py-1 bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded text-xs font-mono focus:outline-none focus:border-blue-500"
                                          />
                                          <p className="text-[10px] text-gray-400 dark:text-gray-600 mt-0.5">
                                            Enforced as <code>CHECK (col ~ 'pattern')</code>.
                                          </p>
                                        </div>
                                      </>
                                    )}
                                    {!pgTextType && dbType === "postgres" && (
                                      <div className="col-span-2 text-[11px] text-gray-400 dark:text-gray-600 italic">
                                        Length / pattern rules apply only to text-type columns.
                                      </div>
                                    )}
                                    {isCql && (
                                      <div className="col-span-2 text-[11px] text-gray-400 dark:text-gray-600 italic">
                                        Length / pattern rules require CHECK constraints, which CQL does not support.
                                      </div>
                                    )}
                                  </div>
                                </td>
                              </tr>
                            )}
                          </Fragment>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              </div>
            </>
          )}

          {errors.length > 0 && (
            <ul className="text-xs text-red-400 bg-red-50 dark:bg-red-900/20 border border-red-300 dark:border-red-900/50 rounded px-3 py-2 space-y-0.5">
              {errors.map((e, i) => (
                <li key={i}>• {e}</li>
              ))}
            </ul>
          )}
        </div>

        <div className="flex justify-end gap-3 mt-6">
          <button
            onClick={onCancel}
            className="px-4 py-2 bg-gray-100 dark:bg-gray-800 hover:bg-gray-300 dark:hover:bg-gray-700 rounded text-sm font-medium transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={!canSubmit}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded text-sm font-medium transition-colors disabled:opacity-50"
          >
            Review & Create
          </button>
        </div>
      </div>
    </div>
  );
}
