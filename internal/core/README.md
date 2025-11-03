# Core 模块 - CD 核心引擎

## 概述

Core 模块是 DevOps CD 系统的核心,负责管理发布批次的整个生命周期,实现了基于状态机的批次流转和自动化部署触发。

## 核心功能

### 1. 批次状态机

实现了完整的批次状态流转机制:

```
待审核(0) 
  ├─> 审核通过(5)
  │     └─> 封板确认(11)
  │           └─> 预发布部署中(21)
  │                 ├─> 预发布已部署(22)
  │                 │     └─> 正式发布部署中(31)
  │                 │           ├─> 正式发布已部署(32)
  │                 │           │     └─> 最终验收通过(40) ✓
  │                 │           │           
  │                 │           └─> 失败回滚到(22)
  │                 │
  │                 └─> 失败回滚到(11)
  │
  ├─> 审核拒绝(6)
  └─> 已取消(90)
```

### 2. 自动化扫描

#### 功能 1: 封板前构建更新

自动扫描处于`待审核`和`审核通过`状态的批次,检查是否有新的构建记录,并自动更新 `batch_releases` 表:

- 查询应用在相同分支的最新成功构建
- 对比当前构建和最新构建
- 如有更新,自动更新 `batch_releases` 中的记录
- 更新字段包括: `build_id`, `commit_id`, `image_name`, `image_tag` 等

#### 功能 2: 自动触发预发布

扫描处于`封板确认`状态的批次,自动触发预发布部署:

- 锁定所有发布记录(`is_locked = true`)
- 状态转换: 封板确认 -> 预发布部署中
- 调用部署服务执行实际部署
- 记录部署开始时间

#### 功能 3: 生产部署准备

扫描处于`预发布已部署`状态的批次,准备生产部署(需人工确认)

### 3. 事件驱动

支持通过事件触发状态转换:

| 事件 | 说明 | 起始状态 | 目标状态 |
|------|------|----------|----------|
| `approve` | 审核通过 | 待审核 | 审核通过 |
| `reject` | 审核拒绝 | 待审核 | 审核拒绝 |
| `cancel` | 取消批次 | 多个 | 已取消 |
| `confirm_tag` | 封板确认 | 审核通过 | 封板确认 |
| `start_pre_deploy` | 开始预发布 | 封板确认 | 预发布部署中 |
| `pre_deploy_completed` | 预发布完成 | 预发布部署中 | 预发布已部署 |
| `pre_deploy_failed` | 预发布失败 | 预发布部署中 | 封板确认 |
| `start_prod_deploy` | 开始生产部署 | 预发布已部署 | 正式发布部署中 |
| `prod_deploy_completed` | 生产部署完成 | 正式发布部署中 | 正式发布已部署 |
| `prod_deploy_failed` | 生产部署失败 | 正式发布部署中 | 预发布已部署 |
| `final_accept` | 最终验收 | 正式发布已部署 | 最终验收通过 |

## 核心组件

### 1. CoreEngine (core.go)

核心引擎,负责整体调度:

```go
engine := NewCoreEngine(db, logger)
engine.Start(30 * time.Second)  // 每30秒扫描一次
defer engine.Stop()

// 处理批次事件
engine.ProcessBatchEvent(batchID, "approve", "admin")
```

### 2. BatchStateMachine (state_machine.go)

状态机,定义状态转换规则:

```go
sm := NewBatchStateMachine()

// 检查是否可以转换
canTransit := sm.CanTransition(currentState, "approve")

// 执行转换
newState, err := sm.Transition(ctx, "approve")

// 获取可用转换
transitions := sm.GetAvailableTransitions(currentState)
```

### 3. BatchScanner (batch_scanner.go)

批次扫描器,执行自动化任务:

```go
scanner := NewBatchScanner(db, deployService, logger)

// 扫描并处理
scanner.ScanAndProcess()

// 处理状态变更
scanner.ProcessBatchStateChange(batchID, "approve", "admin")
```

### 4. DeployService (deploy_service.go)

部署服务接口(当前为模拟实现):

```go
// 当前使用模拟服务
deployService := NewMockDeployService(logger)

// 真实实现需要集成 Kubernetes/Helm/Docker
// type RealDeployService struct { ... }
```

## 数据模型

### Batch (批次)

```go
type Batch struct {
    ID                   int64
    BatchNumber          string     // 批次编号
    Initiator            string     // 发起人
    ApprovedBy           *string    // 审批人
    ApprovedAt           *time.Time // 审批时间
    Status               int8       // 状态
    TaggedAt             *time.Time // 封板时间
    PreDeployStartedAt   *time.Time // 预发布开始
    PreDeployFinishedAt  *time.Time // 预发布完成
    ProdDeployStartedAt  *time.Time // 生产部署开始
    ProdDeployFinishedAt *time.Time // 生产部署完成
    FinalAcceptedAt      *time.Time // 最终验收
    // ... 其他字段
}
```

### BatchRelease (批次发布记录)

```go
type BatchRelease struct {
    ID            int64
    BatchID       int64
    AppID         int64
    BuildID       int64   // 当前使用的构建
    LatestBuildID *int64  // 最新构建(封板前更新)
    ImageName     string
    ImageTag      string
    CommitID      string
    Branch        string
    Status        string
    IsLocked      bool    // 是否已锁定(封板后)
    // ... 其他字段
}
```

### Build (构建记录)

```go
type Build struct {
    ID            int64
    BuildNumber   string
    ApplicationID int64
    Branch        string
    CommitID      string
    ImageName     *string
    ImageTag      *string
    BuildStatus   string  // success/failed/...
    // ... 其他字段
}
```

## 使用示例

### 示例 1: 完整的发布流程

```go
package main

import (
    "time"
    "devops-cd/internal/core"
    "go.uber.org/zap"
    "gorm.io/gorm"
)

func main() {
    // 初始化
    db := initDatabase()
    logger := initLogger()
    
    // 创建核心引擎
    engine := core.NewCoreEngine(db, logger)
    
    // 启动引擎(每30秒扫描一次)
    engine.Start(30 * time.Second)
    defer engine.Stop()
    
    batchID := int64(1)
    
    // 1. 审核通过
    if err := engine.ProcessBatchEvent(batchID, "approve", "admin"); err != nil {
        logger.Error("审核失败", zap.Error(err))
        return
    }
    
    // 2. 封板确认
    if err := engine.ProcessBatchEvent(batchID, "confirm_tag", "admin"); err != nil {
        logger.Error("封板失败", zap.Error(err))
        return
    }
    
    // 3. 等待自动触发预发布(由扫描器自动执行)
    // 扫描器会检测到状态为"封板确认",自动转换为"预发布部署中"
    
    // 4. 预发布完成(由部署系统回调)
    time.Sleep(5 * time.Minute)
    if err := engine.ProcessBatchEvent(batchID, "pre_deploy_completed", "system"); err != nil {
        logger.Error("预发布完成失败", zap.Error(err))
        return
    }
    
    // 5. 验收通过,触发生产部署
    if err := engine.ProcessBatchEvent(batchID, "start_prod_deploy", "admin"); err != nil {
        logger.Error("生产部署启动失败", zap.Error(err))
        return
    }
    
    // 6. 生产部署完成
    time.Sleep(10 * time.Minute)
    if err := engine.ProcessBatchEvent(batchID, "prod_deploy_completed", "system"); err != nil {
        logger.Error("生产部署完成失败", zap.Error(err))
        return
    }
    
    // 7. 最终验收
    if err := engine.ProcessBatchEvent(batchID, "final_accept", "admin"); err != nil {
        logger.Error("最终验收失败", zap.Error(err))
        return
    }
    
    logger.Info("发布流程完成!")
}
```

### 示例 2: 查询批次状态

```go
status, err := engine.GetBatchStatus(batchID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("批次ID: %v\n", status["batch_id"])
fmt.Printf("当前状态: %v\n", status["current_state_name"])
fmt.Printf("可用操作:\n")
for _, event := range status["available_events"].([]map[string]string) {
    fmt.Printf("  - %s: %s -> %s\n", 
        event["event"], 
        event["description"], 
        event["target_state"])
}
```

输出:
```
批次ID: 1
当前状态: 审核通过
可用操作:
  - confirm_tag: 封板确认 -> 封板确认
  - cancel: 取消批次 -> 已取消
```

### 示例 3: 手动触发扫描

```go
scanner := core.NewBatchScanner(db, deployService, logger)

// 立即执行一次扫描
if err := scanner.ScanAndProcess(); err != nil {
    logger.Error("扫描失败", zap.Error(err))
}
```

## 扩展开发

### 1. 实现真实的部署服务

需要实现 `DeployService` 接口:

```go
type RealDeployService struct {
    k8sClient *kubernetes.Clientset
    logger    *zap.Logger
}

func (s *RealDeployService) TriggerPreDeploy(batchID int64, releases []core.BatchRelease) error {
    // 1. 创建 Kubernetes Deployment
    // 2. 配置 Service 和 Ingress
    // 3. 执行健康检查
    // 4. 监控部署进度
    // 5. 更新部署状态到数据库
    return nil
}

func (s *RealDeployService) TriggerProdDeploy(batchID int64, releases []core.BatchRelease) error {
    // 实现生产部署逻辑
    return nil
}
```

### 2. 添加自定义状态转换处理

修改 `state_machine.go` 中的 handler 函数:

```go
func (sm *BatchStateMachine) handleApprove(ctx *TransitionContext) error {
    // 自定义审核通过逻辑
    ctx.Batch.ApprovedBy = &ctx.Operator
    now := time.Now()
    ctx.Batch.ApprovedAt = &now
    
    // 发送通知
    sendNotification("批次已审核通过", ctx.Batch)
    
    // 记录审计日志
    auditLog.Record("batch_approved", ctx.BatchID, ctx.Operator)
    
    return nil
}
```

### 3. 集成通知服务

```go
// 在状态转换时发送通知
type NotificationService interface {
    SendBatchNotification(batch *core.Batch, event string) error
}

// 集成 Lark/钉钉/邮件
func sendLarkNotification(batch *core.Batch, event string) error {
    // 调用 Lark API 发送消息
    return nil
}
```

## 技术要点

- **并发控制**: 使用channel实现worker pool,固定5个并发
- **重试策略**: 指数退避(1s, 2s, 4s),最多重试3次
- **状态追踪**: 应用级状态独立,失败不影响批次状态
- **通知适配**: 接口抽象,支持Lark/日志/多渠道通知
- **K8s集成**: 通过接口与底层K8s部署服务交互

## 配置说明

在 `configs/base.yaml` 中配置 core 模块:

```yaml
core:
  scan_interval: 30s  # 批次扫描间隔
  deploy:
    concurrent_apps: 5              # 并行部署应用数
    single_app_timeout: 10m         # 单应用部署超时
    batch_timeout: 60m              # 批次部署超时
    retry_count: 3                  # 部署失败重试次数
    retry_backoff: exponential      # 重试策略: exponential/linear
    poll_interval: 5s               # 部署状态轮询间隔
  notification:
    enabled: true                   # 是否启用通知
    provider: lark                  # 通知渠道: lark/log
    lark_webhook: "https://..."     # Lark Webhook URL
  k8s:
    base_url: "http://k8s-deploy-service:8080"  # K8s部署服务地址
    api_key: ""                                  # API密钥
```

## 监控指标

建议监控的指标:

- 批次总数和各状态分布
- 平均部署时长
- 部署成功率
- 状态转换失败次数
- 扫描器执行频率和耗时

## 待实现功能

- [ ] 真实的 Kubernetes 部署集成(HTTP/gRPC客户端)
- [ ] Helm Chart 支持
- [ ] 灰度发布策略(金丝雀/蓝绿部署)
- [ ] 钉钉/邮件通知适配器
- [ ] 性能监控和告警
- [x] Worker Pool并发控制
- [x] 自动重试机制
- [x] Lark通知集成
- [x] 部署状态追踪

## 注意事项

1. **封板前更新**: 只有未封板的批次才会自动更新构建记录
2. **锁定机制**: 封板后 `batch_releases` 会被锁定,不再更新
3. **状态一致性**: 所有状态转换都经过状态机验证
4. **并发安全**: 使用数据库事务保证并发安全
5. **日志记录**: 所有操作都有详细的日志记录

---

**版本**: v1.0.0  
**更新日期**: 2025-10-16

