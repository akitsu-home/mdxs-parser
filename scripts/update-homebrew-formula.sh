#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${HOMEBREW_TAP_TOKEN:-}" || -z "${HOMEBREW_TAP_REPO:-}" ]]; then
  echo "HOMEBREW_TAP_TOKEN and HOMEBREW_TAP_REPO are required" >&2
  exit 1
fi

if [[ -z "${APP_NAME:-}" || -z "${VERSION:-}" ]]; then
  echo "APP_NAME and VERSION are required" >&2
  exit 1
fi

echo "TODO: Implement Homebrew tap update for ${HOMEBREW_TAP_REPO}"
echo "Use packaging/homebrew/${APP_NAME}.rb.tmpl and release checksums to render formula."
