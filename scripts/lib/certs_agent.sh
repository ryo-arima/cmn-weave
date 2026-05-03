#!/usr/bin/env bash
# certs_agent.sh — gRPC agent server certificate and agent-as-REST-client certificate.
# Sourced by certs.sh.  Requires AGENT_DIR/CA_KEY/CA_CRT (certs_util.sh) and info (common.sh).

# _certs_gen_agent
#   Generate etc/agent/agent.key + agent.crt (serverAuth, wildcard SAN).
#   Used by: agent gRPC listener (server → agent mTLS)
#   Reads: AGENT_CN (default: agent.cmn.local), CERT_DAYS, WILDCARD_SAN
function _certs_gen_agent() {
    local cn="${AGENT_CN:-agent.cmn.local}"
    local days="${CERT_DAYS:-3650}"
    local san="${WILDCARD_SAN:-DNS:*.cmn.local,DNS:localhost,IP:127.0.0.1}"

    info "  [agent] Generating RSA-4096 private key"
    openssl genrsa -out "${AGENT_DIR}/agent.key" 4096 2>/dev/null
    chmod 600 "${AGENT_DIR}/agent.key"

    info "  [agent] Signing server certificate (SAN=${san})"
    openssl req -new \
        -key  "${AGENT_DIR}/agent.key" \
        -out  "${AGENT_DIR}/agent.csr" \
        -subj "/CN=${cn}/O=cmn-weave" \
        2>/dev/null
    openssl x509 -req \
        -in      "${AGENT_DIR}/agent.csr" \
        -CA      "${CA_CRT}" \
        -CAkey   "${CA_KEY}" \
        -CAcreateserial \
        -out     "${AGENT_DIR}/agent.crt" \
        -days    "${days}" \
        -extfile <(printf "subjectAltName=%s\nextendedKeyUsage=serverAuth\n" "${san}") \
        2>/dev/null
}

# _certs_gen_agent_client
#   Generate etc/agent/agent-client.key + agent-client.crt (clientAuth).
#   Used by: agent REST client (agent → server mTLS)
#   Reads: AGENT_CLIENT_CN (default: agent-client.cmn.local), CERT_DAYS
function _certs_gen_agent_client() {
    local cn="${AGENT_CLIENT_CN:-agent-client.cmn.local}"
    local days="${CERT_DAYS:-3650}"

    info "  [agent-client] Generating RSA-4096 private key"
    openssl genrsa -out "${AGENT_DIR}/agent-client.key" 4096 2>/dev/null
    chmod 600 "${AGENT_DIR}/agent-client.key"

    info "  [agent-client] Signing client certificate (clientAuth)"
    openssl req -new \
        -key  "${AGENT_DIR}/agent-client.key" \
        -out  "${AGENT_DIR}/agent-client.csr" \
        -subj "/CN=${cn}/O=cmn-weave" \
        2>/dev/null
    openssl x509 -req \
        -in      "${AGENT_DIR}/agent-client.csr" \
        -CA      "${CA_CRT}" \
        -CAkey   "${CA_KEY}" \
        -CAcreateserial \
        -out     "${AGENT_DIR}/agent-client.crt" \
        -days    "${days}" \
        -extfile <(printf "extendedKeyUsage=clientAuth\n") \
        2>/dev/null
}
