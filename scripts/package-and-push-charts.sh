#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"${SCRIPT_DIR}/package-charts.sh"
"${SCRIPT_DIR}/push-charts.sh" "$@"
