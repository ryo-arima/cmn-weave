#!/usr/bin/env bash
# Unified CLI for cmn-weave management
# Usage: ./scripts/main.sh <command> [args...]
# Example: ./scripts/main.sh certs gen
#          ./scripts/main.sh certs show
#          ./scripts/main.sh certs clean
set -euo pipefail

COMMAND="${1:-help}"
shift || true

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${ROOT_DIR}"

# Load library functions
source "${SCRIPT_DIR}/lib/common.sh"
source "${SCRIPT_DIR}/lib/certs.sh"
source "${SCRIPT_DIR}/lib/help.sh"

# ============================================================================
# Command Router
# ============================================================================
case "${COMMAND}" in
    certs)
        SUB="${1:-help}"
        shift || true
        case "${SUB}" in
            gen)   certs_gen   ;;
            show)  certs_show  ;;
            clean) certs_clean ;;
            *)     err "Unknown certs sub-command: ${SUB}"; help_show; exit 1 ;;
        esac
        ;;
    help|--help|-h)
        help_show
        ;;
    *)
        err "Unknown command: ${COMMAND}"
        help_show
        exit 1
        ;;
esac
