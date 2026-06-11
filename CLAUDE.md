# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

dbUI is a web-based database management tool supporting PostgreSQL, MongoDB, Cassandra, and ScyllaDB. It provides a connection manager, schema browser, data table with inline CRUD, and a raw query editor.

## Architecture

**Monorepo** with two independent packages:

- `server/` — Go backend with a plugin-based adapter architecture
- `client/` — React + TypeScript + Vite frontend

### Backend (Go)

The backend uses a **plugin/adapter pattern** where each database type implements a common `DatabaseAdapter` interface (`server/internal/adapter/adapter.go`). This is the central abstraction — all CRUD and schema operations go through it.

Key packages:
- `internal/adapter/` — Interface definition + implementations per DB (`postgres/`, `mongodb/`, `cassandra/`)
- `internal/registry/` — Plugin registry that maps `DBType` → `AdapterFactory`. New databases are added by implementing the interface and registering a factory
- `internal/connection/` — Manages saved connections and active database sessions (connect/disconnect/reuse)
- `internal/handler/` — HTTP handlers (chi router) exposing REST API for connections and CRUD
- `internal/models/` — Shared types (`Connection`, `TableInfo`, `Column`, `QueryResult`, `CRUDRequest`)
- `cmd/server/` — Entry point, wires registry + adapters + router

ScyllaDB uses the Cassandra adapter (same CQL protocol) with a different `DBType` discriminator.

### Frontend (React)

- `src/api/client.ts` — API client wrapping all backend endpoints
- `src/components/Sidebar.tsx` — Connection list with connect/disconnect/delete
- `src/components/ConnectionForm.tsx` — Create/edit connection with test button
- `src/components/DatabaseView.tsx` — Table browser + tab switching (data/query)
- `src/components/DataTable.tsx` — Paginated data table with inline edit, insert, delete
- `src/components/QueryEditor.tsx` — Raw query execution (Cmd+Enter)
- `src/types/` — TypeScript types mirroring backend models

Vite proxies `/api` requests to the Go backend at `:8080` during development.

## Commands

```bash
# Run both frontend and backend in development
npm run dev

# Run individually
npm run dev:server    # Go server on :8080
npm run dev:client    # Vite on :5173

# Run all tests
npm test

# Run tests individually
npm run test:server                       # All Go tests
npm run test:client                       # All Vitest tests
cd server && go test ./internal/handler/  # Single Go package
cd client && npx vitest run src/components/Sidebar.test.tsx  # Single test file

# Build
npm run build          # Both
npm run build:server   # Binary → dist/dbui-server
npm run build:client   # Static → client/dist/
```

## Development Approach

This project follows **test-driven development**. Write tests before implementations. Backend tests use Go's standard `testing` package with mock adapters. Frontend tests use Vitest + React Testing Library.

## Adding a New Database Adapter

1. Create `server/internal/adapter/<name>/<name>.go` implementing `adapter.DatabaseAdapter`
2. Add the `DBType` constant in `server/internal/models/models.go`
3. Register the factory in `cmd/server/main.go`
4. Add the type option in `client/src/components/ConnectionForm.tsx` `DB_TYPES` array
