# Hub curl 测试

这些命令验证 Console 使用的 HTTP API；HTTP 不直接启动 Agent，实际执行仍经过 Hub → gRPC/TLS → reind → Backend。需要 `curl`、`jq`、Python 3 和相邻的 rein、proto、a2alinker 仓库。

## 1. 启动可重复的测试环境

在 reinhub 根目录编译当前代码，再启动已有测试栈：

```sh
go build -o /tmp/reinhub-dev-console ./cmd/server
(cd ../a2alinker && cargo build --bins --locked)
(cd ../rein && cargo build --bins --locked)
REIN_TEST_HTTP_PORT=19010 REIN_TEST_GRPC_PORT=19011 REIN_TEST_A2A_PORT=19012 \
  python3 ../rein/tests/hub_stack.py --serve
```

等待 `{"ready": true, ...}`。该脚本使用临时 Hub 数据、daemon SQLite、工作目录及确定性 Agent 子进程，自动通过 CLI 注册 `codex`、`pi`、`a2a-pi`。原生路径实际走管道，A2A 路径实际调用独立 a2alinker HTTP 服务；均不请求模型。

另开终端执行后续命令。测试结束对测试栈按 Ctrl-C；脚本停止自己创建的进程并清理临时目录。不要删除或清空 `~/.reinhub`、`~/.rein`。

测试真实 Agent 时，按 [启动说明](../rein/docs/console-integration.md) 启动 Hub 并注册，再将下文 `HUB_HTTP` 改为 `http://127.0.0.1:9010`。`approval`、`slow` 是 fixture 指令，真实模型不保证触发对应行为。

## 2. 健康检查与注册列表

```sh
HUB_HTTP=http://127.0.0.1:19010
curl -fsS "$HUB_HTTP/api/health" | jq .
curl -fsS "$HUB_HTTP/api/agents" | jq '.agents[] | {agent, backend: .runtime.backend, online}'
AGENT_ID=$(curl -fsS "$HUB_HTTP/api/agents" | jq -er '.agents[] | select(.agent.name == "codex" and .online) | .agent.id')
```

预期健康状态为 `ok`，三个 Agent 在线，分别含 Native / A2A Backend。HTTP 请求字段使用 snake_case；嵌套的协议响应使用 protobuf JSON，例如 `attemptId`、`sessionId`。`sequence` 是字符串形式的 uint64；协议 bytes 字段如 `parametersJson` 是 base64。

## 3. 创建、查询、事件和幂等

```sh
REQUEST_ID=$(uuidgen)
BODY=$(jq -nc --arg request "$REQUEST_ID" --arg agent "$AGENT_ID" \
  '{request_id:$request, agent_id:$agent, prompt:"hello"}')
CREATED=$(curl -fsS "$HUB_HTTP/api/runs" -H 'Content-Type: application/json' --data-binary "$BODY")
printf '%s\n' "$CREATED" | jq .
RUN_ID=$(printf '%s' "$CREATED" | jq -er '.run.id')

# 相同 request_id 和输入重试，返回同一个 Run；不会启动第二次执行。
RETRIED=$(curl -fsS "$HUB_HTTP/api/runs" -H 'Content-Type: application/json' --data-binary "$BODY")
test "$RUN_ID" = "$(printf '%s' "$RETRIED" | jq -er '.run.id')"

curl -fsS "$HUB_HTTP/api/runs/$RUN_ID" | jq .
curl -fsS --get "$HUB_HTTP/api/runs" --data-urlencode "agent_id=$AGENT_ID" | jq .
```

下列小函数只用于手工测试，最多等待 30 秒，超时返回失败。正常流程为 queued → dispatched → running → completed；中间状态可能很快，轮询不一定逐个看到。

```sh
hub_wait() {
  for poll in $(seq 1 150); do
    CURRENT=$(curl -fsS "$HUB_HTTP/api/runs/$RUN_ID") || return 1
    STATE=$(printf '%s' "$CURRENT" | jq -er '.run.state') || return 1
    if [ "$STATE" = "$1" ]; then return 0; fi
    case "$STATE" in RUN_STATE_COMPLETED|RUN_STATE_FAILED|RUN_STATE_CANCELLED) printf '%s\n' "$CURRENT"; return 1;; esac
    sleep 0.2
  done
  echo "等待 $1 超时，当前为 $STATE" >&2
  return 1
}
hub_wait RUN_STATE_COMPLETED
ATTEMPT_ID=$(printf '%s' "$CURRENT" | jq -er '.run.attemptId')
SESSION_ID=$(printf '%s' "$CURRENT" | jq -er '.run.sessionId')
EVENTS=$(curl -fsS --get "$HUB_HTTP/api/runs/$RUN_ID/events" \
  --data-urlencode "attempt_id=$ATTEMPT_ID" --data-urlencode 'after=0')
printf '%s\n' "$EVENTS" | jq .
AFTER=$(printf '%s' "$EVENTS" | jq -r 'last.sequence // "0"')
curl -fsS --get "$HUB_HTTP/api/runs/$RUN_ID/events" \
  --data-urlencode "attempt_id=$ATTEMPT_ID" --data-urlencode "after=$AFTER" | jq .
```

事件按 attempt 内序号递增，最后一项为 completed Result，`executionStopped` 为 true。一个响应最多 100 条且有字节限制；长输出重复使用最后收到的 sequence 查询，直到返回空数组。文本 `complete:true` 是完整快照，应替换对应 item，不重复拼接。

在同一会话继续，fixture 应输出 `Hello 2`：

```sh
jq -nc --arg request "$(uuidgen)" --arg agent "$AGENT_ID" --arg session "$SESSION_ID" \
  '{request_id:$request,agent_id:$agent,session_id:$session,prompt:"again"}' |
  curl -fsS "$HUB_HTTP/api/runs" -H 'Content-Type: application/json' --data-binary @- | jq .
```

## 4. 审批回复

以下使用 Codex fixture。Pi 和 `a2a-pi` 改用 `response:{confirmed:true}`；其他输入必须按 interaction 方法填写，不能把所有交互当成批准。

```sh
CREATED=$(jq -nc --arg request "$(uuidgen)" --arg agent "$AGENT_ID" \
  '{request_id:$request,agent_id:$agent,prompt:"approval"}' |
  curl -fsS "$HUB_HTTP/api/runs" -H 'Content-Type: application/json' --data-binary @-)
RUN_ID=$(printf '%s' "$CREATED" | jq -er '.run.id')
hub_wait RUN_STATE_WAITING_APPROVAL
ATTEMPT_ID=$(printf '%s' "$CURRENT" | jq -er '.run.attemptId')
INTERACTION_ID=$(curl -fsS --get "$HUB_HTTP/api/runs/$RUN_ID/events" \
  --data-urlencode "attempt_id=$ATTEMPT_ID" --data-urlencode 'after=0' |
  jq -er '[.[] | select(.interaction != null)] | last.interaction.interactionId')
COMMAND_ID=$(uuidgen)
REPLY=$(jq -nc --arg command "$COMMAND_ID" --arg attempt "$ATTEMPT_ID" --arg interaction "$INTERACTION_ID" \
  '{command_id:$command,attempt_id:$attempt,interaction_id:$interaction,response:{decision:"accept"}}')
curl -fsS "$HUB_HTTP/api/runs/$RUN_ID/reply" -H 'Content-Type: application/json' --data-binary "$REPLY" | jq .
hub_wait RUN_STATE_COMPLETED
```

返回 `commandId` 只代表 Hub 已持久化命令，不代表 Agent 已执行。`GET /api/runs?agent_id=...` 的 `command_acks` 可查看 daemon 回执。原 command_id/输入可重试；相同 interaction 换 command_id 重复回答会拒绝，避免多次授权。

## 5. 取消与错误边界

```sh
CREATED=$(jq -nc --arg request "$(uuidgen)" --arg agent "$AGENT_ID" \
  '{request_id:$request,agent_id:$agent,prompt:"slow"}' |
  curl -fsS "$HUB_HTTP/api/runs" -H 'Content-Type: application/json' --data-binary @-)
RUN_ID=$(printf '%s' "$CREATED" | jq -er '.run.id')
hub_wait RUN_STATE_RUNNING
ATTEMPT_ID=$(printf '%s' "$CURRENT" | jq -er '.run.attemptId')
jq -nc --arg command "$(uuidgen)" --arg attempt "$ATTEMPT_ID" \
  '{command_id:$command,attempt_id:$attempt}' |
  curl -fsS "$HUB_HTTP/api/runs/$RUN_ID/cancel" -H 'Content-Type: application/json' --data-binary @- | jq .
hub_wait RUN_STATE_CANCELLED
curl -fsS --get "$HUB_HTTP/api/runs/$RUN_ID/events" \
  --data-urlencode "attempt_id=$ATTEMPT_ID" --data-urlencode 'after=0' | jq 'last.result'

# 预期 409：不能用其他 attempt 读取事件。
curl -sS -w '\nHTTP %{http_code}\n' --get "$HUB_HTTP/api/runs/$RUN_ID/events" \
  --data-urlencode 'attempt_id=wrong-attempt' --data-urlencode 'after=0'
# 预期 400：严格 JSON 解析。
curl -sS -w '\nHTTP %{http_code}\n' "$HUB_HTTP/api/runs" -H 'Content-Type: application/json' --data-binary '{'
# 预期 403：拒绝跨站驱动本机进程。
curl -sS -w '\nHTTP %{http_code}\n' "$HUB_HTTP/api/agents" -H 'Origin: https://example.invalid'
```

取消意图 `cancelRequested` 与取消结果分开；以 cancelled Result 且 `executionStopped:true` 为确认。断线不能判断 Agent 完成或停止。Console API 使用标准 HTTP 状态码及 `{"error":"..."}`，并非模板示例的 HTTP 200 业务封装。

## 6. HTTP 历史分页与 SSE 实时增量

Console 页面打开时通过 HTTP 获取所选会话最近 20 轮，SSE 不负责首次加载历史。返回的 `sessions` 是会话摘要；不指定或指定已不存在的 session_id 时选择最近会话。

```sh
HISTORY=$(curl -fsS --get "$HUB_HTTP/api/agents/$AGENT_ID/history" --data-urlencode 'limit=20')
printf '%s\n' "$HISTORY" | jq .
SESSION_ID=$(printf '%s' "$HISTORY" | jq -r '.session_id')
BEFORE=$(printf '%s' "$HISTORY" | jq -r '.next_cursor')
STREAM_CURSOR=$(printf '%s' "$HISTORY" | jq -er '.stream_cursor')

# next_cursor 非空时获取更早一页；新消息不会挤动分页边界。
if [ -n "$BEFORE" ]; then
  curl -fsS --get "$HUB_HTTP/api/agents/$AGENT_ID/history" \
    --data-urlencode 'limit=20' --data-urlencode "session_id=$SESSION_ID" \
    --data-urlencode "before=$BEFORE" | jq .
fi

# 保持此终端连接；从其他终端创建任务、回复或取消来观察推送，Ctrl-C 退出。
curl -NsS --get "$HUB_HTTP/api/agents/$AGENT_ID/events" \
  --data-urlencode "cursor=$STREAM_CURSOR"
```

每轮历史包含协议消息的 JSON 表示、控制回执和完整事件，按时间正序返回。limit 范围 1–50；next_cursor 只适用于相同 Agent 和会话。当前仍受单机总快照 64 MiB 上限约束，一轮很长时响应也会较大。

SSE 的 `update` 包含 RunView 和本批次新增事件；`checkpoint` 的 id 是已完整发送批次的提交游标。游标包含数据集 epoch 和持久化修订号，用字符串传输。重连时可发送 `Last-Event-ID: <最后一个 checkpoint id>`，它优先于 URL cursor；不能在收到半个批次时自行提前推进游标。重复事件按 `(run_id, attempt_id, sequence)` 去重，RunView 按 version 合并，避免较早 HTTP 响应覆盖新状态。

只有提交持久化完成后才会唤醒 SSE。没有变化时仅发送 15 秒一次的保活注释；慢连接单次写超时 10 秒，不阻塞 daemon 事件提交。原有普通 HTTP 写超时保持独立。数据集被替换或游标超出当前范围时，SSE 发送 `reset`，客户端重新通过 HTTP 加载最近对话。checkpoint 是流位置，不代表 Agent 已执行命令；daemon 的持久化 ACK 语义保持不变。

## 7. 自动化检查和已知缺口

```sh
go test ./...
go test -race ./...
# 停止上面的 --serve 栈后可运行完整隔离测试，包括重启恢复。
python3 ../rein/tests/hub_stack.py
```

当前快照限 512 个 Run、64 MiB，无多实例/登录支持。审查发现的待修项：空 ClaimRuns 去重记录持续增长；过期启动事件可能卡住上报和执行；HTTP 入口尚未接入模板访问日志/请求 ID。正常链路测试通过不代表这些边界已修复。不要通过自动重跑结果未知的执行绕过恢复问题。
