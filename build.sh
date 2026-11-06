#!/bin/bash

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
BIN_DIR="${SCRIPT_DIR}/bin"
module_text=$(head -n 1 "${SCRIPT_DIR}/go.mod" | awk '{print $NF}')
app_name=${module_text##*/}
commit_id=$(git -C "${SCRIPT_DIR}" rev-list -1 HEAD 2> /dev/null || echo 1)

mkdir -p "${BIN_DIR}"
CGO_ENABLED=0 go build \
  -ldflags "-X ${module_text}/internal/globals.CommitId=${commit_id}" \
  -o "${BIN_DIR}/${app_name}" \
  "${SCRIPT_DIR}/cmd/server"
