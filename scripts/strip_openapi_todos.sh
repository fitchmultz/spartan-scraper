#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat <<'EOF'
Strip TODO comment lines from generated OpenAPI TypeScript output.

Usage:
  scripts/strip_openapi_todos.sh --path <dir>

Example:
  scripts/strip_openapi_todos.sh --path web/src/api
EOF
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
	usage
	exit 0
fi

path=""
while [[ $# -gt 0 ]]; do
	case "$1" in
		--path)
			path="${2:-}"
			shift 2
			;;
		*)
			echo "Unknown arg: $1" >&2
			usage
			exit 2
			;;
	esac
done

if [[ -z "$path" ]]; then
	echo "Missing --path" >&2
	usage
	exit 2
fi

if [[ ! -d "$path" ]]; then
	echo "Path not found: $path" >&2
	exit 2
fi

matches="$(rg -l "TODO:" "$path" || true)"
if [[ -z "$matches" ]]; then
	exit 0
fi

while read -r file; do
	[[ -z "$file" ]] && continue
	perl -0pi -e 's/^.*TODO:.*\n//mg' "$file"
done <<< "$matches"
