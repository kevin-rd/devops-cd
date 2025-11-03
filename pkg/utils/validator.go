package utils

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// FormatValidationError 格式化验证错误信息
func FormatValidationError(err error) string {
	if err == nil {
		return ""
	}

	// 处理validator的验证错误
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var messages []string
		for _, e := range validationErrors {
			messages = append(messages, formatFieldError(e))
		}
		return strings.Join(messages, "; ")
	}

	// 处理JSON解析错误
	if jsonErr, ok := err.(*json.UnmarshalTypeError); ok {
		return fmt.Sprintf("field '%s' should be %s", jsonErr.Field, jsonErr.Type.String())
	}

	// 处理JSON语法错误
	if _, ok := err.(*json.SyntaxError); ok {
		return "invalid JSON format"
	}

	// 其他错误直接返回错误信息
	return err.Error()
}

// formatFieldError 格式化单个字段的验证错误
func formatFieldError(e validator.FieldError) string {
	field := e.Field()

	switch e.Tag() {
	case "required":
		return fmt.Sprintf("field '%s' is required", field)
	case "max":
		return fmt.Sprintf("field '%s' must be at most %s characters", field, e.Param())
	case "min":
		return fmt.Sprintf("field '%s' must be at least %s characters", field, e.Param())
	case "oneof":
		return fmt.Sprintf("field '%s' must be one of: %s", field, e.Param())
	case "email":
		return fmt.Sprintf("field '%s' must be a valid email address", field)
	case "url":
		return fmt.Sprintf("field '%s' must be a valid URL", field)
	case "len":
		return fmt.Sprintf("field '%s' must be exactly %s characters", field, e.Param())
	case "gt":
		return fmt.Sprintf("field '%s' must be greater than %s", field, e.Param())
	case "gte":
		return fmt.Sprintf("field '%s' must be greater than or equal to %s", field, e.Param())
	case "lt":
		return fmt.Sprintf("field '%s' must be less than %s", field, e.Param())
	case "lte":
		return fmt.Sprintf("field '%s' must be less than or equal to %s", field, e.Param())
	case "numeric":
		return fmt.Sprintf("field '%s' must be numeric", field)
	case "alphanum":
		return fmt.Sprintf("field '%s' must be alphanumeric", field)
	default:
		return fmt.Sprintf("field '%s' validation failed on '%s' tag", field, e.Tag())
	}
}
