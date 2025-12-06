package deploy

import (
	"context"
	"devops-cd/internal/adapter/notification"
	"devops-cd/internal/model"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"time"
)

// DeployConfig 部署配置
type DeployConfig struct {
	ConcurrentApps   int           // 并发部署应用数
	SingleAppTimeout time.Duration // 单应用超时时间
	BatchTimeout     time.Duration // 批次超时时间
	RetryCount       int           // 重试次数
	RetryBackoff     string        // 重试策略: exponential/linear
	PollInterval     time.Duration // 状态轮询间隔
}

// DefaultDeployConfig 默认部署配置
func DefaultDeployConfig() *DeployConfig {
	return &DeployConfig{
		ConcurrentApps:   5,
		SingleAppTimeout: 10 * time.Minute,
		BatchTimeout:     60 * time.Minute,
		RetryCount:       3,
		RetryBackoff:     "exponential",
		PollInterval:     5 * time.Second,
	}
}

// K8sDeployer 实现 Deployer 接口
type K8sDeployer struct {
	k8sClient K8sDeployClient
	notifier  notification.Notifier
	db        *gorm.DB
	config    *DeployConfig
	logger    *zap.Logger
}

func NewK8sDeployer(k8sClient K8sDeployClient, notifier notification.Notifier, db *gorm.DB, config *DeployConfig, logger *zap.Logger) *K8sDeployer {
	if config == nil {
		config = DefaultDeployConfig()
	}
	return &K8sDeployer{
		k8sClient: k8sClient,
		notifier:  notifier,
		db:        db,
		config:    config,
		logger:    logger,
	}
}

// Deploy 触发部署（幂等）
func (d *K8sDeployer) Deploy(ctx context.Context, dep *model.Deployment) error {
	log := d.logger.With(
		zap.Int64("deployment_id", dep.ID),
		zap.Int64("app_id", dep.AppID),
		zap.String("env", dep.Env),
		zap.String("cluster", dep.ClusterName),
	)

	// 1. 检查是否已有 task_id（幂等）
	if dep.TaskID != nil && *dep.TaskID != "" {
		log.Info("部署任务已存在，跳过重复触发", zap.String("task_id", *dep.TaskID))
		return nil
	}

	// 2. 加载 ReleaseApp 获取镜像
	imageURL := ""
	imageTag := ""
	if imageURL == "" || imageTag == "" {
		return fmt.Errorf("镜像信息为空")
	}

	// 3. 触发 K8s 部署
	appCtx, cancel := context.WithTimeout(ctx, d.config.SingleAppTimeout)
	defer cancel()

	taskID, err := d.k8sClient.DeployApp(appCtx, dep.Env, dep.AppID, imageURL, imageTag)
	if err != nil {
		if d.notifier != nil {
			_ = d.notifier.SendAppDeployNotification(appCtx, int64(dep.BatchID), int64(dep.AppID),
				fmt.Sprintf("App-%d", dep.AppID), notification.NotifyAppDeployFailed,
				fmt.Sprintf("触发部署失败: %s", err.Error()))
		}
		return err
	}

	// 4. 更新 task_id（状态机外）
	now := time.Now()
	if err := d.db.Model(dep).Updates(map[string]any{
		"task_id":    taskID,
		"image_url":  imageURL,
		"image_tag":  imageTag,
		"started_at": &now,
	}).Error; err != nil {
		log.Error("更新 task_id 失败", zap.Error(err))
	}

	log.Info("部署任务已触发", zap.String("task_id", taskID))
	return nil
}

// CheckStatus 检查部署状态
func (d *K8sDeployer) CheckStatus(ctx context.Context, dep *model.Deployment) (string, error) {
	if dep.TaskID == nil || *dep.TaskID == "" {
		return "", fmt.Errorf("task_id 为空")
	}

	status, err := d.k8sClient.CheckDeployStatus(ctx, *dep.TaskID)
	if err != nil {
		return "", err
	}

	d.logger.Debug("K8s 部署状态",
		zap.Int64("deployment_id", dep.ID),
		zap.String("task_id", *dep.TaskID),
		zap.String("status", status.Status),
		zap.Int("progress", status.Progress))

	switch status.Status {
	case "success":
		return "success", nil
	case "failed":
		return "failed", fmt.Errorf("k8s 部署失败: %s", status.ErrorMsg)
	default:
		return "running", nil
	}
}

// K8sDeployClient K8s部署客户端接口
// 用于与底层K8s部署服务交互
type K8sDeployClient interface {
	// DeployApp 触发单个应用部署
	// env: 环境(pre/prod)
	// appID: 应用ID
	// imageName: 镜像名称
	// imageTag: 镜像标签
	// 返回: 部署任务ID和错误
	DeployApp(ctx context.Context, env string, appID int64, imageName string, imageTag string) (string, error)

	// CheckDeployStatus 检查部署状态
	// taskID: 部署任务ID
	// 返回: 部署状态和错误
	CheckDeployStatus(ctx context.Context, taskID string) (*DeployStatus, error)

	// GetPodStatus 获取Pod状态
	// env: 环境
	// appID: 应用ID
	// 返回: Pod状态列表和错误
	GetPodStatus(ctx context.Context, env string, appID int64) ([]*PodStatus, error)
}

// DeployStatus 部署状态
type DeployStatus struct {
	TaskID      string     `json:"task_id"`
	Status      string     `json:"status"`       // pending/running/success/failed
	Message     string     `json:"message"`      // 状态消息
	StartedAt   time.Time  `json:"started_at"`   // 开始时间
	FinishedAt  *time.Time `json:"finished_at"`  // 完成时间
	Progress    int        `json:"progress"`     // 进度百分比(0-100)
	PodsReady   int        `json:"pods_ready"`   // 就绪Pod数
	PodsDesired int        `json:"pods_desired"` // 期望Pod数
	ErrorMsg    string     `json:"error_msg"`    // 错误信息
}

// PodStatus Pod状态
type PodStatus struct {
	Name      string    `json:"name"`
	Phase     string    `json:"phase"` // Pending/Running/Succeeded/Failed/Unknown
	Ready     bool      `json:"ready"` // 是否就绪
	StartedAt time.Time `json:"started_at"`
	Message   string    `json:"message"`
}

// MockK8sDeployClient 模拟K8s部署客户端(用于测试)
type MockK8sDeployClient struct {
	// 模拟延迟
	deployDelay time.Duration
}

// NewMockK8sDeployClient 创建模拟客户端
func NewMockK8sDeployClient() *MockK8sDeployClient {
	return &MockK8sDeployClient{
		deployDelay: 5 * time.Second, // 模拟5秒部署时间
	}
}

// DeployApp 模拟部署应用
func (c *MockK8sDeployClient) DeployApp(ctx context.Context, env string, appID int64, imageName string, imageTag string) (string, error) {
	// 生成模拟任务ID
	taskID := time.Now().Format("20060102150405") + "-" + env + "-" + string(rune(appID))
	return taskID, nil
}

// CheckDeployStatus 模拟检查部署状态
func (c *MockK8sDeployClient) CheckDeployStatus(ctx context.Context, taskID string) (*DeployStatus, error) {
	// 模拟返回成功状态
	now := time.Now()
	return &DeployStatus{
		TaskID:      taskID,
		Status:      "success",
		Message:     "部署成功",
		StartedAt:   now.Add(-c.deployDelay),
		FinishedAt:  &now,
		Progress:    100,
		PodsReady:   3,
		PodsDesired: 3,
	}, nil
}

// GetPodStatus 模拟获取Pod状态
func (c *MockK8sDeployClient) GetPodStatus(ctx context.Context, env string, appID int64) ([]*PodStatus, error) {
	// 模拟返回3个就绪的Pod
	now := time.Now()
	pods := []*PodStatus{
		{
			Name:      "app-pod-1",
			Phase:     "Running",
			Ready:     true,
			StartedAt: now.Add(-5 * time.Minute),
			Message:   "Pod is running",
		},
		{
			Name:      "app-pod-2",
			Phase:     "Running",
			Ready:     true,
			StartedAt: now.Add(-5 * time.Minute),
			Message:   "Pod is running",
		},
		{
			Name:      "app-pod-3",
			Phase:     "Running",
			Ready:     true,
			StartedAt: now.Add(-5 * time.Minute),
			Message:   "Pod is running",
		},
	}
	return pods, nil
}

// RealK8sDeployClient 真实的K8s部署客户端(待实现)
// TODO: 实现与实际K8s部署服务的HTTP/gRPC调用
type RealK8sDeployClient struct {
	baseURL string // 部署服务的基础URL
	apiKey  string // API密钥
}

// NewRealK8sDeployClient 创建真实客户端
func NewRealK8sDeployClient(baseURL, apiKey string) *RealK8sDeployClient {
	return &RealK8sDeployClient{
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

// DeployApp 真实部署应用(待实现)
func (c *RealK8sDeployClient) DeployApp(ctx context.Context, env string, appID int64, imageName string, imageTag string) (string, error) {
	// TODO: 实现HTTP/gRPC调用
	// POST /api/v1/deploy
	// Body: {"env": env, "app_id": appID, "image": imageName, "tag": imageTag}
	return "", nil
}

// CheckDeployStatus 真实检查部署状态(待实现)
func (c *RealK8sDeployClient) CheckDeployStatus(ctx context.Context, taskID string) (*DeployStatus, error) {
	// TODO: 实现HTTP/gRPC调用
	// GET /api/v1/deploy/{taskID}/status
	return nil, nil
}

// GetPodStatus 真实获取Pod状态(待实现)
func (c *RealK8sDeployClient) GetPodStatus(ctx context.Context, env string, appID int64) ([]*PodStatus, error) {
	// TODO: 实现HTTP/gRPC调用
	// GET /api/v1/pods?env={env}&app_id={appID}
	return nil, nil
}
