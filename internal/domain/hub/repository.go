package hub

import "gorm.io/gorm"

// Repository 持有模板已有的数据库依赖；不在 gRPC Handler 中拼接 SQL。
type Repository struct{ db *gorm.DB }

// TODO: 增加接收 context.Context 的事务操作：
// 1. 授权后查询/注册 Agent、Runtime、Session。
// 2. 幂等领取 Run，更新当前 attempt 和启动租约。
// 3. 按 (run_id, attempt_id, sequence) 去重事件，并在同一事务更新状态和用量。
// 4. 按 command_id 去重控制意图/ACK；事务提交后再发送确认或唤醒。
// 模型在 model.go 定义；框架阶段尚不注册迁移、不创建假执行结果。
