# ---- Go build stage ----
FROM golang:1.22-bookworm AS go-builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

# ---- Runtime stage ----
FROM ubuntu:24.04
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=go-builder /out/server ./server
EXPOSE 8443
ENTRYPOINT ["./server", "--config", "/etc/cmn-weave/server.yaml"]
