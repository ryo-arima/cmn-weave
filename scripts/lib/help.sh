#!/usr/bin/env bash
# Help text for cmn-weave scripts

function help_show() {
    cat <<'EOF'
cmn-weave scripts — unified CLI

Usage:
  ./scripts/main.sh <command> [args...]

Commands:
  certs gen     Generate the full certificate set (CA + 2 server certs + 3 client certs)

                Communication paths:
                  CLI             --[REST mTLS]--> server  (client.crt)
                  server          --[gRPC mTLS]--> agent   (server-client.crt + agent.crt)
                  agent           --[REST mTLS]--> server  (agent-client.crt)

                Generated files:
                  etc/common/root.key                  CA signing key (not deployed)
                  etc/common/root.crt                  CA cert (trust anchor)
                  etc/server/server.crt / .key         REST server cert    (serverAuth)
                  etc/server/server-client.crt / .key  gRPC client cert    (clientAuth)
                  etc/agent/agent.crt / .key           gRPC server cert    (serverAuth)
                  etc/agent/agent-client.crt / .key    REST client cert    (clientAuth)
                  etc/client/client.crt / .key         CLI client cert     (clientAuth)

                Environment variables (all optional):
                  CERT_DAYS=3650
                  CA_CN=cmn-weave-ca
                  WILDCARD_SAN=DNS:*.cmn.local,DNS:localhost,IP:127.0.0.1
                  SERVER_CN=server.cmn.local
                  AGENT_CN=agent.cmn.local
                  CLIENT_CN=client.cmn.local
                  AGENT_CLIENT_CN=agent-client.cmn.local
                  SERVER_CLIENT_CN=server-client.cmn.local

  certs show    Print subject and validity for each certificate in etc/server/, etc/agent/, etc/client/

  certs clean   Remove all generated cert/key files from etc/server/, etc/agent/, etc/client/ and etc/root.*

  help          Show this message

Examples:
  ./scripts/main.sh certs gen
  SERVER_CN=cmn-server.internal ./scripts/main.sh certs gen
  ./scripts/main.sh certs show
  ./scripts/main.sh certs clean
EOF
}
