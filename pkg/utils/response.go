package utils

import (
	"devops-cd/pkg/errors"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Detail  string      `json:"detail,omitempty"` // 详细错误信息（可选）
	Data    interface{} `json:"data,omitempty"`
}

// PageResponse 分页响应结构
type PageResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Total   int64       `json:"total"`
	Page    int         `json:"page"`
	Size    int         `json:"size"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(200, Response{
		Code:    errors.CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(200, Response{
		Code:    errors.CodeSuccess,
		Message: message,
		Data:    data,
	})
}

// PageSuccess 分页成功响应
func PageSuccess(c *gin.Context, data interface{}, total int64, page, size int) {
	c.JSON(200, PageResponse{
		Code:    errors.CodeSuccess,
		Message: "success",
		Data:    data,
		Total:   total,
		Page:    page,
		Size:    size,
	})
}

// Error 错误响应
func Error(c *gin.Context, err error) {
	if appErr, ok := err.(*errors.AppError); ok {
		// 统一返回HTTP 200，业务错误码在response.code中
		c.JSON(200, Response{
			Code:    appErr.Code,
			Message: appErr.Message,
		})
		return
	}

	// 未知错误也返回HTTP 200
	c.JSON(200, Response{
		Code:    errors.CodeInternalError,
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
