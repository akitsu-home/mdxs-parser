#!/usr/bin/env bash
set -euo pipefail

OWNER="${OWNER:-akitsu-home}"
REPO="${REPO:-mdxs-parser}"
APP_NAME="${APP_NAME:-mdxs-parser}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-}"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command not found: $1" >&2
    exit 1
  fi
}

need_cmd uname
need_cmd mktemp
need_cmd tar
need_cmd install

if command -v curl >/dev/null 2>&1; then
  FETCHER="curl"
elif command -v wget >/dev/null 2>&1; then
  FETCHER="wget"
else
  echo "error: curl or wget is required" >&2
  exit 1
fi

detect_os() {
  case "$(uname -s)" in
    Linux)
      echo "linux"
      ;;
    Darwin)
      echo "darwin"
      ;;
    *)
      echo "error: unsupported OS: $(uname -s)" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)
      echo "amd64"
      ;;
    arm64|aarch64)
      echo "arm64"
      ;;
    *)
      echo "error: unsupported arch: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

http_get() {
  local url="$1"
  local output="$2"

  if [[ "$FETCHER" == "curl" ]]; then
    curl -fsSL "$url" -o "$output"
  else
    wget -qO "$output" "$url"
  fi
}

resolve_version() {
  if [[ -n "$VERSION" ]]; then
    echo "$VERSION"
    return
  fi

  local latest_url="https://github.com/${OWNER}/${REPO}/releases/latest"
  local effective_url

  if [[ "$FETCHER" == "curl" ]]; then
    effective_url="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "$latest_url")"
  else
    effective_url="$(wget -qO /dev/null --max-redirect=20 --server-response "$latest_url" 2>&1 | awk '/^  Location: / {print $2}' | tr -d '\r' | tail -n 1)"
  fi

  local tag
  tag="$(echo "$effective_url" | sed -E 's#.*/tag/([^/?#]+).*#\1#')"

  if [[ -z "$tag" || "$tag" == "$effective_url" ]]; then
    echo "error: failed to resolve latest release tag" >&2
    exit 1
  fi

  echo "$tag"
}

OS="$(detect_os)"
ARCH="$(detect_arch)"
VERSION="$(resolve_version)"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

version_no_v="${VERSION#v}"
archive_path="${tmp_dir}/artifact.tar.gz"
src_bin=""

for candidate_version in "$version_no_v" "$VERSION"; do
  ASSET="${APP_NAME}_${candidate_version}_${OS}_${ARCH}.tar.gz"
  URL="https://github.com/${OWNER}/${REPO}/releases/download/${VERSION}/${ASSET}"

  echo "Downloading ${URL}"
  if http_get "$URL" "$archive_path"; then
    tar -xzf "$archive_path" -C "$tmp_dir"

    candidate_bin="${tmp_dir}/${APP_NAME}_${candidate_version}_${OS}_${ARCH}/${APP_NAME}"
    if [[ ! -f "$candidate_bin" ]]; then
      candidate_bin="${tmp_dir}/${APP_NAME}"
    fi
    if [[ -f "$candidate_bin" ]]; then
      src_bin="$candidate_bin"
      break
    fi
  fi
done

if [[ -z "$src_bin" ]]; then
  echo "error: failed to download a matching release asset for ${VERSION} (${OS}/${ARCH})" >&2
  exit 1
fi

dest_bin="${INSTALL_DIR}/${APP_NAME}"

if [[ -w "$INSTALL_DIR" ]]; then
  install -m 0755 "$src_bin" "$dest_bin"
else
  if command -v sudo >/dev/null 2>&1; then
    sudo install -m 0755 "$src_bin" "$dest_bin"
  else
    echo "error: ${INSTALL_DIR} is not writable and sudo is not available" >&2
    exit 1
  fi
fi

echo "Installed ${APP_NAME} to ${dest_bin}"
echo "Run: ${APP_NAME} version"