#!/bin/bash

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
module_text=$(head -n 1 "${SCRIPT_DIR}/go.mod" | awk '{print $NF}')

# 检查是否包含斜杠
if [[ $module_text == */* ]]; then
    # 使用awk获取最后一个斜杠后面的内容
    module_text=$(echo "$module_text" | awk -F'/' '{print $NF}')
fi

# 检查操作系统类型mingw
if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" ]]; then
    module_text="${module_text}.exe"
fi

cd "${SCRIPT_DIR}/bin" || exit 1
exec "./${module_text}" "$@"
