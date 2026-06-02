#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_DIR="${PACKAGE_DIR:-${SCRIPT_DIR}/packages}"
REMOTE_REPO="${REMOTE_REPO:-}"

usage() {
  cat <<'EOF'
Usage:
  REMOTE_REPO=<repo> ./push.sh
  ./push.sh <repo>

Examples:
  ./push.sh oci://registry.example.com/helm
  REMOTE_REPO=oci://ghcr.io/acme/charts ./push.sh

Optional env:
  PACKAGE_DIR=./packages      Directory containing packaged *.tgz charts.
  PACKAGES="a.tgz b.tgz"      Push only selected package files.
EOF
}

find_packages() {
  if [[ -n "${PACKAGES:-}" ]]; then
    for package in ${PACKAGES}; do
      if [[ "${package}" = /* ]]; then
        printf '%s\n' "${package}"
      else
        printf '%s\n' "${PACKAGE_DIR}/${package}"
      fi
    done
    return
  fi

  find "${PACKAGE_DIR}" -maxdepth 1 -type f -name '*.tgz' -print | sort
}

main() {
  if [[ -n "${1:-}" ]]; then
    REMOTE_REPO="$1"
  fi

  if [[ -z "${REMOTE_REPO}" ]]; then
    usage >&2
    exit 1
  fi

  if [[ ! -d "${PACKAGE_DIR}" ]]; then
    echo "Package directory does not exist: ${PACKAGE_DIR}" >&2
    echo "Run ${SCRIPT_DIR}/package.sh first." >&2
    exit 1
  fi

  mapfile -t packages < <(find_packages)
  if [[ "${#packages[@]}" -eq 0 ]]; then
    echo "No chart packages found in ${PACKAGE_DIR}" >&2
    echo "Run ${SCRIPT_DIR}/package.sh first." >&2
    exit 1
  fi

  for package in "${packages[@]}"; do
    if [[ ! -f "${package}" ]]; then
      echo "Package not found: ${package}" >&2
      exit 1
    fi

    echo "==> Pushing $(basename "${package}") to ${REMOTE_REPO}"
    helm push "${package}" "${REMOTE_REPO}"
  done
}

main "$@"
