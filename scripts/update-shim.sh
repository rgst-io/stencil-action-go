#!/usr/bin/env bash
# Updates the action-go-shim to the latest version.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

shimDir="$REPO_ROOT/shim"

rm -rf "$shimDir"
mkdir -p "$shimDir"

gh --repo rgst-io/action-go-shim \
	release download --dir "$shimDir" --pattern '*'

chmod +x "$shimDir/action-go-shim-"*
rm -f "$shimDir/action.yml"
