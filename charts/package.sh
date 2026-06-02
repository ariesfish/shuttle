#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_DIR="${PACKAGE_DIR:-${SCRIPT_DIR}/packages}"
SKIP_LINT="${SKIP_LINT:-false}"

find_charts() {
  if [[ -n "${CHARTS:-}" ]]; then
    for chart in ${CHARTS}; do
      if [[ "${chart}" = /* ]]; then
        printf '%s\n' "${chart}"
      else
        printf '%s\n' "${SCRIPT_DIR}/${chart}"
      fi
    done
    return
  fi

  find "${SCRIPT_DIR}" \
    -mindepth 2 \
    -maxdepth 2 \
    -name Chart.yaml \
    -not -path "${PACKAGE_DIR}/*" \
    -print \
    | xargs -n1 dirname \
    | sort
}

main() {
  mkdir -p "${PACKAGE_DIR}"

  mapfile -t charts < <(find_charts)
  if [[ "${#charts[@]}" -eq 0 ]]; then
    echo "No charts found under ${SCRIPT_DIR}" >&2
    exit 1
  fi

  for chart_dir in "${charts[@]}"; do
    if [[ ! -f "${chart_dir}/Chart.yaml" ]]; then
      echo "Missing Chart.yaml: ${chart_dir}" >&2
      exit 1
    fi

    echo "==> Packaging $(basename "${chart_dir}")"

    if [[ "${SKIP_LINT}" != "true" ]]; then
      helm lint "${chart_dir}"
    fi

    helm package "${chart_dir}" --destination "${PACKAGE_DIR}"
  done

  echo "==> Packages written to ${PACKAGE_DIR}"
  find "${PACKAGE_DIR}" -maxdepth 1 -type f -name '*.tgz' -print | sort
}

main "$@"
