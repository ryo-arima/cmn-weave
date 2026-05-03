#!/usr/bin/env bash
# certs_util.sh — Shared constants and helper functions for certs_*.sh modules.
# Sourced automatically by certs.sh; do not source directly.

# Shared CA files (referenced by all components as trust anchor).
COMMON_DIR="${ROOT_DIR}/etc/common"
CA_KEY="${COMMON_DIR}/root.key"
CA_CRT="${COMMON_DIR}/root.crt"

# Per-role output directories (match existing etc/ layout).
SERVER_DIR="${ROOT_DIR}/etc/server" # server.crt/key, server-client.crt/key
AGENT_DIR="${ROOT_DIR}/etc/agent"   # agent.crt/key, agent-client.crt/key
CLIENT_DIR="${ROOT_DIR}/etc/client" # client.crt/key

function _certs_require_openssl() {
    command -v openssl >/dev/null 2>&1 && return
    err "openssl not found; please install it and retry"
    exit 1
}

function _certs_ensure_dir() {
    mkdir -p "${COMMON_DIR}" "${SERVER_DIR}" "${AGENT_DIR}" "${CLIENT_DIR}"
    info "Output directories: etc/common  etc/server  etc/agent  etc/client"
}

function _certs_cleanup_tmp() {
    for dir in "${COMMON_DIR}" "${SERVER_DIR}" "${AGENT_DIR}" "${CLIENT_DIR}"; do
        rm -f "${dir}"/*.csr "${dir}"/*.srl 2>/dev/null || true
    done
}

function _certs_list_generated() {
    success "Certificates generated:"
    for f in root.key root.crt; do
        [[ -f "${COMMON_DIR}/${f}" ]] && info "  etc/common/${f}"
    done
    for f in server.crt server.key server-client.crt server-client.key; do
        [[ -f "${SERVER_DIR}/${f}" ]] && info "  etc/server/${f}"
    done
    for f in agent.crt agent.key agent-client.crt agent-client.key; do
        [[ -f "${AGENT_DIR}/${f}" ]] && info "  etc/agent/${f}"
    done
    for f in client.crt client.key; do
        [[ -f "${CLIENT_DIR}/${f}" ]] && info "  etc/client/${f}"
    done
}
