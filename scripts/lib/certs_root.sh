#!/usr/bin/env bash
# certs_root.sh — CA private key and self-signed CA certificate.
# Sourced by certs.sh.  Requires COMMON_DIR/CA_KEY/CA_CRT (certs_util.sh) and info (common.sh).

# _certs_gen_root
#   Generate etc/common/root.key (signing key, not deployed) and etc/common/root.crt.
#   All components reference etc/common/root.crt as the CA trust anchor.
#   Reads: CA_CN (default: cmn-weave-ca), CERT_DAYS (default: 3650)
function _certs_gen_root() {
    local cn="${CA_CN:-cmn-weave-ca}"
    local days="${CERT_DAYS:-3650}"

    info "  [root] Generating RSA-4096 private key -> etc/common/root.key"
    openssl genrsa -out "${CA_KEY}" 4096 2>/dev/null
    chmod 600 "${CA_KEY}"

    info "  [root] Creating self-signed CA certificate (CN=${cn}, days=${days}) -> etc/common/root.crt"
    openssl req -new -x509 \
        -key  "${CA_KEY}" \
        -out  "${CA_CRT}" \
        -days "${days}" \
        -subj "/CN=${cn}/O=cmn-weave" \
        2>/dev/null
}
