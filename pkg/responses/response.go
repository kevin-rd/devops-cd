package responses

import (
	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Detail  string      `json:"detail,omitempty"` // 详细错误信息（可选）
	Data    interface{} `json:"data,omitempty"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(200, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(200, Response{
		Code:    CodeSuccess,
		Message: message,
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, err error) {
	if appErr, ok := err.(*AppError); ok {
		// 统一返回HTTP 200，业务错误码在response.code中
		c.JSON(200, Response{
			Code:    appErr.Code,
			Message: appErr.Message,
		})
		return
	}

	// 未知错误也返回HTTP 200
	c.JSON(200, Response{
		Code:    CodeInternalError,
		Message: err.Error(),
	})
}

// ErrorWithCode 自定义错误响应
func ErrorWithCode(c *gin.Context, code int, message string) {
	// 统一返回HTTP 200，业务错误码在response.code中
	c.JSON(200, Response{
		Code:    code,
		Message: message,
	})
}

// ErrorWithDetail 带详细信息的错误响应
func ErrorWithDetail(c *gin.Context, code int, message, detail string) {
	c.JSON(200, Response{
		Code:    code,
		Message: message,
		Detail:  detail,
	})
}
