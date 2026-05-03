#!/usr/bin/env bash
# certs_client.sh — CLI client certificate (CLI → server REST mTLS).
# Sourced by certs.sh.  Requires CLIENT_DIR/CA_KEY/CA_CRT (certs_util.sh) and info (common.sh).

# _certs_gen_client
#   Generate etc/client/client.key + client.crt (clientAuth).
#   Used by: CLI when connecting to the server REST API over mTLS.
#   Reads: CLIENT_CN (default: client.cmn.local), CERT_DAYS
function _certs_gen_client() {
    local cn="${CLIENT_CN:-client.cmn.local}"
    local days="${CERT_DAYS:-3650}"

    info "  [client] Generating RSA-4096 private key"
    openssl genrsa -out "${CLIENT_DIR}/client.key" 4096 2>/dev/null
    chmod 600 "${CLIENT_DIR}/client.key"

    info "  [client] Signing client certificate (clientAuth)"
    openssl req -new \
        -key  "${CLIENT_DIR}/client.key" \
        -out  "${CLIENT_DIR}/client.csr" \
        -subj "/CN=${cn}/O=cmn-weave" \
        2>/dev/null
    openssl x509 -req \
        -in      "${CLIENT_DIR}/client.csr" \
        -CA      "${CA_CRT}" \
        -CAkey   "${CA_KEY}" \
        -CAcreateserial \
        -out     "${CLIENT_DIR}/client.crt" \
        -days    "${days}" \
        -extfile <(printf "extendedKeyUsage=clientAuth\n") \
        2>/dev/null
}
