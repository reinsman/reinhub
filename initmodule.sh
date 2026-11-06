#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

usage() {
  echo "用法: $0 <新 module 名>" >&2
  echo "示例: $0 github.com/someone/someserver" >&2
  echo "      $0 someserver" >&2
}

if [[ $# -ne 1 ]]; then
  usage
  exit 2
fi

new_module=$1
if [[ -z "${new_module}" || "${new_module}" =~ [[:space:]] ||
      "${new_module}" == /* || "${new_module}" == */ ||
      "${new_module}" == *//* || "${new_module}" == *\\* ]]; then
  echo "错误: module 名不能为空，且不能包含空白、反斜杠、首尾斜杠或连续斜杠。" >&2
  exit 2
fi

cd "${SCRIPT_DIR}"
old_module=$(go list -m)

if [[ "${new_module}" == "${old_module}" ]]; then
  echo "项目已经使用 module ${new_module}，无需修改。"
  exit 0
fi

go run ./tools/initmodule "${old_module}" "${new_module}"
go mod edit -module="${new_module}"

echo "module 初始化完成: ${old_module} -> ${new_module}"
