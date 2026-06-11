import { useEffect, useState } from "react";
import type { IndexInfo } from "@/types";
import { api } from "@/api/client";

interface IndexesViewProps {
  connectionId: string;
  table: string;
}

export function IndexesView({ connectionId, table }: IndexesViewProps) {
  const [indexes, setIndexes] = useState<IndexInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    setError(null);
    api.db
      .indexes(connectionId, table)
      .then((i) => setIndexes(i ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : "Failed to load indexes"))
      .finally(() => setLoading(false));
  }, [connectionId, table]);

  if (loading) {
    return <div className="flex-1 flex items-center justify-center text-gray-500">Loading indexes…</div>;
  }
  if (error) {
    return <div className="flex-1 flex items-center justify-center text-red-400">{error}</div>;
  }
  if (indexes.length === 0) {
    return <div className="flex-1 flex items-center justify-center text-gray-500">No indexes on this table.</div>;
  }

  return (
    <div className="flex-1 overflow-auto p-4">
      <table className="w-full text-sm">
        <thead className="bg-gray-900 sticky top-0">
          <tr>
            <th className="px-3 py-2 text-left text-xs text-gray-500 font-medium">Name</th>
            <th className="px-3 py-2 text-left text-xs text-gray-500 font-medium">Columns</th>
            <th className="px-3 py-2 text-left text-xs text-gray-500 font-medium">Attributes</th>
            <th className="px-3 py-2 text-left text-xs text-gray-500 font-medium">Definition</th>
          </tr>
        </thead>
        <tbody>
          {indexes.map((ix) => (
            <tr key={ix.name} className="border-t border-gray-800/50 hover:bg-gray-800/30 align-top">
              <td className="px-3 py-1.5 text-gray-200">{ix.name}</td>
              <td className="px-3 py-1.5 text-gray-400 font-mono text-xs">
                {ix.columns?.join(", ") || <span className="text-gray-600 italic">—</span>}
              </td>
              <td className="px-3 py-1.5 text-xs space-x-2">
                {ix.primary && <span className="text-yellow-500">PRIMARY</span>}
                {ix.unique && <span className="text-blue-400">UNIQUE</span>}
                {!ix.primary && !ix.unique && <span className="text-gray-600">—</span>}
              </td>
              <td className="px-3 py-1.5 text-gray-500 font-mono text-xs break-all">
                {ix.definition || <span className="text-gray-700 italic">—</span>}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
