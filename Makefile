.PHONY: build proto core test clean tidy \
        dev-up dev-down \
        svr-up svr-down \
        agt-up agt-down

BIN_DIR  := .bin

# Docker Compose files
# dev-up starts all infrastructure except server and agent.
DEV_COMPOSE := \
	-f docker/network.yaml \
	-f docker/postgres.yaml \
	-f docker/redis.yaml \
	-f docker/dns.yaml \
	-f docker/pgadmin.yaml \
	-f docker/swagger.yaml \
	-f docker/godoc.yaml

SVR_COMPOSE := \
	-f docker/network.yaml \
	-f docker/server.yaml

AGT_COMPOSE := \
	-f docker/network.yaml \
	-f docker/agent.yaml

# Build the three component binaries.
build: proto core
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/server ./cmd/server
	go build -o $(BIN_DIR)/agent  ./cmd/agent
	go build -o $(BIN_DIR)/client ./cmd/client

# Generate Go gRPC stubs from pkg/agent/grpc/proto into pkg/agent/grpc/auto.
# Requires: protoc, protoc-gen-go, protoc-gen-go-grpc.
proto:
	@command -v protoc >/dev/null || { echo "protoc not installed; skipping" ; exit 0 ; }
	protoc \
		--proto_path=pkg/agent/grpc/proto \
		--go_out=pkg/agent/grpc/auto --go_opt=paths=source_relative \
		--go-grpc_out=pkg/agent/grpc/auto --go-grpc_opt=paths=source_relative \
		pkg/agent/grpc/proto/*.proto

# Build the Rust core static library.
# Generated headers / Go bindings land under pkg/agent/grpc/auto via the build
# pipeline once cgo glue is added.
core:
	@command -v cargo >/dev/null || { echo "cargo not installed; skipping" ; exit 0 ; }
	cd pkg/agent/rust && cargo build --release

tidy:
	go mod tidy

test:
	go test ./...

clean:
	rm -rf $(BIN_DIR)
	cd pkg/agent/rust && cargo clean 2>/dev/null || true

# ---------------------------------------------------------------------------
# Docker Compose helpers
# ---------------------------------------------------------------------------

## dev-up: start all infrastructure containers (postgres, redis, dns,
##         pgadmin, swagger, godoc) — excludes server and agent.
dev-up:
	docker compose $(DEV_COMPOSE) up -d

## dev-down: stop and remove all dev infrastructure containers and resources.
dev-down:
	docker compose $(DEV_COMPOSE) down -v --remove-orphans

## svr-up / svr-down: server container only.
svr-up:
	docker compose $(SVR_COMPOSE) up -d

svr-down:
	docker compose $(SVR_COMPOSE) down -v

## agt-up / agt-down: agent container only.
agt-up:
	docker compose $(AGT_COMPOSE) up -d

agt-down:
	docker compose $(AGT_COMPOSE) down -v

# ---------------------------------------------------------------------------
# Certificate management
# ---------------------------------------------------------------------------

## certs-gen: generate CA, server, and client certificates into etc/server/, etc/agent/, etc/client/.
##   Override defaults with env vars:
##     CERT_DAYS=3650  CA_CN=cmn-weave-ca  WILDCARD_SAN=DNS:*.cmn.local,DNS:localhost,IP:127.0.0.1
certs-gen:
	./scripts/main.sh certs gen

## certs-show: print subject/issuer/dates for each certificate in etc/server/, etc/agent/, etc/client/.
certs-show:
	./scripts/main.sh certs show

## certs-clean: remove all generated cert/key files from etc/server/, etc/agent/, etc/client/ and etc/root.*.
certs-clean:
	./scripts/main.sh certs clean
