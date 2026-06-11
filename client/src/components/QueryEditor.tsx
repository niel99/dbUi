import { useState } from "react";
import type { QueryResult } from "@/types";
import { api } from "@/api/client";

interface QueryEditorProps {
  connectionId: string;
}

export function QueryEditor({ connectionId }: QueryEditorProps) {
  const [query, setQuery] = useState("");
  const [result, setResult] = useState<QueryResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleExecute = async () => {
    if (!query.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const res = await api.db.rawQuery(connectionId, query);
      if (res.error) {
        setError(res.error);
        setResult(null);
      } else {
        setResult(res);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Query failed");
      setResult(null);
    } finally {
      setLoading(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
      handleExecute();
    }
  };

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      {/* Query input */}
      <div className="flex flex-col border-b border-gray-800">
        <textarea
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Enter your query... (Cmd+Enter to execute)"
          className="w-full h-40 p-4 bg-gray-950 text-sm font-mono text-gray-200 resize-none focus:outline-none placeholder:text-gray-700"
          spellCheck={false}
        />
        <div className="flex items-center gap-2 px-4 py-2 bg-gray-900">
          <button
            onClick={handleExecute}
            disabled={loading || !query.trim()}
            className="px-4 py-1.5 bg-blue-600 hover:bg-blue-700 rounded text-xs font-medium transition-colors disabled:opacity-50"
          >
            {loading ? "Executing..." : "Execute"}
          </button>
          <span className="text-xs text-gray-600">Cmd+Enter</span>
          {result && (
            <span className="text-xs text-gray-500">
              {result.rowsAffected} row(s) returned
            </span>
          )}
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="px-4 py-3 bg-red-900/30 border-b border-red-800 text-red-300 text-sm font-mono">
          {error}
        </div>
      )}

      {/* Results */}
      {result && result.rows.length > 0 && (
        <div className="flex-1 overflow-auto">
          <table className="w-full text-sm">
            <thead className="bg-gray-900 sticky top-0">
              <tr>
                {result.columns?.map((col) => (
                  <th
                    key={col}
                    className="px-3 py-2 text-left text-xs text-gray-500 font-medium"
                  >
                    {col}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {result.rows.map((row, idx) => (
                <tr
                  key={idx}
                  className="border-t border-gray-800/50 hover:bg-gray-800/30"
                >
                  {result.columns?.map((col) => (
                    <td key={col} className="px-3 py-1.5 text-xs text-gray-300">
                      {row[col] === null ? (
                        <span className="text-gray-600 italic">NULL</span>
                      ) : typeof row[col] === "object" ? (
                        JSON.stringify(row[col])
                      ) : (
                        String(row[col])
                      )}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {result && result.rows.length === 0 && !error && (
        <div className="flex-1 flex items-center justify-center text-gray-500 text-sm">
          Query executed successfully. No rows returned.
        </div>
      )}
    </div>
  );
}
