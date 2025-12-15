package transitions

import "devops-cd/internal/model"

type TransitionHandler interface {
	// Handle 检查合法性, 处理强依赖操作
	Handle(batch *model.Batch, from, to int8, options *TransitionOptions) error

	// After 状态转换成功后, 异步操作
	After(batch *model.Batch, from, to int8, options *TransitionOptions)
}

type TransitionHandleFunc func(batch *model.Batch, from, to int8) error

func (h TransitionHandleFunc) Handle(batch *model.Batch, from, to int8) error {
	return h(batch, from, to)
}

func (h TransitionHandleFunc) After(batch *model.Batch, from, to int8) {

}

// 状态流转来源: 内部/外部
const (
	SourceInside  int8 = 1 << 0
	SourceOutside int8 = 1 << 1
)

type StateTransition struct {
	From    int8
	To      int8
	Event   string
	Handler TransitionHandler

	AllowSource int8 // 使用位运算
}

type TransitionOption func(*TransitionOptions)

type TransitionOptions struct {
	operator string
	reason   string
	// data       map[string]interface{}
	SideEffect func(b *model.Batch)
}

func WithModelEffects(sideEffects func(b *model.Batch)) TransitionOption {
	return func(o *TransitionOptions) { o.SideEffect = sideEffects }
}
func WithOperator(operator string) TransitionOption {
	return func(o *TransitionOptions) { o.operator = operator }
}
func WithReason(reason string) TransitionOption {
	return func(o *TransitionOptions) { o.reason = reason }
}
