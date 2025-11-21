package constants

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
