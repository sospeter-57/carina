# Carina — Architecture

## 1. Overview

Carina is a file storage and sharing service: users authenticate, upload files, and generate share links others can use to access them without an account. It's a REST API backed by Postgres (durable metadata), Redis (ephemeral state), and S3-compatible object storage (file bytes).

The design goal isn't novelty — it's a well-understood problem (Dropbox-lite) built correctly: clean separation of concerns, explicit data ownership per store, and a structure that scales in complexity without a rewrite.

## 2. Goals and Non-Goals

**Goals:**
- Clear separation between identity (who can do what) and content (the files themselves)
- Stateless request auth, with revocation handled deliberately, not accidentally
- Object storage from day one — no assumption that files live on local disk
- A structure two people can work in simultaneously without constant merge conflicts

**Non-goals (for now):**
- Multi-region / multi-tenant architecture
- Client-side encryption of file contents
- Real-time collaborative editing of shared files
- Worker infrastructure beyond a simple in-process pool (no Kafka/NATS yet)

## 3. High-Level Architecture

```
                         ┌─────────────┐
                         │   Client    │
                         └──────┬──────┘
                                │ HTTPS (REST)
                                ▼
                       ┌────────────────┐
                       │  cmd/api        │
                       │  (Go binary)    │
                       └───────┬─────────┘
                               │
        ┌──────────────────────┼──────────────────────┐
        ▼                      ▼                       ▼
 ┌─────────────┐      ┌────────────────┐      ┌──────────────────┐
 │  identity/   │      │   content/      │      │  cross-cutting    │
 │  auth        │      │   file          │      │  ratelimit        │
 │  sharing     │      │   upload        │      │  middleware       │
 │              │      │   worker        │      │                    │
 └──────┬───────┘      └───────┬─────────┘      └──────────┬────────┘
        │                      │                            │
        ▼                      ▼                            ▼
 ┌─────────────┐      ┌─────────────────┐          ┌──────────────┐
 │  Postgres    │      │  Postgres        │          │   Redis       │
 │  (users,     │      │  (file/share     │          │  (upload      │
 │  shares)     │      │  metadata)       │          │  progress,    │
 └─────────────┘      └────────┬─────────┘          │  rate limits) │
                                │                     └──────────────┘
                                ▼
                       ┌─────────────────┐
                       │  S3 / MinIO      │
                       │  (file bytes)    │
                       └─────────────────┘
```

Every feature package talks to the platform layer (`internal/platform/postgres`, `internal/platform/redis`) through a repository interface it defines itself — nothing reaches into another feature's tables or another feature's Redis keys directly.

## 4. Directory Structure and Rationale

Carina uses a **feature-based (vertical slice)** layout, with features grouped into two domains:

```
internal/
├── config/          # single source of truth for env-based settings
├── server/           # HTTP server lifecycle, router
├── identity/         # who is allowed to do what
│   ├── auth/          # authentication: register, login, JWT
│   └── sharing/       # authorization on shared resources: share links
├── content/          # the file lifecycle, end to end
│   ├── file/          # metadata + storage for uploaded files
│   ├── upload/        # in-progress upload session state (Redis-backed)
│   └── worker/        # background processing: thumbnails, malware scan
├── ratelimit/        # cross-cutting: Redis-backed request limiting
├── middleware/       # cross-cutting: auth check, logging, panic recovery
└── platform/         # infra clients only — no business logic
    ├── postgres/
    └── redis/
```

**Why feature-based, not layer-based:** a layer-based split (`handlers/`, `services/`, `repositories/`) scatters one feature's logic across three folders. Vertical slices keep everything for "upload" in one place — easier to reason about, easier to hand off a whole feature to one contributor.

**Why `identity` and `content` specifically:** `auth` and `sharing` both answer "who is allowed to do what" — one establishes identity, the other authorizes access to a specific resource. `file`, `upload`, and `worker` all track one thing — a file's journey from bytes-in to processed-and-stored — even though they lean on different stores (`upload` is Redis-heavy, `file`/`worker` are Postgres+storage-heavy).

**Why `platform/` holds no logic:** it exists purely to construct connections (pools, clients) and expose health checks. Feature packages define their own repository interfaces and implement them against `platform` clients — this means swapping Postgres drivers or Redis clients later doesn't ripple into business logic.

**Why `ratelimit/` and `middleware/` sit outside both domains:** they apply across every feature, not to one. Folding them into `identity` or `content` would create an artificial ownership that doesn't reflect how they're actually used.

## 5. Configuration

`internal/config` is the only package that reads environment variables directly (`os.Getenv`). Everything else receives a typed struct.

- One struct per concern: `ServerConfig`, `PostgresConfig`, `RedisConfig`, `AuthConfig`, `StorageConfig`, `RateLimitConfig`
- A single `Load()` function parses env vars into correctly typed fields (`time.Duration`, `int`, `bool` — never left as raw strings) and validates required fields are present, failing fast at startup rather than deep into a request
- `main.go` calls `Load()` once, then hands each sub-struct only to the part of the app that needs it (`postgres.New(cfg.Postgres)`, not the whole `Config`)

`WorkerConfig` was deliberately deferred — added back once background workers are actually implemented, rather than carrying config for code that doesn't exist yet.

## 6. Data Stores and Their Roles

| Store | Holds | Why |
|---|---|---|
| **Postgres** | Users, file metadata, share links | Durable, relational, source of truth. Nothing here is allowed to disappear on restart. |
| **Redis** | Upload progress, rate-limit counters | Ephemeral by design — TTL-bound, safe to lose, exists only to coordinate short-lived state. |
| **S3 / MinIO** | Actual file bytes | Durable, replicated, independent of any single app instance — required the moment the app runs as more than one process or gets redeployed. |

**Why not just Postgres for everything:** rate-limit counters and upload-progress tracking are high-write, short-lived, and don't need relational guarantees or durability — Redis's atomic `INCR`/`HINCRBY` and native TTL support fit that shape directly, and keeping this traffic off Postgres avoids adding write pressure to the store that actually needs to be reliable.

**Why not local disk for files:** covered in depth in `docs/decisions/` — the short version is that ephemeral container filesystems, horizontal scaling, and durability all break the "just write to disk" approach as soon as this leaves a single laptop process.

## 7. Authentication and Authorization

- **Access tokens** — JWT, short TTL (15 min target), stateless verification (signature check only, no DB lookup per request)
- **Refresh tokens** — longer TTL (7 day target), tracked server-side (hashed, in Postgres) so they can be revoked on logout or password change — this is where Carina's "statefulness" in auth actually lives, since access tokens can't be revoked early by design
- **Auth middleware** (`internal/middleware/auth.go`) validates the bearer token and injects the user ID into request context; handlers never parse tokens themselves
- **Sharing** is a separate authorization layer on top of authentication — a share link (`sharing` package) grants scoped, tokenized, expiring access to a specific file without requiring the requester to have an account at all

## 8. Rate Limiting

Redis-backed, using the `INCR` + `EXPIRE` pattern keyed by user ID or IP. Applied via middleware to auth and upload endpoints specifically, since those are the most abuse-prone (credential stuffing, upload flooding). Returns `429` with `Retry-After` on limit breach.

## 9. Background Workers

Two workers run against completed uploads:
- **Thumbnail generation** — image files only, async, doesn't block the upload response
- **Malware scan** — all files pass through a `pending` → `available`/`rejected` status gate; downloads are blocked until a file clears scanning

Both are coordinated through a shared worker pool/dispatcher (`content/worker`) rather than ad-hoc goroutines, so shutdown and retry behavior is consistent across job types. Queue mechanism (Redis Streams vs. Go channels) is an open decision to be recorded as an ADR once workers are actually built.

## 10. Timeouts and Connection Limits

Rough defaults for a single-instance deployment, tuned for an I/O-bound service rather than assuming CPU-bound thread-pool sizing:

**Server (`ServerConfig`):**
- `ReadTimeout` / `WriteTimeout`: sized differently for JSON routes (short, ~5–10s) vs. file upload/download routes (long, minutes — scaled to max file size), since a single blanket timeout would either kill large uploads or leave slow-client JSON requests hanging too long
- `IdleTimeout`: matched to whatever's in front of the app (load balancer/proxy idle timeout), commonly 60s
- `ShutdownTimeout`: ~30s grace window for in-flight requests to finish before a deploy force-closes them

**Postgres pool (`pgxpool`):**
- `MaxConns`: 10–25 per instance (not per core — this is I/O-bound, not CPU-bound)
- `MinConns`: 2–5
- `MaxConnLifetime` / `MaxConnIdleTime`: periodic recycling so the pool doesn't hold dead connections after a failover, and shrinks back down during quiet periods

These are starting points, not fixed — raise them only in response to observed latency under real load, not preemptively.

## 11. Deployment Notes

- TLS termination point (at the app itself vs. upstream load balancer/proxy) is an open decision — if terminated upstream, `ServerConfig`'s TLS fields go unused in production and the app only ever speaks plain HTTP internally
- `StorageConfig.Endpoint` is meaningful for MinIO (local/self-hosted); against real AWS S3 it's typically left empty and resolved from `Region` by the AWS SDK instead
- Local dev environment (Postgres, Redis, MinIO, app) is fully described in `deployments/docker-compose.yml` — no manual infra setup should be required to run Carina locally

## 12. Open Questions / Future ADRs

- Queue mechanism for background workers (Redis Streams vs. in-process channel pool)
- Whether Carina terminates TLS itself or always sits behind a proxy that does
- Resumable/chunked upload protocol (tus or custom) once large-file reliability becomes a priority
- Whether `Endpoint`-based storage config stays a MinIO-only concern or becomes a shared override path for both MinIO and AWS