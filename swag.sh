#!/bin/bash

set -euo pipefail

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "${SCRIPT_DIR}"

# 自动发现领域和公开 API 包，新增领域时无需修改此脚本。
scan_dirs=$(
  find ./internal/domain ./pkg/api \
    -type f -name '*.go' ! -name '*_test.go' \
    -exec dirname {} \; | LC_ALL=C sort -u | paste -sd, -
)

swag init \
  --generalInfo main.go \
  --dir "./cmd/server,${scan_dirs}" \
  --parseInternal
