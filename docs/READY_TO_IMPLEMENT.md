## ✅ 方案确认

基于与你的讨论，新 Tag 处理方案已经确认，现在准备开始实现。

### 核心需求

1. ✅ Drone 构建完成后推送新 tag 信息
2. ✅ 在特定条件下允许用户切换版本
3. ✅ PreDeploying 状态下用户可见新 tag 但不能切换
4. ✅ 每个 ReleaseApp 独立控制版本
5. ✅ 任何有权限的用户都可以切换
6. ✅ 使用状态机管理，操作幂等
7. ✅ 不需要 TagUpdateHistory 表
8. ✅ 不需要通知和告警机制
9. ✅ 不需要版本对比功能
10. ✅ 不需要取消 K8s 任务

## 📋 实现清单

### 第一阶段：数据库和模型（1 天）

- [ ] 创建数据库迁移脚本 `scripts/005_add_new_tag_support.sql`
  - 为 `release_apps` 表添加 4 个字段
  - 为 `deployments` 表添加 3 个字段
  - 添加必要的索引

- [ ] 更新 `internal/model/release.go`
  - 为 `ReleaseApp` 添加 4 个新字段

- [ ] 更新 `internal/model/deploy.go`
  - 为 `Deployment` 添加 3 个新字段

### 第二阶段：Webhook 处理（1 天）

- [ ] 在 `internal/handler/build_handler.go` 中实现 `NotifyNewTag` 方法
  - 接收 Drone Webhook
  - 查询所有活跃 batch 中的 ReleaseApp
  - 更新 `latest_build_id` 和 `has_new_tag`

- [ ] 添加路由 `POST /api/v1/builds/notify`

### 第三阶段：版本切换逻辑（1.5 天）

- [ ] 创建 `internal/core/release_app/version_switcher.go`
  - 实现 `SwitchVersion` 方法
  - 检查前置条件
  - 事务处理

- [ ] 在 `internal/core/release_app/outside_action.go` 中更新 `new_tag` action

### 第四阶段：API 端点（1 天）

- [ ] 在 `api/handler/release_handler.go` 中添加：
  - `SwitchVersion` - 切换版本
  - `GetReleaseStatus` - 查询状态

- [ ] 添加路由：
  - `POST /api/v1/releases/{release_id}/switch-version`
  - `GET /api/v1/releases/{release_id}/status`

### 第五阶段：测试（1.5 天）

- [ ] 单元测试
  - 新 Tag 检测
  - 版本切换
  - 错误处理

- [ ] 集成测试
  - 完整流程
  - 并发场景

## 🎯 关键实现点

### 1. 前置条件检查

```
✓ ReleaseApp 存在
✓ has_new_tag = true
✓ latest_build_id != null
✓ Batch 状态 = PreWaiting (20) 或 ProdWaiting (30)
✓ 没有正在运行的 Deployment（status != "running"）
```

### 2. 事务处理

```
1. 重新加载 ReleaseApp（乐观锁）
2. 获取新 Build 信息
3. 更新 ReleaseApp
4. 标记旧 Deployment 为 superseded
5. 创建新 Deployment
```

### 3. 状态机集成

- Deployment 使用状态机管理，不需要主动取消任务
- 旧 Deployment 标记为 `is_superseded=true` 后，状态机会继续处理
- 新 Deployment 创建后，状态机会自动处理后续流程

## 📊 工作量估算

| 阶段 | 任务 | 工作量 |
|------|------|--------|
| 1 | 数据库和模型 | 1 天 |
| 2 | Webhook 处理 | 1 天 |
| 3 | 版本切换逻辑 | 1.5 天 |
| 4 | API 端点 | 1 天 |
| 5 | 测试 | 1.5 天 |
| **总计** | | **6 天** |

## 📚 参考文档

- `SIMPLIFIED_IMPLEMENTATION_PLAN.md` - 简化的实现计划
- `IMPLEMENTATION_DETAILS.md` - 详细的实现细节
- `DISCUSSION_CHECKLIST.md` - 讨论清单

## 🚀 下一步

1. **确认开始** - 是否准备开始实现？
2. **分配资源** - 谁负责哪个阶段？
3. **制定时间表** - 什么时候开始？
4. **开始实现** - 从第一阶段开始

## 📞 问题和讨论

如果在实现过程中有任何问题或需要调整，请随时讨论。

---

**版本：** v1.0  
**最后更新：** 2024-01-15  
**状态：** 准备就绪

