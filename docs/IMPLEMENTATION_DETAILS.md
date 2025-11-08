# 新 Tag 处理方案 - 实现细节

## 📍 文件修改清单

### 1. 数据库迁移

**文件：** `scripts/005_add_new_tag_support.sql`

```sql
-- 为 release_apps 表添加新字段
ALTER TABLE release_apps ADD COLUMN (
  `latest_build_id` BIGINT COMMENT '最新检测到的构建ID',
  `has_new_tag` BOOLEAN DEFAULT FALSE COMMENT '是否有新tag待处理',
  `tag_updated_at` TIMESTAMP NULL COMMENT '新tag更新时间',
  `tag_updated_by` VARCHAR(50) COMMENT '新tag更新者'
);

-- 为 deployments 表添加新字段
ALTER TABLE deployments ADD COLUMN (
  `is_superseded` BOOLEAN DEFAULT FALSE COMMENT '是否已被新版本替代',
  `superseded_at` TIMESTAMP NULL COMMENT '被替代时间',
  `superseded_by` BIGINT COMMENT '被哪个Deployment替代'
);

-- 添加索引
ALTER TABLE release_apps ADD INDEX idx_has_new_tag (has_new_tag);
ALTER TABLE deployments ADD INDEX idx_is_superseded (is_superseded);
```

### 2. 模型更新

**文件：** `internal/model/release.go`

在 `ReleaseApp` 结构体中添加：

```go
LatestBuildID *int64     `gorm:"column:latest_build_id" json:"latest_build_id"`
HasNewTag     bool       `gorm:"default:false" json:"has_new_tag"`
TagUpdatedAt  *time.Time `gorm:"column:tag_updated_at" json:"tag_updated_at"`
TagUpdatedBy  *string    `gorm:"column:tag_updated_by;size:50" json:"tag_updated_by"`
```

**文件：** `internal/model/deploy.go`

在 `Deployment` 结构体中添加：

```go
IsSuperseded bool       `gorm:"default:false" json:"is_superseded"`
SupersededAt *time.Time `gorm:"column:superseded_at" json:"superseded_at"`
SupersededBy *int64     `gorm:"column:superseded_by" json:"superseded_by"`
```

### 3. Webhook 处理

**文件：** `internal/handler/build_handler.go`

添加新方法 `NotifyNewTag`：

```go
func (h *BuildHandler) NotifyNewTag(ctx context.Context, req *dto.BuildNotifyRequest) error {
    // 1. 遍历 apps
    // 2. 对每个 app，查询所有活跃 batch 中的 ReleaseApp
    // 3. 更新 latest_build_id 和 has_new_tag
    // 4. 返回受影响的 ReleaseApp 列表
}
```

### 4. 版本切换逻辑

**文件：** `internal/core/release_app/version_switcher.go` (新建)

```go
type VersionSwitcher struct {
    db     *gorm.DB
    logger *zap.Logger
}

func (vs *VersionSwitcher) SwitchVersion(ctx context.Context, releaseID int64, operator string) error {
    // 1. 检查前置条件
    // 2. 事务处理：
    //    - 更新 ReleaseApp
    //    - 标记旧 Deployment 为 superseded
    //    - 创建新 Deployment
}
```

### 5. API 端点

**文件：** `api/handler/release_handler.go`

添加两个新方法：

```go
// SwitchVersion 切换版本
func (h *ReleaseHandler) SwitchVersion(c *gin.Context) {
    // POST /api/v1/releases/{release_id}/switch-version
}

// GetReleaseStatus 查询状态
func (h *ReleaseHandler) GetReleaseStatus(c *gin.Context) {
    // GET /api/v1/releases/{release_id}/status
}
```

### 6. 状态机 Action

**文件：** `internal/core/release_app/outside_action.go`

更新 `new_tag` action：

```go
"new_tag": {
    Handle: func(releaseId int64) error {
        // 验证 latest_build_id 是否存在
        return nil
    },
    Update: func(release *model.ReleaseApp, operator, reason string) {
        // 不改变状态，只更新字段
    },
},
```

## 🔍 关键检查点

### 前置条件检查

```go
// 1. ReleaseApp 存在
// 2. has_new_tag = true
// 3. latest_build_id != null
// 4. Batch 状态 = PreWaiting (20) 或 ProdWaiting (30)
// 5. 没有正在运行的 Deployment（status != "running"）
```

### 事务处理步骤

```go
tx.Transaction(func(tx *gorm.DB) error {
    // 1. 重新加载 ReleaseApp（乐观锁）
    // 2. 获取新 Build 信息
    // 3. 更新 ReleaseApp：
    //    - build_id = latest_build_id
    //    - target_tag = new_build.image_tag
    //    - has_new_tag = false
    //    - latest_build_id = null
    // 4. 查询旧 Deployment（status != success/failed）
    // 5. 标记为 superseded
    // 6. 创建新 Deployment
    return nil
})
```

## 📊 状态转换矩阵

| 当前状态 | 操作 | 新状态 | 说明 |
|---------|------|--------|------|
| PreWaiting | 有新 tag | PreWaiting | 只更新字段，不改变状态 |
| PreWaiting | 切换版本 | PreWaiting | 更新 build_id，创建新 Deployment |
| PreDeploying | 有新 tag | PreDeploying | 用户可见但不能切换 |
| ProdWaiting | 有新 tag | ProdWaiting | 只更新字段，不改变状态 |
| ProdWaiting | 切换版本 | ProdWaiting | 更新 build_id，创建新 Deployment |
| ProdDeploying | 有新 tag | ProdDeploying | 用户可见但不能切换 |

## 🧪 测试场景

### 单元测试

1. **新 Tag 检测**
   - 测试 Webhook 处理
   - 测试 latest_build_id 更新
   - 测试 has_new_tag 标记

2. **版本切换**
   - 测试前置条件检查
   - 测试事务处理
   - 测试 Deployment 创建

3. **错误处理**
   - 测试无效的 release_id
   - 测试状态不允许切换
   - 测试事务回滚

### 集成测试

1. **完整流程**
   - 创建 Batch
   - 封板
   - 触发预发布
   - 接收新 tag
   - 切换版本
   - 验证新 Deployment 创建

2. **并发场景**
   - 多个用户同时切换版本
   - 验证幂等性

## 📝 API 文档

### 切换版本

```
POST /api/v1/releases/{release_id}/switch-version

请求体：
{
  "reason": "切换到新版本"
}

响应：
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

### 查询状态

```
GET /api/v1/releases/{release_id}/status

响应：
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

---

**版本：** v1.0  
**最后更新：** 2024-01-15  
**状态：** 待实现

