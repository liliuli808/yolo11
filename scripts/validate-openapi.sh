#!/usr/bin/env bash
set -euo pipefail

# validate-openapi.sh
#
# Validates the canonical OpenAPI contract for the Lantern API.
# Installs the Redocly CLI validator locally under .build/openapi-validator on
# first run. The .build directory is gitignored and should never be committed.

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OPENAPI_FILE="$REPO_ROOT/contracts/openapi/openapi.yaml"
VALIDATOR_DIR="$REPO_ROOT/.build/openapi-validator"
VALIDATOR_BIN="$VALIDATOR_DIR/node_modules/.bin/redocly"

if [ ! -f "$OPENAPI_FILE" ]; then
  echo "Error: OpenAPI file not found: $OPENAPI_FILE" >&2
  exit 1
fi

if ! command -v npm >/dev/null 2>&1; then
  echo "Error: npm is required to install the OpenAPI validator." >&2
  echo "Please install Node.js/npm or provide a validator manually." >&2
  exit 1
fi

if [ ! -x "$VALIDATOR_BIN" ]; then
  echo "Installing Redocly CLI OpenAPI validator to $VALIDATOR_DIR ..." >&2
  mkdir -p "$VALIDATOR_DIR"
  npm install --prefix "$VALIDATOR_DIR" @redocly/cli@latest --no-audit --no-fund >&2
fi

echo "Linting $OPENAPI_FILE ..." >&2
"$VALIDATOR_BIN" lint "$OPENAPI_FILE"
