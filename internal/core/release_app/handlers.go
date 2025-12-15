package release_app

import (
	"context"
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
	"go.uber.org/zap"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

type Handler interface {
	Handle(ctx context.Context, release *model.ReleaseApp) (nextStatus *int8, updateFunc func(*model.ReleaseApp), err error)
	Name() string
}

type HandlerFunc func(ctx context.Context, release *model.ReleaseApp) (int8, func(*model.ReleaseApp), error)

func (h HandlerFunc) Handle(ctx context.Context, release *model.ReleaseApp) (nextStatus *int8, updateFunc func(*model.ReleaseApp), err error) {
	var status int8
	status, updateFunc, err = h(ctx, release)
	if status != 0 {
		nextStatus = &status
	}
	return
}

func (h HandlerFunc) Name() string {
	if h == nil {
		return "<nil>"
	}

	// 拿到函数的程序计数器
	pc := reflect.ValueOf(h).Pointer()
	if pc == 0 {
		return "<zero>"
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "<unknown>"
	}

	fullName := fn.Name()

	// case 1: 方法值
	if strings.Contains(fullName, "-fm") {
		// 去掉包名和类型，只保留方法名
		if idx := strings.LastIndex(fullName, "."); idx != -1 {
			name := fullName[idx+1:]
			// 去掉 Go 自动加的 -fm 后缀
			if dash := strings.Index(name, "-fm"); dash != -1 {
				name = name[:dash]
			}
			// 可选：去掉公共的 "Handle" 前缀，让日志更简洁
			return strings.TrimPrefix(name, "Handle")
		}
	}

	// case 2: 匿名函数
	if strings.Contains(fullName, "func") {
		// 尝试从文件名+行号提取有意义的名字
		_, file, line, ok := runtime.Caller(3) // 上上上层调用者
		if ok {
			fileName := filepath.Base(file)
			return fmt.Sprintf("%s:%d", fileName, line)
		}
	}

	return fullName
}

func (sm *ReleaseStateMachine) registerHandlers() {
	// Pre
	sm.handlers[constants.ReleaseAppStatusPreWaiting] = HandlerFunc(sm.HandlePreWaiting)
	sm.handlers[constants.ReleaseAppStatusPreCanTrigger] = HandlerFunc(sm.HandlePreCanTrigger)
	sm.handlers[constants.ReleaseAppStatusPreTriggered] = HandlerFunc(sm.HandlePreTriggered)
	sm.handlers[constants.ReleaseAppStatusPreDeployed] = HandlerFunc(sm.HandlePreDeployed)
	sm.handlers[constants.ReleaseAppStatusPreFailed] = HandlerFunc(sm.HandleEmpty)

	// Prod
	sm.handlers[constants.ReleaseAppStatusProdWaiting] = HandlerFunc(sm.HandleProdWaiting)
	sm.handlers[constants.ReleaseAppStatusProdCanTrigger] = HandlerFunc(sm.HandleProdCanTrigger)
	sm.handlers[constants.ReleaseAppStatusProdTriggered] = HandlerFunc(sm.HandleProdTriggered)
	sm.handlers[constants.ReleaseAppStatusProdDeployed] = HandlerFunc(sm.HandleProdDeployed)
	sm.handlers[constants.ReleaseAppStatusProdFailed] = HandlerFunc(sm.HandleEmpty)
}

// handlers

// HandlePreWaiting handle PreWaiting:10 -> PreCanTrigger:11
func (sm *ReleaseStateMachine) HandlePreWaiting(ctx context.Context, release *model.ReleaseApp) (int8, func(*model.ReleaseApp), error) {
	if sm.resolver == nil {
		return constants.ReleaseAppStatusPreCanTrigger, nil, nil
	}

	result, err := sm.resolver.CheckRelease(ctx, release, constants.EnvTypePre)
	if err != nil {
		return 0, nil, err
	}

	if result.HasFailed() {
		reason := result.Summary()
		if reason == "" {
			reason = "依赖检查失败"
		}
		sm.logger.Warn("预发布依赖失败", zap.Int64("release_id", release.ID), zap.String("reason", reason))
		return constants.ReleaseAppStatusPreFailed, func(r *model.ReleaseApp) {
			r.Reason = reason
		}, nil
	}

	if result.HasPending() {
		reason := result.Summary()
		sm.logger.Debug("预发布依赖等待", zap.Int64("release_id", release.ID), zap.String("reason", reason))
		return 0, func(r *model.ReleaseApp) {
			r.Reason = reason
		}, nil
	}

	return constants.ReleaseAppStatusPreCanTrigger, func(r *model.ReleaseApp) {
		r.Reason = ""
	}, nil
}

// HandlePreCanTrigger handle PreCanTrigger:11 -> PreTriggered:12, gen deployments record
func (sm *ReleaseStateMachine) HandlePreCanTrigger(ctx context.Context, release *model.ReleaseApp) (int8, func(*model.ReleaseApp), error) {
	log := sm.logger.With(zap.Int64("release_id", release.ID))

	// 1. 校验 Build
	if release.BuildID == nil {
		return 0, nil, fmt.Errorf("build_id 为空")
	}
	var build model.Build
	if err := sm.db.First(&build, release.BuildID).Error; err != nil {
		return 0, nil, fmt.Errorf("build record not found: %w", err)
	}
	if build.BuildStatus != constants.BuildStatusSuccess {
		return 0, nil, fmt.Errorf("build status: %v", build.BuildStatus)
	}

	// 2. 加载 App
	var app model.Application
	if err := sm.db.Preload("Project").First(&app, release.AppID).Error; err != nil {
		return 0, nil, fmt.Errorf("app record not found: %w", err)
	}

	// 3. 查询 Pre 环境的集群配置
	var configs []model.AppEnvConfig
	if err := sm.db.Where("app_id = ? AND env = ? AND status = 1", release.AppID, constants.EnvTypePre).
		Find(&configs).Error; err != nil {
		return 0, nil, fmt.Errorf("查询 Pre 环境配置失败: %w", err)
	}
	if len(configs) == 0 {
		return 0, nil, fmt.Errorf("应用未配置 Pre 环境")
	}

	// 3.1 查询project级配置
	var projectConfigs model.ProjectEnvConfig
	if err := sm.db.Where("project_id = ? AND env = ?", app.ProjectID, constants.EnvTypePre).First(&projectConfigs).Error; err != nil {
		return 0, nil, fmt.Errorf("查询项目 Pre 环境配置失败: %w", err)
	}

	// 4. 为每个集群创建 Deployment
	// 计算Deployment Name
	// 为Helm计算values.yaml
	var failed []string
	for _, config := range configs {
		if projectConfigs.Namespace == "" {
			return 0, nil, fmt.Errorf("项目 Pre 环境配置未配置 namespace")
		}

		deploymentName, err := ParseDeploymentName(&app, &projectConfigs, &config)
		if err != nil {
			return 0, nil, fmt.Errorf("计算 Deployment Name 失败: %w", err)
		}

		values, err := ParseValues(&app, &build, &projectConfigs, &config)
		if err != nil {
			return 0, nil, fmt.Errorf("计算 values.yaml 失败: %w", err)
		}

		dep := model.Deployment{
			BatchID:   release.BatchID,
			AppID:     release.AppID,
			ReleaseID: release.ID,

			Env:            constants.EnvTypePre,
			ClusterName:    config.Cluster,
			Namespace:      projectConfigs.Namespace,
			DeploymentName: deploymentName,
			Values:         values,

			Status:     "pending",
			RetryCount: 0,
		}

		// 正确使用 FirstOrCreate
		result := sm.db.Where("release_id = ? AND env = ? AND cluster = ?", release.ID, constants.EnvTypePre, config.Cluster).FirstOrCreate(&dep)

		if result.Error != nil {
			failed = append(failed, config.Cluster)
			log.Error("创建 Deployment 失败", zap.String("cluster", config.Cluster), zap.Error(result.Error))
		}
	}

	// 5. 记录失败信息
	if len(failed) > 0 {
		release.Reason = fmt.Sprintf("failed clusters: %v", failed)
		return 0, nil, fmt.Errorf("部分集群创建失败")
	}

	log.Info(fmt.Sprintf("PreDeploy 触发成功,创建了 %d 个集群的 Deployment", len(configs)), zap.String("image", build.ImageTag))
	return constants.ReleaseAppStatusPreTriggered, nil, nil
}

// HandlePreTriggered handle PreTriggered:12 -> PreDeployed:13, check deployments record
func (sm *ReleaseStateMachine) HandlePreTriggered(ctx context.Context, release *model.ReleaseApp) (int8, func(*model.ReleaseApp), error) {
	log := sm.logger.With(zap.Int64("release_id", release.ID), zap.Int64("batch_id", release.BatchID))

	// 1. 查询该 ReleaseApp 下的所有 Deployment
	var deployments []model.Deployment
	if err := sm.db.Where("release_id = ? AND env = ?", release.ID, constants.EnvTypePre).Find(&deployments).Error; err != nil {
		return 0, nil, fmt.Errorf("查询Deployment 失败: %w", err)
	}
	if len(deployments) == 0 {
		log.Warn("PreTriggered 状态下未找到任何 Deployment")
		return 0, nil, fmt.Errorf("no deployments found")
	}

	// 2. 统计状态
	var successCount, failedCount, pendingCount int
	for _, dep := range deployments {
		switch dep.Status {
		case constants.DeploymentStatusSuccess:
			successCount++
		case constants.DeploymentStatusFailed:
			failedCount++
		default:
			pendingCount++
		}
	}

	total := len(deployments)
	log.Info("Deployment 状态统计", zap.Int("total", total), zap.Int("success", successCount), zap.Int("failed", failedCount), zap.Int("pending", pendingCount))

	// 3. 判断下一步
	if failedCount > 0 {
		// 有失败 → ReleaseApp 失败
		return constants.ReleaseAppStatusPreFailed, func(r *model.ReleaseApp) {
			r.Reason = fmt.Sprintf("预发布失败: %d 个 Deployment 失败", failedCount)
		}, nil
	}
	if successCount == total {
		// 全部成功 → 进入 PreDeployed
		return constants.ReleaseAppStatusPreDeployed, nil, nil
	}

	// 4. 还有进行中的 → 继续等待
	log.Debug("预发布进行中，等待所有 Deployment 完成")
	return 0, nil, nil
}

// HandlePreDeployed handle StatusPreDeployed:13
func (sm *ReleaseStateMachine) HandlePreDeployed(ctx context.Context, release *model.ReleaseApp) (int8, func(*model.ReleaseApp), error) {
	// 无动作
	return 0, nil, nil
}

// HandleProdWaiting handle ProdWaiting:20 -> ProdCanTrigger:21
func (sm *ReleaseStateMachine) HandleProdWaiting(ctx context.Context, release *model.ReleaseApp) (int8, func(*model.ReleaseApp), error) {
	if sm.resolver == nil {
		return constants.ReleaseAppStatusProdCanTrigger, nil, nil
	}

	result, err := sm.resolver.CheckRelease(ctx, release, constants.EnvTypeProd)
	if err != nil {
		return 0, nil, err
	}

	if result.HasFailed() {
		reason := result.Summary()
		if reason == "" {
			reason = "依赖检查失败"
		}
		sm.logger.Warn("生产发布依赖失败", zap.Int64("release_id", release.ID), zap.String("reason", reason))
		return constants.ReleaseAppStatusProdFailed, func(r *model.ReleaseApp) {
			r.Reason = reason
		}, nil
	}

	if result.HasPending() {
		reason := result.Summary()
		sm.logger.Debug("生产发布依赖等待", zap.Int64("release_id", release.ID), zap.String("reason", reason))
		return 0, func(r *model.ReleaseApp) {
			r.Reason = reason
		}, nil
	}

	return constants.ReleaseAppStatusProdCanTrigger, func(r *model.ReleaseApp) {
		r.Reason = ""
	}, nil
}

// HandleProdCanTrigger handle ProdCanTrigger:21 -> ProdTriggered:22, gen deployments record
func (sm *ReleaseStateMachine) HandleProdCanTrigger(ctx context.Context, release *model.ReleaseApp) (int8, func(*model.ReleaseApp), error) {
	log := sm.logger.With(zap.Int64("release_id", release.ID)).Sugar()

	// 1. 校验 Build
	if release.BuildID == nil {
		return 0, nil, fmt.Errorf("build_id 为空")
	}
	var build model.Build
	if err := sm.db.First(&build, release.BuildID).Error; err != nil {
		return 0, nil, fmt.Errorf("build record not found: %w", err)
	}
	if build.BuildStatus != constants.BuildStatusSuccess {
		return 0, nil, fmt.Errorf("build status: %v", build.BuildStatus)
	}

	// 2. 加载 App
	var app model.Application
	if err := sm.db.First(&app, release.AppID).Error; err != nil {
		return 0, nil, fmt.Errorf("app record not found: %w", err)
	}

	// 3. 查询 Prod 环境的集群配置
	var configs []model.AppEnvConfig
	if err := sm.db.Where("app_id = ? AND env = ? AND status = 1", release.AppID, constants.EnvTypeProd).
		Find(&configs).Error; err != nil {
		return 0, nil, fmt.Errorf("查询 Prod 环境配置失败: %w", err)
	}
	if len(configs) == 0 {
		return 0, nil, fmt.Errorf("应用未配置生产环境")
	}

	// 4. 为每个集群创建 Deployment
	var failed []string
	for _, config := range configs {
		dep := model.Deployment{
			BatchID:        release.BatchID,
			AppID:          release.AppID,
			ReleaseID:      release.ID,
			DeploymentName: app.Name,
			Env:            constants.EnvTypeProd, // 生产环境
			ClusterName:    config.Cluster,
			Status:         "pending",
			RetryCount:     0,
		}

		result := sm.db.Where("release_id = ? AND env = ? AND cluster = ?", release.ID, constants.EnvTypeProd, config.Cluster).FirstOrCreate(&dep)
		if result.Error != nil {
			failed = append(failed, config.Cluster)
			log.With(zap.String("cluster", config.Cluster)).Errorf("创建 Deployment 失败, %v", result.Error)
		}
	}

	// 5. 记录失败信息
	if len(failed) > 0 {
		return 0, func(r *model.ReleaseApp) {
			r.Reason = fmt.Sprintf("生产部署触发失败: %v", failed)
		}, fmt.Errorf("部分集群创建失败")
	}

	log.Info(fmt.Sprintf("ProdDeploy 触发成功,创建了 %d 个集群的 Deployment", len(configs)), zap.String("image", build.ImageTag))
	return constants.ReleaseAppStatusProdTriggered, nil, nil
}

// HandleProdTriggered handle ProdTriggered:22 -> ProdDeployed:23, check deployments record
func (sm *ReleaseStateMachine) HandleProdTriggered(ctx context.Context, release *model.ReleaseApp) (int8, func(*model.ReleaseApp), error) {
	log := sm.logger.Sugar().With(zap.Int64("batch_id", release.BatchID), zap.Int64("release_id", release.ID))

	// 1. 查询该 ReleaseApp 下的所有 Production Deployment
	var deployments []model.Deployment
	if err := sm.db.Where("release_id = ? AND environment = ?", release.ID, constants.EnvTypeProd).Find(&deployments).Error; err != nil {
		return 0, nil, fmt.Errorf("查询Deployment 失败: %w", err)
	}
	if len(deployments) == 0 {
		log.Warnf("[ReleaseApp SM] Batch:%v ReleaseApp:%v ProdTriggered 状态下未找到任何 Deployment", release.BatchID, release.ID)
		return 0, nil, fmt.Errorf("no deployments found")
	}

	// 2. 统计状态
	var successCount, failedCount, pendingCount int
	for _, dep := range deployments {
		switch dep.Status {
		case constants.DeploymentStatusSuccess:
			successCount++
		case constants.DeploymentStatusFailed:
			failedCount++
		default:
			pendingCount++
		}
	}

	total := len(deployments)
	log.Infof("[ReleaseApp SM] Batch:%v ReleaseApp:%v Deployment成功数量: %v/%v", release.BatchID, release.ID, successCount, total)

	// 3. 判断下一步
	if failedCount > 0 {
		return constants.ReleaseAppStatusProdFailed, func(r *model.ReleaseApp) {
			r.Reason = fmt.Sprintf("生产部署失败: %d 个 Deployment 失败", failedCount)
		}, nil
	}
	if successCount == total {
		return constants.ReleaseAppStatusProdDeployed, nil, nil
	}

	// 4. 还有进行中的 → 继续等待
	log.Debugf("[ReleaseApp SM] Batch:%v ReleaseApp:%v 生产部署进行中，等待所有 Deployment 完成", release.BatchID, release.ID)
	return 0, nil, nil
}

// HandleProdDeployed handle StatusProdDeployed:23
func (sm *ReleaseStateMachine) HandleProdDeployed(ctx context.Context, release *model.ReleaseApp) (int8, func(*model.ReleaseApp), error) {
	return 0, nil, nil
}

func (sm *ReleaseStateMachine) HandleEmpty(ctx context.Context, release *model.ReleaseApp) (int8, func(*model.ReleaseApp), error) {
	// todo
	return 0, nil, nil
}
