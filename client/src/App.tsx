import { useState } from "react";
import type { Connection } from "@/types";
import { Sidebar } from "@/components/Sidebar";
import { ConnectionForm } from "@/components/ConnectionForm";
import { DatabaseView } from "@/components/DatabaseView";

type Selection = { connection: Connection; table: string | null };

export default function App() {
  const [selection, setSelection] = useState<Selection | null>(null);
  const [showConnectionForm, setShowConnectionForm] = useState(false);
  const [editConnection, setEditConnection] = useState<Connection | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);

  const activeSelection = selection
    ? { connectionId: selection.connection.id, table: selection.table ?? "" }
    : null;

  return (
    <div className="flex h-screen bg-gray-100 text-gray-900 dark:bg-gray-950 dark:text-gray-100">
      <Sidebar
        refreshKey={refreshKey}
        activeSelection={activeSelection}
        onSelectTable={(conn, table) => setSelection({ connection: conn, table })}
        onOpenQuery={(conn) => setSelection({ connection: conn, table: null })}
        onNewConnection={() => {
          setEditConnection(null);
          setShowConnectionForm(true);
        }}
        onEditConnection={(conn) => {
          setEditConnection(conn);
          setShowConnectionForm(true);
        }}
      />

      <main className="flex-1 flex flex-col overflow-hidden">
        {showConnectionForm ? (
          <ConnectionForm
            connection={editConnection}
            onSaved={(conn) => {
              setShowConnectionForm(false);
              setSelection({ connection: conn, table: null });
              setRefreshKey((k) => k + 1);
            }}
            onCancel={() => setShowConnectionForm(false)}
          />
        ) : selection ? (
          <DatabaseView
            connection={selection.connection}
            selectedTable={selection.table}
          />
        ) : (
          <div className="flex-1 flex items-center justify-center text-gray-400 dark:text-gray-500">
            <div className="text-center">
              <h2 className="text-2xl font-semibold mb-2">dbUI</h2>
              <p>Select a connection or create a new one to get started.</p>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
