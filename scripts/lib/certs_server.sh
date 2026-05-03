#!/usr/bin/env bash
# certs_server.sh — REST server certificate and server-as-gRPC-client certificate.
# Sourced by certs.sh.  Requires SERVER_DIR/CA_KEY/CA_CRT (certs_util.sh) and info (common.sh).

# _certs_gen_server
#   Generate etc/server/server.key + server.crt (serverAuth, wildcard SAN).
#   Used by: server REST listener (client → server mTLS)
#   Reads: SERVER_CN (default: server.cmn.local), CERT_DAYS, WILDCARD_SAN
function _certs_gen_server() {
    local cn="${SERVER_CN:-server.cmn.local}"
    local days="${CERT_DAYS:-3650}"
    local san="${WILDCARD_SAN:-DNS:*.cmn.local,DNS:localhost,IP:127.0.0.1}"

    info "  [server] Generating RSA-4096 private key"
    openssl genrsa -out "${SERVER_DIR}/server.key" 4096 2>/dev/null
    chmod 600 "${SERVER_DIR}/server.key"

    info "  [server] Signing server certificate (SAN=${san})"
    openssl req -new \
        -key  "${SERVER_DIR}/server.key" \
        -out  "${SERVER_DIR}/server.csr" \
        -subj "/CN=${cn}/O=cmn-weave" \
        2>/dev/null
    openssl x509 -req \
        -in      "${SERVER_DIR}/server.csr" \
        -CA      "${CA_CRT}" \
        -CAkey   "${CA_KEY}" \
        -CAcreateserial \
        -out     "${SERVER_DIR}/server.crt" \
        -days    "${days}" \
        -extfile <(printf "subjectAltName=%s\nextendedKeyUsage=serverAuth\n" "${san}") \
        2>/dev/null
}

# _certs_gen_server_client
#   Generate etc/server/server-client.key + server-client.crt (clientAuth).
#   Used by: server gRPC client (server → agent mTLS)
#   Reads: SERVER_CLIENT_CN (default: server-client.cmn.local), CERT_DAYS
function _certs_gen_server_client() {
    local cn="${SERVER_CLIENT_CN:-server-client.cmn.local}"
    local days="${CERT_DAYS:-3650}"

    info "  [server-client] Generating RSA-4096 private key"
    openssl genrsa -out "${SERVER_DIR}/server-client.key" 4096 2>/dev/null
    chmod 600 "${SERVER_DIR}/server-client.key"

    info "  [server-client] Signing client certificate (clientAuth)"
    openssl req -new \
        -key  "${SERVER_DIR}/server-client.key" \
        -out  "${SERVER_DIR}/server-client.csr" \
        -subj "/CN=${cn}/O=cmn-weave" \
        2>/dev/null
    openssl x509 -req \
        -in      "${SERVER_DIR}/server-client.csr" \
        -CA      "${CA_CRT}" \
        -CAkey   "${CA_KEY}" \
        -CAcreateserial \
        -out     "${SERVER_DIR}/server-client.crt" \
        -days    "${days}" \
        -extfile <(printf "extendedKeyUsage=clientAuth\n") \
        2>/dev/null
}
