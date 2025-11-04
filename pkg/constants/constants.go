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
	ReleaseAppStatusPending        int8 = 0
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
	DeploymentStatusPending = "pending"
	DeploymentStatusRunning = "running"
	DeploymentStatusSuccess = "success"
	DeploymentStatusFailed  = "failed"
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

// 应用类型
const (
	AppTypeWeb    = "static"
	AppTypeNode   = "node"
	appTypeJava   = "java"
	AppTypeGo     = "go"
	APpTypePython = "py"
)

// ValidAppTypes 有效的应用类型列表（用于验证）
var ValidAppTypes = []string{
	AppTypeWeb,
	AppTypeNode,
	appTypeJava,
	AppTypeGo,
	APpTypePython,
}

// IsValidAppType 验证应用类型是否有效
func IsValidAppType(appType string) bool {
	for _, valid := range ValidAppTypes {
		if appType == valid {
			return true
		}
	}
	return false
}

// AppTypeMetadata 应用类型元数据
type AppTypeMetadata struct {
	Value       string
	Label       string
	Description string
	Icon        string
	Color       string
}

// GetAppTypeMetadata 获取应用类型元数据列表
func GetAppTypeMetadata() []AppTypeMetadata {
	return []AppTypeMetadata{
		{
			Value:       AppTypeWeb,
			Label:       "静态网站",
			Description: "纯静态网站，使用Nginx部署",
			Icon:        "html5",
			Color:       "#E34F26",
		},
		{
			Value:       AppTypeNode,
			Label:       "Node.js应用",
			Description: "Node.js/Express/Nest.js等应用",
			Icon:        "nodejs",
			Color:       "#339933",
		},
		{
			Value:       appTypeJava,
			Label:       "Java应用",
			Description: "Spring Boot/Spring Cloud等应用",
			Icon:        "java",
			Color:       "#007396",
		},
		{
			Value:       AppTypeGo,
			Label:       "Go应用",
			Description: "Golang应用",
			Icon:        "go",
			Color:       "#00ADD8",
		},
		{
			Value:       APpTypePython,
			Label:       "Python应用",
			Description: "Python/Django/Flask等应用",
			Icon:        "python",
			Color:       "#3776AB",
		},
	}
}

// GetAppTypeLabel 获取应用类型的显示名称
func GetAppTypeLabel(appType string) string {
	metadata := GetAppTypeMetadata()
	for _, meta := range metadata {
		if meta.Value == appType {
			return meta.Label
		}
	}
	return appType
}

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
