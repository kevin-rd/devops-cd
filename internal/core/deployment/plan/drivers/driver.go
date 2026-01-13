package drivers

import (
	"context"
)

// Stage 表示一次 deployment 的执行阶段（先固定为 pre/main）。
type Stage string

const (
	StagePre  Stage = "pre"
	StageMain Stage = "main"
)

// Status 表示一次 driver 执行的结果状态。
type Status string

const (
	StatusSuccess Status = "success"
	StatusRunning Status = "running"
	StatusFailed  Status = "failed"
)

type ExecuteRequest struct {
	Stage     Stage
	Namespace string

	// Payload 由调用方组装（为了避免 driver 依赖 deployment state machine 的内部细节）
	Payload interface{}
}

type ExecuteResult struct {
	Status  Status
	Message string
}

func Success() *ExecuteResult {
	return &ExecuteResult{
		Status:  StatusSuccess,
		Message: "",
	}
}

func Running(message string) *ExecuteResult {
	return &ExecuteResult{
		Status:  StatusRunning,
		Message: message,
	}
}

func Failed(message string) *ExecuteResult {
	return &ExecuteResult{
		Status:  StatusFailed,
		Message: message,
	}
}

type Driver interface {
	Name() string
	Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResult, error)
	CheckStatus(ctx context.Context, req *ExecuteRequest) (*ExecuteResult, error)
}

type Registry interface {
	Get(driverType string) (Driver, bool)
}

type StaticRegistry map[string]Driver

func (r StaticRegistry) Get(driverType string) (Driver, bool) {
	d, ok := r[driverType]
	return d, ok
}
