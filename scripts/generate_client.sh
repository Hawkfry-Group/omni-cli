#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
in_spec="$repo_root/api/openapi.json"
out_spec="$repo_root/api/openapi.codegen.json"

if [[ ! -f "$in_spec" ]]; then
  echo "missing $in_spec"
  exit 1
fi

jq '
  .openapi = "3.0.3"
  | walk(
      if type == "object" then
        (
          if (.type | type?) == "string" and .type == "null" then
            .type = "string"
            | .nullable = true
            | del(.enum)
          else . end
        )
        |
        (
          if (.type | type?) == "array" then
            .nullable = ((.nullable // false) or ((.type | index("null")) != null))
            | .type = ([.type[] | select(. != "null")] | .[0] // "string")
          else . end
        )
        |
        (
          if (.exclusiveMinimum | type?) == "number" then
            .minimum = .exclusiveMinimum
            | .exclusiveMinimum = true
          else . end
        )
        | (
          if (.exclusiveMaximum | type?) == "number" then
            .maximum = .exclusiveMaximum
            | .exclusiveMaximum = true
          else . end
        )
      else . end
    )
' "$in_spec" > "$out_spec"

(
  cd "$repo_root"
  go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen \
    -response-type-suffix Resp \
    -config "api/oapi-codegen.yaml" \
    "api/openapi.codegen.json"
)
echo "generated internal/client/gen/client.gen.go"
