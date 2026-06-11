# dbUI Design

## Overview

dbUI is a web-based database management tool that presents a uniform UI over heterogeneous databases (PostgreSQL, MongoDB, Cassandra, ScyllaDB). It is split into a Go backend that brokers all database access and a React/TypeScript frontend that consumes a single REST API regardless of the underlying engine.

The core design problem is **abstracting over fundamentally different data models** (relational rows, documents, wide-column partitions) without leaking engine-specific concepts into the UI, while still letting power users drop down to raw queries when needed.

## Goals & Non-Goals

**Goals**
- One UI for browsing schema, editing data, and running queries against any supported database.
- A small, well-defined plugin surface so a new database can be added by implementing one Go interface.
- Safe defaults: passwords never round-tripped to clients in list views, connections reused across requests.
- Test-first: every adapter and handler has a Go test; every component has a Vitest test.

**Non-Goals**
- Not a replacement for full-featured IDEs (DBeaver, DataGrip). No ER diagrams, query plans, profiling.
- Not multi-tenant or auth-gated — assumes a trusted local user.
- No persistence of saved connections across server restarts (yet) — connections live in process memory.
- No migrations, schema diff, or change-tracking tooling.

## High-Level Architecture

```
┌──────────────────────┐        HTTP/JSON         ┌────────────────────────┐
│   React Client       │ ───────────────────────► │   Go Server (chi)      │
│   (Vite, :5173)      │ ◄─── /api proxy ────────►│   :8080                │
│                      │                          │                         │
│  Sidebar             │                          │  handler/  ─ routes    │
│  ConnectionForm      │                          │  connection/ ─ manager │
│  DatabaseView        │                          │  registry/ ─ plugins   │
│  DataTable           │                          │  adapter/  ─ interface │
│  QueryEditor         │                          │    ├─ postgres/        │
│  SchemaView          │                          │    ├─ mongodb/         │
│  IndexesView         │                          │    └─ cassandra/       │
└──────────────────────┘                          └────────────────────────┘
                                                          │
                                                          ▼
                                             ┌──────────────────────────┐
                                             │  PG / Mongo / C* / Scylla│
                                             └──────────────────────────┘
```

Vite proxies `/api/*` to `:8080` in development; in production the Go binary can serve the built static assets alongside the API.

## Backend

### The Adapter Interface

The single most important type is [`DatabaseAdapter`](server/internal/adapter/adapter.go) in [server/internal/adapter/adapter.go](server/internal/adapter/adapter.go). All backend logic above the adapter is engine-agnostic; everything below is engine-specific. The interface surface is intentionally narrow and groups operations into four buckets:

1. **Lifecycle** — `Connect`, `Disconnect`, `Ping`.
2. **Schema discovery** — `ListDatabases`, `ListTables`, `DescribeTable`, `ListIndexes`.
3. **Schema mutation** — `CreateTable`, `DropTable`.
4. **Data access** — `Read`, `Create`, `Update`, `Delete`, `RawQuery`.

CRUD methods all take a single [`CRUDRequest`](server/internal/models/models.go) and return a single [`QueryResult`](server/internal/models/models.go). This shape was chosen so that handlers can be written generically (see `doCRUD` in [server/internal/handler/handler.go](server/internal/handler/handler.go)) and so that the wire format the client sees is the same for every database.

The cost of this generality is that some abstraction leaks through the optional fields on `TableInfo` (`Schema` for SQL, `Keyspace` for Cassandra) and `CRUDRequest` (`Data`, `Where` as untyped maps). This was a deliberate trade — pushing those into the type system would either explode the interface or force lossy mappings.

### Plugin Registry

[`registry.Registry`](server/internal/registry/registry.go) is a `map[DBType]AdapterFactory` guarded by a `sync.RWMutex`. Adapters are registered at startup in [server/cmd/server/main.go](server/cmd/server/main.go); adding a new engine is purely additive.

ScyllaDB reuses the Cassandra adapter — both speak CQL — by registering the same factory under a different `DBType`. The factory takes the `DBType` so the adapter knows which discriminator to report from `Type()`.

### Connection Manager

[`connection.Manager`](server/internal/connection/manager.go) owns two maps:
- `connections` — saved connection configs (`map[id]Connection`).
- `active` — live adapter instances for connected sessions (`map[id]DatabaseAdapter`).

Key behaviors:
- `Connect` is idempotent: if there is already an active adapter, it `Ping`s it; healthy connections are returned, stale ones are torn down and rebuilt.
- `Remove` disconnects before deleting so we never leak connections.
- `SwitchDatabase` always reconnects — engines like PostgreSQL bind a database at connect time, so a true switch requires a new connection. This is uniform across engines for simplicity even when not strictly required.
- `List` strips passwords. The detail endpoint returns the full record because the form needs it for editing; this is acceptable for a local-trust deployment but would need tightening in a hosted context.

State is in-memory only. Restarting the server loses saved connections — fine for the current single-user local model, but the obvious place to add persistence (SQLite next to the binary) when needed.

### HTTP Layer

The router (chi) is built in [`Handler.Routes`](server/internal/handler/handler.go) and splits into two route groups:

- `/api/connections` — CRUD over saved connections plus `/connect`, `/disconnect`, `/test`.
- `/api/db/{connID}` — operations against an active connection. The `connID` URL param resolves to an adapter via `manager.GetActive`; if no active session exists, the handler returns 400, telling the client to reconnect.

`TestConnection` returns `200 OK` with `{success: false, error: ...}` rather than a 4xx/5xx on failure. This is intentional: the client treats a test failure as expected feedback, not as an HTTP error. Real errors (bad request body, missing connection) still use the standard error path.

The `doCRUD` helper folds all four CRUD handlers into one body — they differ only in which adapter method they invoke.

## Frontend

### Component Map

| Component | Responsibility |
|-----------|----------------|
| [App.tsx](client/src/App.tsx) | Top-level layout: sidebar + active workspace. Holds the active connection ID. |
| [Sidebar.tsx](client/src/components/Sidebar.tsx) | List/connect/disconnect/delete saved connections. |
| [ConnectionForm.tsx](client/src/components/ConnectionForm.tsx) | Create/edit connection; "Test" button hits `/test`. |
| [DatabaseView.tsx](client/src/components/DatabaseView.tsx) | Table list + tab switcher between data, schema, indexes, query. |
| [DataTable.tsx](client/src/components/DataTable.tsx) | Paginated row view with inline edit/insert/delete. |
| [SchemaView.tsx](client/src/components/SchemaView.tsx) | Read-only column listing for a table. |
| [IndexesView.tsx](client/src/components/IndexesView.tsx) | Index listing for a table. |
| [CreateTableDialog.tsx](client/src/components/CreateTableDialog.tsx) | Form-driven `CreateTable` flow. |
| [QueryEditor.tsx](client/src/components/QueryEditor.tsx) | Raw query box (Cmd+Enter to run); renders `QueryResult`. |
| [ConfirmDialog.tsx](client/src/components/ConfirmDialog.tsx) | Reusable destructive-action confirmation. |

### API Client

[client/src/api/client.ts](client/src/api/client.ts) is a thin typed wrapper over `fetch` against `/api`. All methods return parsed JSON typed against [client/src/types/index.ts](client/src/types/index.ts), which mirrors `server/internal/models/models.go`. Keeping the two type definitions in sync by hand is the current approach; if drift becomes painful, generating TS from Go (via `gen` or OpenAPI) is the path forward.

### State

There is no global store (no Redux, Zustand). Each view manages its own data; the active connection ID is held at the `App` level and threaded down. This is sufficient for the current depth of the UI tree; introducing a store before there's evidence of prop-drilling pain would be premature.

## Request Flow Example: Editing a Row

1. User edits a cell in `DataTable` and confirms.
2. Client calls `apiClient.updateRows(connID, table, { where: {id: 7}, data: {name: "x"} })`.
3. Request hits `POST /api/db/{connID}/tables/{table}/update`.
4. `Handler.UpdateRows` → `doCRUD` → `manager.GetActive(connID)` → adapter's `Update`.
5. Adapter translates the `CRUDRequest` into engine-native syntax (parameterized SQL for PG, BSON filter+update for Mongo, CQL for C*).
6. `QueryResult` flows back; `DataTable` refreshes the page.

`Read` follows the same path with pagination via `Limit`/`Offset` on `CRUDRequest`.

## Adding a New Database

1. Create `server/internal/adapter/<name>/<name>.go` implementing `DatabaseAdapter`. Tests next to it.
2. Add a `DBType` constant in `server/internal/models/models.go`.
3. Register the factory in `server/cmd/server/main.go`.
4. Add the type to `DB_TYPES` in [client/src/components/ConnectionForm.tsx](client/src/components/ConnectionForm.tsx).

No other code changes. The handler, manager, and UI views are engine-agnostic and pick up the new type automatically.

## Testing Strategy

- **Backend** uses Go's standard `testing` package. The handler layer is tested against a mock adapter — see [server/internal/handler/handler_test.go](server/internal/handler/handler_test.go) — which lets us verify routing, payload shapes, and error propagation without spinning up real databases. Adapter packages are tested individually with engine-specific approaches.
- **Frontend** uses Vitest + React Testing Library. Component tests stub the API client and assert on rendered output and user interactions.
- TDD is the working norm: write the test, then the implementation.

## Known Trade-offs & Future Work

- **In-memory connections.** Saved connections are lost on server restart. SQLite-backed persistence is the obvious next step.
- **No auth.** The server trusts any caller. Acceptable locally, blocking for any hosted deployment.
- **Untyped `Data`/`Where` maps.** Powerful and uniform, but pushes validation into adapters. A typed query DSL is a larger redesign that's not currently justified.
- **Type duplication across Go and TS.** Manual sync today. Codegen if it starts to bite.
- **No streaming for large result sets.** `Read` returns one page at a time; large `RawQuery` results are buffered fully into a `QueryResult`. Streaming would require an interface change.
- **Schema mutation is minimal.** `CreateTable`/`DropTable` only — no alters, renames, or migration tooling.
