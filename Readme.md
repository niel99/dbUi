# dbUI

A local database UI with a React frontend and a Go backend that supports multiple adapters.

Supported database types:
- PostgreSQL
- MongoDB
- Cassandra
- ScyllaDB

## What it does

- Save and manage connection profiles
- Connect/disconnect and test connections
- Browse databases and tables/collections
- Create/drop tables
- Inspect schemas/columns and indexes
- CRUD operations on table data
- Run raw queries

## Tech stack

- Frontend: React 19, TypeScript, Vite, Tailwind CSS
- Backend: Go, Chi router

## Project structure

```text
.
├── client/   # React app (Vite)
├── server/   # Go API server
├── dist/     # Build output (server binary / client build)
└── package.json (root scripts for full-stack dev/build/test)
```

## Prerequisites

- Node.js 20+
- npm 10+
- Go 1.22+

## Setup

Install root dependencies:

```bash
npm install
```

Install client dependencies:

```bash
cd client && npm install
```

## Run in development

From the repo root:

```bash
npm run dev
```

This starts:
- Backend on `http://localhost:8080`
- Frontend on `http://localhost:5173`

The Vite dev server proxies `/api` requests to the backend.

## Build

Build both backend and frontend:

```bash
npm run build
```

Artifacts:
- Server binary: `dist/dbui-server`
- Client build: `client/dist`

## Test

Run all tests:

```bash
npm test
```

Run only backend tests:

```bash
npm run test:server
```

Run only frontend tests:

```bash
npm run test:client
```

## Root scripts

- `npm run dev` - run backend and frontend together
- `npm run dev:server` - run Go backend
- `npm run dev:client` - run Vite frontend
- `npm run build` - build backend and frontend
- `npm run build:server` - build Go server binary
- `npm run build:client` - build frontend assets
- `npm run test` - run backend + frontend tests

## API overview

Main route groups:
- `/api/connections` for connection lifecycle (create/list/update/delete, connect/disconnect/test)
- `/api/db/{connID}` for data operations (databases, tables, describe, indexes, CRUD, raw query)

Health check:
- `GET /health`

## Notes

- CORS allows `http://localhost:5173` and `http://localhost:3000`.
- Server port defaults to `8080` and can be overridden with `PORT`.
