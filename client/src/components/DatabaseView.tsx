import { useState, useEffect } from "react";
import type { Connection } from "@/types";
import { DataTable } from "@/components/DataTable";
import { QueryEditor } from "@/components/QueryEditor";
import { SchemaView } from "@/components/SchemaView";
import { IndexesView } from "@/components/IndexesView";

type Tab = "data" | "schema" | "indexes" | "query";

interface DatabaseViewProps {
  connection: Connection;
  selectedTable: string | null;
}

export function DatabaseView({ connection, selectedTable }: DatabaseViewProps) {
  const [activeTab, setActiveTab] = useState<Tab>(selectedTable ? "data" : "query");

  useEffect(() => {
    if (selectedTable) setActiveTab((t) => (t === "query" ? "data" : t));
    else setActiveTab("query");
  }, [selectedTable]);

  const tableTabs: { id: Tab; label: string }[] = [
    { id: "data", label: "Data" },
    { id: "schema", label: "Schema" },
    { id: "indexes", label: "Indexes" },
  ];

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      <div className="flex items-center gap-1 px-3 py-2 bg-gray-900 border-b border-gray-800">
        <div className="text-sm text-gray-400">
          <span className="text-gray-500">{connection.name}</span>
          {connection.database && (
            <>
              <span className="text-gray-700 mx-1">/</span>
              <span className="text-gray-300">{connection.database}</span>
            </>
          )}
          {selectedTable && activeTab !== "query" && (
            <>
              <span className="text-gray-700 mx-1">/</span>
              <span className="text-white">{selectedTable}</span>
            </>
          )}
        </div>
        <div className="flex-1" />
        {selectedTable &&
          tableTabs.map((t) => (
            <button
              key={t.id}
              onClick={() => setActiveTab(t.id)}
              className={`px-3 py-1 rounded text-xs font-medium ${
                activeTab === t.id
                  ? "bg-gray-700 text-white"
                  : "text-gray-500 hover:text-gray-300"
              }`}
            >
              {t.label}
            </button>
          ))}
        <button
          onClick={() => setActiveTab("query")}
          className={`px-3 py-1 rounded text-xs font-medium ${
            activeTab === "query"
              ? "bg-gray-700 text-white"
              : "text-gray-500 hover:text-gray-300"
          }`}
        >
          Query
        </button>
      </div>

      <div className="flex-1 flex overflow-hidden">
        {activeTab === "query" ? (
          <QueryEditor connectionId={connection.id} />
        ) : !selectedTable ? (
          <div className="flex-1 flex items-center justify-center text-gray-500">
            Select a table from the sidebar to view data.
          </div>
        ) : activeTab === "schema" ? (
          <SchemaView connectionId={connection.id} table={selectedTable} dbType={connection.type} />
        ) : activeTab === "indexes" ? (
          <IndexesView connectionId={connection.id} table={selectedTable} />
        ) : (
          <DataTable connectionId={connection.id} table={selectedTable} />
        )}
      </div>
    </div>
  );
}
