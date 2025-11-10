package deploy

import (
	"context"
	"devops-cd/internal/model"
	"github.com/stretchr/testify/mock"
	"sync"
	"time"
)

// MockDeployer 模拟部署器
type MockDeployer struct {
	mock.Mock

	// 可控行为
	deployDelay  time.Duration // 部署延迟
	checkDelay   time.Duration // 状态轮询延迟
	finalStatus  string        // 最终状态: "success"/"failed"/"running"
	deployError  error         // Deploy 是否返回错误
	checkError   error         // CheckStatus 是否返回错误
	deployCalled int           // Deploy 被调用次数
	checkCalled  map[int64]int // 每个 deployment 的 CheckStatus 调用次数
	mu           sync.Mutex
}

func NewMockDeployer() *MockDeployer {
	return &MockDeployer{
		finalStatus: "success",
		checkCalled: make(map[int64]int),
	}
}

// === 配置方法 ===

func (m *MockDeployer) SetFinalStatus(status string) *MockDeployer {
	m.finalStatus = status
	return m
}

func (m *MockDeployer) SetDeployError(err error) *MockDeployer {
	m.deployError = err
	return m
}

func (m *MockDeployer) SetCheckError(err error) *MockDeployer {
	m.checkError = err
	return m
}

func (m *MockDeployer) SetDeployDelay(d time.Duration) *MockDeployer {
	m.deployDelay = d
	return m
}

func (m *MockDeployer) SetCheckDelay(d time.Duration) *MockDeployer {
	m.checkDelay = d
	return m
}

// === 接口实现 ===

func (m *MockDeployer) Deploy(ctx context.Context, dep *model.Deployment) error {
	m.mu.Lock()
	m.deployCalled++
	m.mu.Unlock()

	if m.deployDelay > 0 {
		select {
		case <-time.After(m.deployDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if m.deployError != nil {
		return m.deployError
	}

	// 模拟 task_id
	taskID := "mock-task-123"
	dep.TaskID = &taskID
	return nil
}

func (m *MockDeployer) CheckStatus(ctx context.Context, dep *model.Deployment) (string, error) {
	if dep.TaskID == nil {
		return "", nil
	}

	m.mu.Lock()
	m.checkCalled[dep.ID]++
	count := m.checkCalled[dep.ID]
	m.mu.Unlock()

	if m.checkDelay > 0 {
		select {
		case <-time.After(m.checkDelay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	if m.checkError != nil {
		return "", m.checkError
	}

	// 模拟状态流转
	switch count {
	case 1:
		return "running", nil
	case 2:
		if m.finalStatus == "running" {
			return "running", nil
		}
		return m.finalStatus, nil
	case 3:
		return "success", nil
	default:
		return m.finalStatus, nil
	}
}

// === 验证方法 ===

func (m *MockDeployer) AssertDeployCalled(t mock.TestingT, times int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deployCalled != times {
		t.Errorf("Deploy called %d times, want %d", m.deployCalled, times)
	}
}

func (m *MockDeployer) AssertCheckCalled(t mock.TestingT, depID int64, times int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.checkCalled[depID] != times {
		t.Errorf("CheckStatus for deployment %d called %d times, want %d", depID, m.checkCalled[depID], times)
	}
}
