# Carina

A file storage and sharing service. Upload files, manage them securely, and share them via expiring links — built as a REST API on Go, Postgres, Redis, and S3-compatible object storage.

## What it does

Carina lets an authenticated user upload files, keep track of them, and generate share links so others can access a specific file without needing an account. Under the hood it separates concerns deliberately: Postgres holds durable metadata, Redis holds short-lived coordination state (upload progress, rate limits), and object storage (S3 or MinIO) holds the actual file bytes.

## Features

- **Auth** — registration and login with JWT access/refresh tokens; refresh tokens are revocable server-side
- **File upload/download** — authenticated file storage with ownership checks
- **Sharing** — generate expiring, revocable share links for unauthenticated access to a specific file
- **Upload sessions** — Redis-backed progress tracking for in-flight uploads
- **Rate limiting** — Redis-backed request limiting on auth and upload endpoints
- **Background processing** — async thumbnail generation and malware scanning on uploaded files

## Tech stack

| Layer | Choice |
|---|---|
| Language | Go |
| API | REST (documented in `api/openapi.yaml`) |
| Relational store | PostgreSQL |
| Ephemeral store | Redis |
| Object storage | S3-compatible (MinIO for local dev, AWS S3 in production) |
| Auth | JWT (access + refresh) |
| Local dev environment | Docker Compose |

## Architecture

Carina follows a feature-based (vertical slice) structure — each feature owns its handler, service, and repository rather than splitting by technical layer. See [`docs/architecture.md`](docs/architecture.md) for the full breakdown, including data flow, store responsibilities, and the reasoning behind key decisions.

## Project structure

```
carina/
├── api/                # OpenAPI spec
├── cmd/api/            # entrypoint
├── deployments/        # Docker Compose + Dockerfile
├── docs/                # architecture, setup, deployment docs, ADRs
├── internal/
│   ├── auth/            # registration, login, JWT issuance/verification
│   ├── config/          # typed config structs, env loading
│   ├── file/             # file metadata + storage client
│   ├── middleware/       # auth check, logging, panic recovery
│   ├── platform/         # Postgres + Redis connection setup
│   ├── ratelimit/        # Redis-backed request limiting
│   ├── server/            # HTTP server + router
│   ├── sharing/           # share link creation, revocation, public access
│   ├── upload/            # upload session + progress tracking (Redis)
│   └── worker/            # thumbnail generation, malware scanning
├── pkg/validator/         # shared input validation helpers
├── scripts/                # migration and seed scripts
└── test/integration/        # end-to-end tests
```

Full file-level tree is in [`docs/architecture.md`](docs/architecture.md).

## Getting started

### Prerequisites
- Go (see `go.mod` for version)
- Docker + Docker Compose

### 1. Clone and configure
```bash
git clone <repo-url> carina
cd carina
cp .env.example .env
```
Fill in `.env` — see [Configuration](#configuration) below for what's required.

### 2. Start the stack
```bash
docker-compose -f deployments/docker-compose.yml up
```
This brings up Postgres, Redis, MinIO, and the app itself.

### 3. Run migrations
```bash
make migrate
```

### 4. Verify it's running
```bash
curl http://localhost:8080/health
```
Should return `200` with Postgres and Redis connectivity confirmed.

## Configuration

All configuration is environment-based and loaded once at startup via `internal/config`. Required variables (see `.env.example` for the full list and defaults):

| Group | Examples |
|---|---|
| Server | `SERVER_PORT`, `SERVER_READ_TIMEOUT`, `SERVER_SHUTDOWN_TIMEOUT` |
| Postgres | `DATABASE_URL`, `DB_MAX_CONNS` |
| Redis | `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_DB` |
| Auth | `JWT_SECRET`, `ACCESS_TOKEN_TTL`, `REFRESH_TOKEN_TTL` |
| Storage | `STORAGE_ENDPOINT`, `STORAGE_BUCKET`, `STORAGE_ACCESS_KEY_ID`, `STORAGE_SECRET_ACCESS_KEY`, `STORAGE_REGION` |
| Rate limiting | `RATE_LIMIT_REQUESTS`, `RATE_LIMIT_WINDOW` |

The app refuses to start if a required variable is missing — check the error message for which one.

## API overview

Full spec: [`api/openapi.yaml`](api/openapi.yaml).

| Method | Path | Description |
|---|---|---|
| POST | `/auth/register` | Create an account |
| POST | `/auth/login` | Authenticate, receive tokens |
| POST | `/auth/refresh` | Exchange refresh token for new access token |
| POST | `/files` | Upload a file |
| GET | `/files` | List your files |
| GET | `/files/:id` | Download a file |
| DELETE | `/files/:id` | Delete a file |
| POST | `/files/:id/share` | Create a share link |
| DELETE | `/shares/:id` | Revoke a share link |
| GET | `/shared/:token` | Public access to a shared file |
| POST | `/uploads/session` | Start a trackable upload session |
| GET | `/uploads/:id/progress` | Poll upload progress |
| GET | `/health` | Health check |

## Development

```bash
make build      # build the binary
make run        # run locally
make test       # run unit tests
make lint       # run linter
make migrate    # apply Postgres migrations
```

## Documentation

- [`docs/architecture.md`](docs/architecture.md) — system design, data flow, store rationale
- [`docs/setup.md`](docs/setup.md) — local dev setup in detail
- [`docs/deployment.md`](docs/deployment.md) — deployment steps and infra requirements
- [`docs/api-guide.md`](docs/api-guide.md) — human-readable API usage guide
- [`docs/decisions/`](docs/decisions/) — architecture decision records (ADRs)

## License

See [`LICENSE`](LICENSE).
