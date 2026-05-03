---
applyTo: "**"
---

# General Instructions

All documentation, code comments, commit messages, and identifiers in this
repository MUST be written in English. Conversational replies to the user may
be in Japanese, but any artifact written to disk must be English-only.

---

## 1. Project Overview

`cmn-weave` is a distributed processing system composed of three components
that cooperate to execute work on behalf of a single operator.

```
+--------+   REST/HTTPS (mTLS)   +--------+   gRPC/HTTPS (mTLS)   +-------+
| client | ───────────────────▶ | server | ────────────────────▶ | agent |
+--------+                       +--------+                        +-------+
```

- **client**: CLI / front-end that submits requests to the server.
- **server**: REST API gateway. Accepts requests from the client, performs
  validation and orchestration, and dispatches work to one or more agents
  via gRPC.
- **agent**: gRPC service that executes the actual work on a target host
  and reports results back to the server.

---

## 2. Prerequisites

### 2.1 Operational assumptions

- **Single user**: The system is operated by exactly one human user.
  Therefore, user authentication / authorization (login, RBAC, sessions,
  JWT, OIDC user flows, etc.) is **out of scope**.
- **Trust boundary is the certificate**: Identity is established solely by
  X.509 client/server certificates issued from a private CA managed by the
  operator. Possession of a valid certificate == authorization to act.
- **Private deployment**: Components are deployed inside a network the
  operator controls. Public exposure of the server or agent endpoints is
  not a supported deployment mode.

### 2.2 Runtime / toolchain

- Language: Go (see `go.mod` for the required minor version).
- Transport:
  - client ↔ server: HTTP/1.1 or HTTP/2 REST over TLS (Gin).
  - server ↔ agent: gRPC over HTTP/2 with TLS.
- Storage: PostgreSQL and Redis are available but optional per component.
- Configuration: YAML files under `etc/` (see `*.yaml.example`).

### 2.3 Security prerequisites

- A private CA (root, and optionally intermediates) exists and is trusted
  by every component.
- Each component has its own key pair and certificate:
  - `server`: server cert with SAN matching its reachable address; also
    acts as a gRPC client toward agents (uses the same identity or a
    dedicated client cert).
  - `agent`: server cert for its gRPC listener.
  - `client`: client cert presented to the server.
- Certificate material is supplied via files referenced from configuration
  (path-based); secrets are never embedded in source or images.

---

## 3. Design Policy

### 3.1 Communication

1. **client → server** uses REST (JSON) over TLS with **mutual TLS**.
   - The server requires and verifies a client certificate on every
     request (`tls.RequireAndVerifyClientCert`).
   - There is no cookie/session/token layer; the certificate is the
     identity.
2. **server → agent** uses **gRPC with mutual TLS**.
   - The server is a gRPC client; the agent is a gRPC server.
   - Both sides verify the peer certificate against the shared CA bundle.
3. **agent → server** callbacks (if any) reuse the same mTLS trust store.
4. Plaintext (`h2c`, plain HTTP) is forbidden in non-test builds.

### 3.2 Certificate handling

- All TLS material is loaded at process start via the `crypto/tls`
  standard library. Hot reload is **not** required for the initial
  implementation.
- Configuration keys (suggested):
  ```
  tls:
    ca_file:   /path/to/ca.pem
    cert_file: /path/to/<component>.pem
    key_file:  /path/to/<component>-key.pem
    client_auth: require_and_verify   # server-side only
  ```
- A small shared helper inside each component's `share` package
  (e.g. `pkg/server/share`) builds `*tls.Config` for both REST and gRPC
  sides to avoid divergent settings. The `share` package is kept flat —
  no subdirectories.

### 3.3 Layering (per component)

The `client` and `server` components follow the existing repository
layout:

```
controller  -> HTTP/gRPC handlers, request/response mapping only
usecase     -> business logic, orchestration, transactions
repository  -> persistence and outbound calls (DB, Redis, gRPC client)
share       -> cross-cutting helpers (tls, logging, errors)
entity      -> domain models, request/response DTOs
```

- Controllers MUST NOT contain business logic.
- Usecases MUST NOT depend on transport types (`*gin.Context`, gRPC
  metadata, etc.).
- Repositories own all I/O; everything else is pure logic where possible.

The `agent` component uses a different layout because it hosts the gRPC
service surface and a native execution core:

```
pkg/agent/
  grpc/proto/  -> .proto source files (single source of truth for gRPC API)
  grpc/auto/   -> generated code only (Go gRPC stubs, C header from Rust)
  rust/        -> Rust crate: execution primitives + cgo bridge (staticlib)
  cplane/
    controller/ -> gRPC handler methods (common.go + task.go per resource)
    repository/ -> ExecutionRepository interface + OS/psql/rust impls
    proc/       -> process snapshot helpers (cgo + stub builds)
```

Rules:

- **`grpc/proto/`** holds hand-written `.proto` files. Service / message
  changes start here. Nothing else may live in this directory.
- **`grpc/auto/`** contains ONLY machine-generated artifacts:
  - Go: `*.pb.go`, `*_grpc.pb.go` produced by `protoc-gen-go` and
    `protoc-gen-go-grpc`.
  - C header produced by `cbindgen` from the Rust crate.
  - Files here are regenerated by `make` targets and MUST NOT be edited
    by hand. The directory is committed so consumers do not need the
    full toolchain to build.
- **`rust/`** is a Rust crate that implements the agent's execution
  primitives (CPU/IO heavy work, platform-specific calls, etc.) and is
  exposed to Go via a C ABI:
  - Built as a `staticlib` (preferred) or `cdylib`; the resulting
    artifact is linked into the agent binary through cgo.
  - Public functions are declared `extern "C"` and panic-free at the
    boundary; errors are returned via out-parameters or status codes,
    never by unwinding into Go.
  - All allocations crossing the FFI boundary have explicit ownership
    rules (allocator that frees == allocator that allocated).
  - Headers / Go bindings are emitted into `pkg/agent/grpc/auto/` by the
    build pipeline, keeping `rust/` free of generated files.

Build pipeline (high level):

1. `make proto` — runs `protoc` against `pkg/agent/grpc/proto/*.proto` and
   writes Go stubs into `pkg/agent/grpc/auto/`.
2. `make core` — runs `cargo build --release` in `pkg/agent/rust/`,
   produces the static library and a C header, and writes the Go
   binding shim into `pkg/agent/grpc/auto/`.
3. `make build` — depends on the two steps above, then builds the Go
   agent binary with cgo enabled and the static library linked in.

Out-of-tree consumers (server, tests) import the gRPC client stubs
from `pkg/agent/grpc/auto/...` only; they never touch `rust/` directly.

### 3.4 API style

- REST (client ↔ server):
  - JSON request/response, snake_case fields.
  - Versioned path prefix: `/api/v1/...`.
  - Errors use a uniform envelope (see `pkg/code` and
    `pkg/entity/response`).
- gRPC (server ↔ agent):
  - `.proto` files are the source of truth; generated code is committed.
  - Service definitions are task-oriented (e.g. `Dispatch`, `Status`,
    `Cancel`), not CRUD.
  - Streaming is preferred for long-running jobs and log forwarding.

### 3.5 Configuration & secrets

- Configuration is loaded from `etc/*.yaml` with environment variable
  overrides for deployment-specific values.
- Secrets (DB passwords, TLS keys) are referenced by path or pulled from
  the secrets manager described in `docs/SECRETS_MANAGER.md`. They are
  never logged.

### 3.6 Observability

- Structured logging via `pkg/global/logger.go`. No `fmt.Println`.
- Every inbound request is logged with: peer certificate CN, request id,
  route, status, latency. Certificate fingerprints may be logged; private
  keys and request bodies must not.
- Errors carry stable codes from `pkg/code/mcode.go` / `rcode.go`.

### 3.7 Testing

- Unit tests live under `test/unit/...` mirroring the source layout.
- E2E tests under `test/e2e/...` exercise the full
  client → server → agent path over real mTLS using ephemeral certs
  generated in test setup.
- No test may disable TLS verification (`InsecureSkipVerify: true`)
  outside of clearly scoped test helpers.

### 3.8 Out of scope (initial version)

- Multi-tenant authentication / user management.
- Authorization policies beyond "valid cert from our CA".
- Certificate rotation without process restart.
- Public-internet exposure hardening (rate limiting per user, CAPTCHA,
  etc.).

### 3.9 File-splitting convention

Within each layer package (`controller`, `usecase`, `repository`) files are
split by **resource**:

- `common.go` — shared struct / constructor / helpers that are not
  resource-specific (e.g. `Controller`, `New()`, output-format helpers,
  cross-resource interfaces).
- `<resource>.go` — types, interfaces, and methods that belong to exactly
  one resource (e.g. `task.go`, `node.go`, `ga.go`).

Non-resource helpers that are used across multiple resource files are
consolidated into `common.go` rather than placed in a separate utility file.
This keeps the number of files small and import paths stable.

### 3.10 Bootstrap (direct-DB) policy

The `bootstrap db` client command connects **directly to PostgreSQL** using a
DSN supplied in `etc/client.yaml` (`postgres_dsn`). It does **not** go
through the server REST API.

- ORM: **GORM** (`gorm.io/driver/postgres`) — no raw `pgx` in the client.
- Relevant types live in:
  - `pkg/client/repository/common.go` — `AdminRepository` interface +
    `GORMAdminRepository` implementation.
  - `pkg/client/usecase/common.go` — `AdminUsecase` interface +
    `AdminUsecaseImpl`.
  - `pkg/client/controller/common.go` — `NewBootstrapDBCmd` factory.
- The server has **no** bootstrap endpoint; adding one is out of scope.

---

## 4. Coding Conventions

- All Go code follows `gofmt` / `goimports`; run `make` targets before
  committing.
- Package comments, exported identifiers, and inline comments are written
  in **English only**.
- Prefer returning errors over panicking; wrap with `fmt.Errorf("...: %w",
  err)` to preserve the chain.
- Avoid global mutable state; pass dependencies through constructors.
- Keep handlers thin; put logic in `usecase`.
