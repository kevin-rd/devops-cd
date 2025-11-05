package constants

import "fmt"

// BatchStatus 批次状态
const (
	BatchStatusDraft         int8 = 0  // 草稿/准备中
	BatchStatusSealed        int8 = 10 // 已封板
	BatchStatusPreWaiting    int8 = 20 // 预发布待触发中
	BatchStatusPreDeploying  int8 = 21 // 预发布部署中
	BatchStatusPreDeployed   int8 = 22 // 预发布已部署完成, 验收中
	BatchStatusProdWaiting   int8 = 30 // 生产待触发中
	BatchStatusProdDeploying int8 = 31 // 生产部署中
	BatchStatusProdDeployed  int8 = 32 // 生产已部署完成, 验收中
	BatchStatusCompleted     int8 = 40 // 已完成
	BatchStatusFinalAccepted int8 = 40
	BatchStatusCancelled     int8 = 90 // 已取消
)

// int8 → string
var batchStatusName = map[int8]string{
	BatchStatusDraft:         "Draft",
	BatchStatusSealed:        "Sealed",
	BatchStatusPreWaiting:    "PreWaiting",
	BatchStatusPreDeploying:  "PreDeploying",
	BatchStatusPreDeployed:   "PreDeployed",
	BatchStatusProdWaiting:   "ProdWaiting",
	BatchStatusProdDeploying: "ProdDeploying",
	BatchStatusProdDeployed:  "ProdDeployed",
	BatchStatusCompleted:     "Completed",
	BatchStatusCancelled:     "Cancelled",
}

// BatchStatusToString int8 → string
func BatchStatusToString(status int8) string {
	if name, ok := batchStatusName[status]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(%d)", status)
}

// ReleaseAppStatus 应用发布状态
const (
	ReleaseAppStatusPending        int8 = 0  // 初始化
	ReleaseAppStatusTagged         int8 = 1  // 应用已打tag并发版成功
	ReleaseAppStatusPreWaiting     int8 = 10 // Pre 等待被触发
	ReleaseAppStatusPreCanTrigger  int8 = 11 // Pre 可以触发
	ReleaseAppStatusPreTriggered   int8 = 12 // Pre 均已触发
	ReleaseAppStatusPreDeployed    int8 = 13 // Pre 均部署完成
	ReleaseAppStatusPreFailed      int8 = 14
	ReleaseAppStatusProdWaiting    int8 = 20 // Prod 等待被触发
	ReleaseAppStatusProdCanTrigger int8 = 21 // Prod 可以触发
	ReleaseAppStatusProdTriggered  int8 = 22 // Prod 均已触发
	ReleaseAppStatusProdDeployed   int8 = 23 // Prod 均部署完成
	ReleaseAppStatusProdFailed     int8 = 24
)

// DeploymentStatus 部署状态
const (
	DeploymentStatusPending             = "pending"
	DeploymentStatusRunning             = "running"
	DeploymentStatusSuccess             = "success"
	DeploymentStatusFailed              = "failed"
	DeploymentStatusWaitingDependencies = "waiting_dependencies"
)

const (
	BuildStatusSuccess = "success"
)

// ApprovalStatus 审批状态（独立于部署流程）
const (
	ApprovalStatusPending  = "pending"  // 待审批
	ApprovalStatusApproved = "approved" // 已通过
	ApprovalStatusRejected = "rejected" // 已拒绝
	ApprovalStatusSkipped  = "skipped"  // 跳过审批（可选功能）
)

// 认证类型
const (
	AuthTypeLDAP  = "ldap"
	AuthTypeLocal = "local"
)

// 状态
const (
	StatusEnabled  int8 = 1
	StatusDisabled int8 = 0
)

// Git 类型
const (
	GitTypeGitea  = "gitea"
	GitTypeGitLab = "gitlab"
	GitTypeGitHub = "github"
)

// 应用类型, 使用配置文件查看
const (
	AppTypeWeb    = "static"
	AppTypeNode   = "node"
	appTypeJava   = "java"
	AppTypeGo     = "go"
	AppTypePython = "py"
)

// 环境类型
const (
	EnvTypeDev  = "dev"
	EnvTypeTest = "test"
	EnvTypePre  = "pre"
	EnvTypeProd = "prod"
)

// JWT 相关
const (
	JWTContextKey  = "jwt_user"
	JWTTypeAccess  = "access"
	JWTTypeRefresh = "refresh"
)

// HTTP Header
const (
	HeaderAuthorization = "Authorization"
	HeaderBearerPrefix  = "Bearer "
)
