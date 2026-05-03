#!/usr/bin/env bash
# certs.sh — Public certificate management commands for cmn-weave.
# Sourced by main.sh; loads the role-specific sub-modules automatically.
#
# Public functions (called from main.sh):
#   certs_gen    — generate a full certificate set for all roles
#   certs_show   — print subject/issuer/dates for each certificate
#   certs_clean  — remove all generated files from etc/secrets/
#
# Role sub-modules:
#   certs_util.sh    — SECRETS_DIR constant + shared helpers
#   certs_root.sh    — CA key and self-signed certificate
#   certs_server.sh  — REST server cert + server-as-gRPC-client cert
#   certs_agent.sh   — gRPC agent server cert + agent-as-REST-client cert
#   certs_client.sh  — CLI client cert

source "${SCRIPT_DIR}/lib/certs_util.sh"
source "${SCRIPT_DIR}/lib/certs_root.sh"
source "${SCRIPT_DIR}/lib/certs_server.sh"
source "${SCRIPT_DIR}/lib/certs_agent.sh"
source "${SCRIPT_DIR}/lib/certs_client.sh"

# ---------------------------------------------------------------------------
# certs_gen
#   Orchestrate the full certificate generation pipeline.
#
#   Communication paths covered:
#     CLI     --[REST mTLS]--> server  (client cert)
#     server  --[gRPC mTLS]--> agent   (server-client cert + agent server cert)
#     agent   --[REST mTLS]--> server  (agent-client cert)
#
#   Step 1 — root:   CA key + self-signed CA certificate
#   Step 2 — server: REST server cert (serverAuth) + gRPC client cert (clientAuth)
#   Step 3 — agent:  gRPC server cert (serverAuth) + REST client cert (clientAuth)
#   Step 4 — client: CLI client cert (clientAuth)
#
#   Environment variables (all optional):
#     CERT_DAYS=3650
#     CA_CN=cmn-weave-ca
#     WILDCARD_SAN=DNS:*.cmn.local,DNS:localhost,IP:127.0.0.1
#     SERVER_CN=server.cmn.local
#     SERVER_CLIENT_CN=server-client.cmn.local
#     AGENT_CN=agent.cmn.local
#     AGENT_CLIENT_CN=agent-client.cmn.local
#     CLIENT_CN=client.cmn.local
# ---------------------------------------------------------------------------
function certs_gen() {
    _certs_require_openssl
    _certs_ensure_dir

    info "Step 1/4: root"
    _certs_gen_root

    info "Step 2/4: server"
    _certs_gen_server
    _certs_gen_server_client

    info "Step 3/4: agent"
    _certs_gen_agent
    _certs_gen_agent_client

    info "Step 4/4: client (CLI)"
    _certs_gen_client

    _certs_cleanup_tmp
    _certs_list_generated
}

# ---------------------------------------------------------------------------
# certs_show
#   Display the subject, issuer, and validity window for each certificate.
# ---------------------------------------------------------------------------
function certs_show() {
    local found=0
    info "--- root.crt (etc/common) ---"
    if [[ -f "${CA_CRT}" ]]; then
        openssl x509 -in "${CA_CRT}" -noout -subject -issuer -dates
        found=$((found + 1))
    else
        warn "root.crt not found (run: make certs-gen)"
    fi
    local -A role_dirs=(
        [server]="${SERVER_DIR}"
        [server-client]="${SERVER_DIR}"
        [agent]="${AGENT_DIR}"
        [agent-client]="${AGENT_DIR}"
        [client]="${CLIENT_DIR}"
    )
    for role in server server-client agent agent-client client; do
        local dir="${role_dirs[${role}]}"
        local path="${dir}/${role}.crt"
        if [[ -f "${path}" ]]; then
            info "--- ${role}.crt ---"
            openssl x509 -in "${path}" -noout -subject -issuer -dates
            found=$((found + 1))
        else
            warn "${role}.crt not found (run: make certs-gen)"
        fi
    done
    [[ ${found} -eq 0 ]] && err "No certificates found. Run 'certs gen' first."
}

# ---------------------------------------------------------------------------
# certs_clean
#   Remove every certificate, key, CSR, and serial file from SECRETS_DIR.
# ---------------------------------------------------------------------------
function certs_clean() {
    warn "Removing certificate/key files from etc/common, etc/server, etc/agent, etc/client"
    rm -f "${COMMON_DIR}"/*.crt "${COMMON_DIR}"/*.key \
          "${SERVER_DIR}"/*.crt "${SERVER_DIR}"/*.key \
          "${AGENT_DIR}"/*.crt  "${AGENT_DIR}"/*.key \
          "${CLIENT_DIR}"/*.crt "${CLIENT_DIR}"/*.key 2>/dev/null || true
    success "Certificate files removed"
}
