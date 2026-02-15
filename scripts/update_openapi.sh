#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <openapi-url>"
  echo "example: $0 https://your-instance.omniapp.co/openapi.json"
  exit 1
fi

url="$1"
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out_file="$repo_root/api/openapi.json"

tmp_file="${out_file}.tmp"

curl -fsSL "$url" -o "$tmp_file"

if ! jq -e '.openapi and .paths and .components' "$tmp_file" >/dev/null; then
  echo "downloaded file is not a valid OpenAPI document"
  rm -f "$tmp_file"
  exit 1
fi

mv "$tmp_file" "$out_file"

echo "wrote $out_file"
echo "run: go generate ./internal/client/gen"
