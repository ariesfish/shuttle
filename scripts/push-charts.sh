#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CHARTS_DIR="${CHARTS_DIR:-${REPO_ROOT}/charts}"
PACKAGE_DIR="${PACKAGE_DIR:-${CHARTS_DIR}/packages}"
REMOTE_REPO="${REMOTE_REPO:-}"

usage() {
  cat <<'EOF'
Usage:
  REMOTE_REPO=<repo> ./scripts/push-charts.sh
  ./scripts/push-charts.sh <repo>

Examples:
  ./scripts/push-charts.sh oci://registry.example.com/helm
  REMOTE_REPO=oci://ghcr.io/acme/charts ./scripts/push-charts.sh

Optional env:
  CHARTS_DIR=./charts         Directory containing chart directories.
  PACKAGE_DIR=./charts/packages
                             Directory containing packaged *.tgz charts.
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
    echo "Run ${SCRIPT_DIR}/package-charts.sh first." >&2
    exit 1
  fi

  mapfile -t packages < <(find_packages)
  if [[ "${#packages[@]}" -eq 0 ]]; then
    echo "No chart packages found in ${PACKAGE_DIR}" >&2
    echo "Run ${SCRIPT_DIR}/package-charts.sh first." >&2
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
