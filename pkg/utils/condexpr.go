package utils

func Condexpr(condition bool, trueValue, falseValue interface{}) interface{} {
	if condition {
		return trueValue
	}
	return falseValue
}

func CopyInt8(src int8) *int8 {
	return &src
}
