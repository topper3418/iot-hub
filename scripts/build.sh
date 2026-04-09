#!/usr/bin/env bash
# Directory: scripts/
# Modified: 2026-04-08
# Description: Builds both the frontend and backend by delegating to build_frontend.sh and build_backend.sh.
# Uses: scripts/build_frontend.sh, scripts/build_backend.sh
# Used by: scripts/install_pi.sh, scripts/update_pi.sh
set -euo pipefail

SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"$SCRIPTS_DIR/build_frontend.sh"
"$SCRIPTS_DIR/build_backend.sh"

echo "Full build complete."
