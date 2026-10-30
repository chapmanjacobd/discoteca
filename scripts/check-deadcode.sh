#!/usr/bin/env bash
# Fail when production code contains functions that deadcode cannot reach.
# Test helpers in internal/testutils are intentionally exempt.
set -euo pipefail

cd "$(dirname "$0")/.."

allowlist="scripts/deadcode-allowlist.txt"
if ! command -v deadcode >/dev/null 2>&1; then
	echo "deadcode is not installed; run: go install golang.org/x/tools/cmd/deadcode@latest" >&2
	exit 1
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

deadcode -tags fts5 ./... >"$tmp_dir/raw.txt" 2>&1

awk -F': unreachable func: ' '
	/: unreachable func: / {
		file = $1
		sub(/:[0-9]+:[0-9]+$/, "", file)
		if (file ~ /^internal\/testutils\//) {
			next
		}
		print file ":" $2
	}
' "$tmp_dir/raw.txt" | sort -u >"$tmp_dir/current.txt"

{
	grep -v '^[[:space:]]*#' "$allowlist" || true
} | sed '/^[[:space:]]*$/d' | sort -u >"$tmp_dir/allowlist.txt"

if ! diff -u "$tmp_dir/allowlist.txt" "$tmp_dir/current.txt"; then
	echo >&2
	echo "Dead-code set changed. Remove unreachable functions instead of growing the allowlist." >&2
	echo "Allowlist: $allowlist" >&2
	exit 1
fi
