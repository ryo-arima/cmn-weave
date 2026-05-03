# cmn-weave

> **⚠ Work in progress — not functional yet.**
> The codebase is under active development. APIs, config schemas, and build
> steps will change without notice. Do not use in production.

---

Distributed processing system composed of three components communicating
exclusively over mutually-authenticated TLS:

```
+--------+   REST/HTTPS (mTLS)   +--------+   gRPC/HTTPS (mTLS)   +-------+
| client | ───────────────────▶ | server | ────────────────────▶ | agent |
+--------+                       +--------+                        +-------+
```

- **client** — CLI that submits work to the server.
- **server** — REST API gateway; validates the request and dispatches to one
  or more agents via gRPC.
- **agent** — gRPC service that executes the work, with a Rust execution
  core linked in via cgo.

## Trust model

cmn-weave is operated by a single user, so there is no login, RBAC, or token
layer. Identity is established **only** by X.509 certificates issued from a
private CA the operator controls; possession of a valid certificate equals
authorisation to act. All listeners require and verify the peer certificate
(`tls.RequireAndVerifyClientCert`).

## Repository layout

```
cmd/
  server/   REST API binary
  agent/    gRPC binary (links Rust core via cgo)
  client/   CLI binary
pkg/
  config/   Configuration loader and unified YAML schema
  global/   Process-wide singletons (logger)
  server/   REST router, controllers, mTLS middleware, tlsutil helper
  agent/
    grpc/proto/  .proto sources (single source of truth for gRPC API)
    grpc/auto/   Generated Go stubs and Rust FFI bindings (do not edit)
    rust/        Rust crate built as a staticlib and linked via cgo
etc/
  server/   Server config example and TLS material
  agent/    Agent config example and TLS material
  client/   Client config example and TLS material
  common/   Shared CA certificate
docker/     Container build helpers and runtime configs
```

## Configuration schema

All three components share a single YAML schema (`pkg/config.YamlConfig`).
Each component reads only the sections it needs:

```yaml
application:
  server:
    listen_addr: ":8443"
    tls:
      ca_file:   etc/common/root.crt
      cert_file: etc/server/server.crt
      key_file:  etc/server/server.key
    agents:
      - agent-1.cmn.local:9443
  agent:
    listen_addr: ":9443"
    tls:
      ca_file:   etc/common/root.crt
      cert_file: etc/agent/agent.crt
      key_file:  etc/agent/agent.key
  client:
    server_endpoint: "https://server.cmn.local:8443"
    tls:
      ca_file:   etc/common/root.crt
      cert_file: etc/client/client.crt
      key_file:  etc/client/client.key

postgresql:
  host:    localhost
  port:    5432
  user:    cmnweave
  pass:    secret
  dbname:  cmnweave
  sslmode: require

redis:
  host: localhost
  port: 6379
  db:   0
```

See the `*.yaml.example` files in each `etc/` subdirectory for full examples.

## Build

```sh
# Prerequisites: Go, Rust (cargo), protoc + protoc-gen-go + protoc-gen-go-grpc

# Regenerate gRPC stubs
make proto

# Build Rust core staticlib
make core

# Build all Go binaries
make build
```

See [.github/instructions/general.instructions.md](.github/instructions/general.instructions.md)
for the full design policy and coding conventions.

