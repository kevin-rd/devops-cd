package responses

import "fmt"

// 错误码
const (
	CodeSuccess         = 2000000
	CodePartialSuccess  = 206 // 部分成功
	CodeBadRequest      = 4000000
	CodeUnauthorized    = 4010000
	CodeForbidden       = 4030000
	CodeNotFound        = 4040000
	CodeConflict        = 4009000
	CodeInternalError   = 5000000
	CodeDatabaseError   = 5001000
	CodeAuthError       = 5002000
	CodeValidationError = 5003000
)

// AppError 应用错误
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// New 创建新错误
func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装错误
func Wrap(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// 预定义错误
var (
	ErrBadRequest      = New(CodeBadRequest, "请求参数错误")
	ErrUnauthorized    = New(CodeUnauthorized, "未授权")
	ErrForbidden       = New(CodeForbidden, "禁止访问")
	ErrNotFound        = New(CodeNotFound, "资源不存在")
	ErrConflict        = New(CodeConflict, "资源冲突")
	ErrInternalError   = New(CodeInternalError, "内部服务器错误")
	ErrDatabaseError   = New(CodeDatabaseError, "数据库错误")
	ErrAuthError       = New(CodeAuthError, "认证失败")
	ErrValidationError = New(CodeValidationError, "数据验证失败")

	ErrInvalidParams        = New(CodeBadRequest, "请求参数错误")
	ErrInvalidCredentials   = New(CodeAuthError, "用户名或密码错误")
	ErrLDAPConnectionFailed = New(CodeAuthError, "LDAP连接失败")
	ErrUserNotFound         = New(CodeNotFound, "用户不存在")
	ErrUserDisabled         = New(CodeForbidden, "用户已禁用")
	ErrInvalidToken         = New(CodeUnauthorized, "无效的Token")
	ErrTokenExpired         = New(CodeUnauthorized, "Token已过期")
	ErrRecordNotFound       = New(CodeNotFound, "记录不存在")
	ErrRecordExists         = New(CodeConflict, "记录已存在")
)
