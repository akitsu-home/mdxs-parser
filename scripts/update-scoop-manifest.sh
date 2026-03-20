#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${SCOOP_BUCKET_TOKEN:-}" || -z "${SCOOP_BUCKET_REPO:-}" ]]; then
  echo "SCOOP_BUCKET_TOKEN and SCOOP_BUCKET_REPO are required" >&2
  exit 1
fi

if [[ -z "${APP_NAME:-}" || -z "${VERSION:-}" ]]; then
  echo "APP_NAME and VERSION are required" >&2
  exit 1
fi

echo "TODO: Implement Scoop bucket update for ${SCOOP_BUCKET_REPO}"
echo "Use packaging/scoop/${APP_NAME}.json.tmpl and release checksums to render manifest."
