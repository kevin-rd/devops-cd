# 新 Tag 处理方案 - 简化实现计划

## 📋 核心需求总结

基于讨论，新 Tag 处理的核心需求是：

1. **Drone 构建完成后推送新 tag 信息**
2. **在特定条件下允许用户切换版本**
3. **PreDeploying 状态下用户可见新 tag 但不能切换**
4. **每个 ReleaseApp 独立控制版本**
5. **任何有权限的用户都可以切换**
6. **使用状态机管理，操作幂等**

## 🏗️ 数据模型设计

### ReleaseApp 表扩展

需要添加以下字段到 `release_apps` 表：

```sql
ALTER TABLE release_apps ADD COLUMN (
  `latest_build_id` BIGINT COMMENT '最新检测到的构建ID',
  `has_new_tag` BOOLEAN DEFAULT FALSE COMMENT '是否有新tag待处理',
  `tag_updated_at` TIMESTAMP NULL COMMENT '新tag更新时间',
  `tag_updated_by` VARCHAR(50) COMMENT '新tag更新者'
);
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `build_id` | BIGINT | 当前使用的构建ID（封板时固定） |
| `latest_build_id` | BIGINT | 最新检测到的构建ID（新tag到达时更新） |
| `target_tag` | VARCHAR | 当前使用的版本tag（从 build.image_tag 获取） |
| `has_new_tag` | BOOLEAN | 是否有新tag待处理 |
| `tag_updated_at` | TIMESTAMP | 新tag更新时间 |
| `tag_updated_by` | VARCHAR | 新tag更新者 |

### Deployment 表扩展

需要添加以下字段到 `deployments` 表：

```sql
ALTER TABLE deployments ADD COLUMN (
  `is_superseded` BOOLEAN DEFAULT FALSE COMMENT '是否已被新版本替代',
  `superseded_at` TIMESTAMP NULL COMMENT '被替代时间',
  `superseded_by` BIGINT COMMENT '被哪个Deployment替代'
);
```

## 🔄 核心流程

### 1. 新 Tag 检测流程

```
Drone 构建完成
    ↓
推送 Webhook 到服务
    ↓
服务接收 Webhook，获取 app_id 和 image_tag
    ↓
查询所有包含该 app 的活跃 batch（status != Completed/Cancelled）
    ↓
对每个 ReleaseApp：
  - 如果 latest_build_id 与 build_id 相同，跳过
  - 否则，更新 latest_build_id 和 has_new_tag=true
  - 记录 tag_updated_at 和 tag_updated_by
    ↓
返回成功
```

### 2. 版本切换流程

```
用户调用 API：POST /releases/{release_id}/switch-version
    ↓
检查前置条件：
  - ReleaseApp 存在
  - has_new_tag = true
  - Batch 状态允许切换（PreWaiting/ProdWaiting）
  - 没有正在运行的 Deployment（status != running）
    ↓
如果检查失败，返回错误
    ↓
开始事务：
  1. 更新 ReleaseApp：
     - build_id = latest_build_id
     - target_tag = 新版本的 image_tag
     - has_new_tag = false
     - latest_build_id = null

  2. 标记旧 Deployment 为 superseded：
     - 查询该 ReleaseApp 的所有未完成 Deployment
     - 对每个 Deployment，标记为 superseded=true, superseded_at=now()
     - 不需要取消 K8s 任务（状态机会处理）

  3. 创建新 Deployment：
     - 使用新的 build_id 和 image_tag
     - 状态设为 pending
     - 状态机会自动处理后续流程
    ↓
提交事务
    ↓
返回成功
```

## 🎯 API 设计

### 1. 检测新 Tag

**Endpoint:** `POST /api/v1/builds/notify`

**请求体：** 来自 Drone 的 Webhook

```json
{
  "build_number": 11,
  "build_status": "success",
  "build_created": 1234567890,
  "build_started": 1234567891,
  "build_finished": 1234567900,
  "build_link": "https://drone.example.com/...",
  "commit_sha": "abc123...",
  "commit_ref": "refs/tags/v1.2.3",
  "commit_branch": "main",
  "commit_message": "Release v1.2.3",
  "commit_link": "https://github.com/...",
  "commit_author": "user@example.com",
  "apps": [
    {
      "name": "app-name",
      "image_tag": "v1.2.3",
      "image": "registry.example.com/app:v1.2.3",
      "build_success": true
    }
  ]
}
```

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "affected_releases": 3,
    "details": [
      {
        "batch_id": 1,
        "release_id": 1,
        "app_id": 1,
        "old_tag": "v1.2.2",
        "new_tag": "v1.2.3"
      }
    ]
  }
}
```

### 2. 切换版本

**Endpoint:** `POST /api/v1/releases/{release_id}/switch-version`

**请求体：**

```json
{
  "reason": "切换到新版本"
}
```

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "release_id": 1,
    "old_build_id": 1,
    "new_build_id": 2,
    "old_tag": "v1.2.2",
    "new_tag": "v1.2.3",
    "affected_deployments": 1,
    "new_deployment_id": 10
  }
}
```

### 3. 查询 ReleaseApp 状态

**Endpoint:** `GET /api/v1/releases/{release_id}/status`

**响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "release_id": 1,
    "batch_id": 1,
    "app_id": 1,
    "current_build_id": 1,
    "current_tag": "v1.2.2",
    "latest_build_id": 2,
    "latest_tag": "v1.2.3",
    "has_new_tag": true,
    "tag_updated_at": "2024-01-15T10:00:00Z",
    "tag_updated_by": "user@example.com",
    "can_switch": true,
    "reason": ""
  }
}
```

## 🔐 状态转换规则

### 允许切换版本的条件

```
Batch 状态 = PreWaiting (20) 或 ProdWaiting (30)
AND
ReleaseApp.has_new_tag = true
AND
ReleaseApp.latest_build_id != null
AND
没有正在运行的 Deployment（status != "running"）
```

### 禁止切换版本的条件

```
Batch 状态 = PreDeploying (21) 或 ProdDeploying (31)
OR
ReleaseApp.has_new_tag = false
OR
有正在运行的 Deployment（status = "running"）
```

### Deployment 状态机

根据现有代码，Deployment 的状态流转为：
- `pending` → `running` → `success` / `failed`
- `pending` → `waiting_dependencies` → `pending` → ...

当版本切换时，旧 Deployment 标记为 `is_superseded=true`，但状态机仍会继续处理。

## 📝 实现步骤

### Step 1: 数据库迁移

创建迁移脚本 `scripts/xxx_add_new_tag_fields.sql`
- 为 `release_apps` 表添加 4 个字段
- 为 `deployments` 表添加 3 个字段

### Step 2: 更新模型

在 `internal/model/release.go` 中：
- 为 `ReleaseApp` 添加 4 个新字段

在 `internal/model/deploy.go` 中：
- 为 `Deployment` 添加 3 个新字段

### Step 3: 实现 Webhook 处理

在 `internal/handler/build_handler.go` 中实现 `NotifyNewTag` 方法：
- 接收 Drone Webhook
- 查询所有包含该 app 的活跃 batch
- 更新 `latest_build_id` 和 `has_new_tag`

### Step 4: 实现版本切换逻辑

在 `internal/core/release_app/` 中创建 `version_switcher.go`：
- 实现 `SwitchVersion` 方法
- 检查前置条件
- 事务处理：更新 ReleaseApp、标记旧 Deployment、创建新 Deployment

### Step 5: 添加 API 端点

在 `api/handler/release_handler.go` 中添加：
- `SwitchVersion` - 切换版本
- `GetReleaseStatus` - 查询状态

### Step 6: 添加状态机 Action

在 `internal/core/release_app/outside_action.go` 中实现 `new_tag` action

### Step 7: 测试

编写单元测试和集成测试

## 🛡️ 关键实现细节

### 1. 幂等性处理

由于使用状态机管理，同一个操作多次调用应该是幂等的：

```go
// 如果 has_new_tag = false，说明已经处理过，直接返回成功
if !release.HasNewTag {
    return nil
}
```

### 2. 事务处理

版本切换必须在事务中进行，确保数据一致性（参考 ReleaseStateMachine.UnifiedUpdate）：

```go
return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    // 1. 重新加载最新状态（乐观锁）
    // 2. 更新 ReleaseApp
    // 3. 标记旧 Deployment 为 superseded
    // 4. 创建新 Deployment
    return nil
})
```

### 3. 不需要取消 K8s 任务

根据现有架构，Deployment 使用状态机管理，不需要主动取消任务。
旧 Deployment 标记为 `is_superseded=true` 后，状态机会继续处理。

### 4. Deployment 标记为 superseded

在 Deployment 模型中添加字段：

```go
type Deployment struct {
    // ... 现有字段
    IsSuperseded bool `gorm:"default:false" json:"is_superseded"`
    SupersededAt *time.Time `json:"superseded_at"`
    SupersededBy *int64 `json:"superseded_by"` // 新 Deployment 的 ID
}
```

### 5. 新 Tag 检测的触发

Webhook 处理流程：
1. 接收 Drone 构建完成通知
2. 查询所有包含该 app 的活跃 batch（status != Completed/Cancelled）
3. 对每个 ReleaseApp，比较 build_id 和 latest_build_id
4. 如果不同，更新 latest_build_id 和 has_new_tag=true

## 📊 工作量估算

| 任务 | 工作量 | 说明 |
|------|--------|------|
| 数据库迁移 | 0.5 天 | 添加 4 个字段 |
| 模型更新 | 0.5 天 | 更新 ReleaseApp 和 Deployment |
| Webhook 处理 | 1 天 | 实现新 tag 检测逻辑 |
| 版本切换逻辑 | 1.5 天 | 实现核心业务逻辑 |
| API 端点 | 1 天 | 添加 2-3 个端点 |
| 状态机集成 | 0.5 天 | 添加 action |
| 测试 | 1.5 天 | 单元测试和集成测试 |
| **总计** | **6.5 天** | 约 1-1.5 周 |

## 🎯 下一步

1. **确认数据库字段** - 是否需要添加其他字段？
2. **确认 Webhook 格式** - Drone 推送的具体格式是什么？
3. **确认 K8s 集成** - 如何取消 K8s 任务？
4. **确认权限控制** - 是否需要添加权限检查？
5. **开始实现** - 从 Step 1 开始

---

**版本：** v1.0  
**最后更新：** 2024-01-15  
**状态：** 待讨论

