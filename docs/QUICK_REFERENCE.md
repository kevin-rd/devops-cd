# 新 Tag 处理方案 - 快速参考

## 🎯 核心流程

### 新 Tag 检测流程

```
Drone 构建完成
    ↓
推送 Webhook 到 /api/v1/builds/notify
    ↓
查询所有活跃 batch 中的 ReleaseApp
    ↓
更新 latest_build_id 和 has_new_tag=true
    ↓
返回受影响的 ReleaseApp 列表
```

### 版本切换流程

```
用户调用 POST /api/v1/releases/{release_id}/switch-version
    ↓
检查前置条件（Batch 状态、has_new_tag、Deployment 状态）
    ↓
事务处理：
  1. 更新 ReleaseApp（build_id、target_tag、has_new_tag）
  2. 标记旧 Deployment 为 superseded
  3. 创建新 Deployment
    ↓
返回成功
```

## 📊 状态转换规则

### 允许切换版本

```
✓ Batch 状态 = PreWaiting (20) 或 ProdWaiting (30)
✓ ReleaseApp.has_new_tag = true
✓ ReleaseApp.latest_build_id != null
✓ 没有正在运行的 Deployment（status != "running"）
```

### 禁止切换版本

```
✗ Batch 状态 = PreDeploying (21) 或 ProdDeploying (31)
✗ ReleaseApp.has_new_tag = false
✗ 有正在运行的 Deployment（status = "running"）
```

## 🗄️ 数据库变更

### release_apps 表

```sql
ALTER TABLE release_apps ADD COLUMN (
  `latest_build_id` BIGINT,
  `has_new_tag` BOOLEAN DEFAULT FALSE,
  `tag_updated_at` TIMESTAMP NULL,
  `tag_updated_by` VARCHAR(50)
);
```

### deployments 表

```sql
ALTER TABLE deployments ADD COLUMN (
  `is_superseded` BOOLEAN DEFAULT FALSE,
  `superseded_at` TIMESTAMP NULL,
  `superseded_by` BIGINT
);
```

## 📝 模型字段

### ReleaseApp

```go
LatestBuildID *int64     // 最新构建ID
HasNewTag     bool       // 是否有新tag
TagUpdatedAt  *time.Time // 新tag更新时间
TagUpdatedBy  *string    // 新tag更新者
```

### Deployment

```go
IsSuperseded bool       // 是否已被替代
SupersededAt *time.Time // 被替代时间
SupersededBy *int64     // 被哪个Deployment替代
```

## 🔌 API 端点

### 切换版本

```
POST /api/v1/releases/{release_id}/switch-version

请求：
{
  "reason": "切换到新版本"
}

响应：
{
  "code": 0,
  "data": {
    "release_id": 1,
    "old_build_id": 1,
    "new_build_id": 2,
    "old_tag": "v1.2.2",
    "new_tag": "v1.2.3",
    "new_deployment_id": 10
  }
}
```

### 查询状态

```
GET /api/v1/releases/{release_id}/status

响应：
{
  "code": 0,
  "data": {
    "release_id": 1,
    "current_build_id": 1,
    "current_tag": "v1.2.2",
    "latest_build_id": 2,
    "latest_tag": "v1.2.3",
    "has_new_tag": true,
    "can_switch": true
  }
}
```

## 📂 文件修改清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `scripts/005_add_new_tag_support.sql` | 新建 | 数据库迁移 |
| `internal/model/release.go` | 修改 | 添加 4 个字段 |
| `internal/model/deploy.go` | 修改 | 添加 3 个字段 |
| `internal/handler/build_handler.go` | 修改 | 实现 NotifyNewTag |
| `internal/core/release_app/version_switcher.go` | 新建 | 版本切换逻辑 |
| `internal/core/release_app/outside_action.go` | 修改 | 更新 new_tag action |
| `api/handler/release_handler.go` | 修改 | 添加 2 个 API 端点 |

## 🧪 关键测试场景

### 新 Tag 检测

- [ ] Webhook 接收和处理
- [ ] latest_build_id 更新
- [ ] has_new_tag 标记
- [ ] 多个 app 处理

### 版本切换

- [ ] 前置条件检查
- [ ] 事务处理
- [ ] Deployment 创建
- [ ] 旧 Deployment 标记

### 错误处理

- [ ] 无效的 release_id
- [ ] 状态不允许切换
- [ ] 正在运行的 Deployment
- [ ] 事务回滚

## 📋 实现检查清单

### 第 1 阶段：数据库和模型

- [ ] 创建迁移脚本
- [ ] 执行迁移
- [ ] 更新 ReleaseApp 模型
- [ ] 更新 Deployment 模型

### 第 2 阶段：Webhook 处理

- [ ] 实现 NotifyNewTag 方法
- [ ] 添加路由
- [ ] 测试 Webhook 处理

### 第 3 阶段：版本切换逻辑

- [ ] 创建 VersionSwitcher
- [ ] 实现 SwitchVersion 方法
- [ ] 更新 new_tag action

### 第 4 阶段：API 端点

- [ ] 实现 SwitchVersion 端点
- [ ] 实现 GetReleaseStatus 端点
- [ ] 添加路由

### 第 5 阶段：测试

- [ ] 单元测试
- [ ] 集成测试
- [ ] 手动测试

## 🚀 快速开始

1. **阅读文档**
   ```
   READY_TO_IMPLEMENT.md → SIMPLIFIED_IMPLEMENTATION_PLAN.md → IMPLEMENTATION_DETAILS.md
   ```

2. **执行第 1 阶段**
   ```
   创建迁移脚本 → 更新模型
   ```

3. **执行第 2 阶段**
   ```
   实现 Webhook 处理 → 添加路由
   ```

4. **执行第 3 阶段**
   ```
   创建 VersionSwitcher → 实现切换逻辑
   ```

5. **执行第 4 阶段**
   ```
   添加 API 端点 → 添加路由
   ```

6. **执行第 5 阶段**
   ```
   编写测试 → 运行测试
   ```

## 💡 关键提示

- ✅ 使用事务处理确保数据一致性
- ✅ 使用乐观锁防止并发冲突
- ✅ 不需要取消 K8s 任务
- ✅ 状态机会自动处理 Deployment 生命周期
- ✅ 操作应该是幂等的

---

**版本：** v1.0  
**最后更新：** 2024-01-15

