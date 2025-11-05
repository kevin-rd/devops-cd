package release_app

import (
	"context"
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
	"go.uber.org/zap"
)

type Handler interface {
	Handle(ctx context.Context, release *model.ReleaseApp) (nextStatus int8, updateFunc func(*model.ReleaseApp), err error)
}

type HandlerFunc func(ctx context.Context, release *model.ReleaseApp) (nextStatus int8, updateFunc func(*model.ReleaseApp), err error)

func (h HandlerFunc) Handle(ctx context.Context, release *model.ReleaseApp) (nextStatus int8, updateFunc func(*model.ReleaseApp), err error) {
	return h(ctx, release)
}

func (sm *ReleaseStateMachine) registerHandlers() {
	sm.handlers[constants.ReleaseAppStatusPreWaiting] = HandlerFunc(sm.HandlePreWaiting)
	sm.handlers[constants.ReleaseAppStatusPreCanTrigger] = HandlerFunc(sm.HandleCanTrigger)
	sm.handlers[constants.ReleaseAppStatusPreTriggered] = HandlerFunc(sm.HandlePreTriggered)
	sm.handlers[constants.ReleaseAppStatusPreDeployed] = HandlerFunc(sm.HandlePreDeployed)
	sm.handlers[constants.ReleaseAppStatusProdWaiting] = HandlerFunc(sm.HandleProdWaiting)
	sm.handlers[constants.ReleaseAppStatusProdCanTrigger] = HandlerFunc(sm.HandleProdCanTrigger)
	sm.handlers[constants.ReleaseAppStatusProdTriggered] = HandlerFunc(sm.HandleProdTriggered)
	sm.handlers[constants.ReleaseAppStatusProdDeployed] = HandlerFunc(sm.HandleProdDeployed)
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

// HandleCanTrigger handle PreCanTrigger:11 -> PreTriggered:12, gen deployments record
func (sm *ReleaseStateMachine) HandleCanTrigger(ctx context.Context, release *model.ReleaseApp) (int8, func(*model.ReleaseApp), error) {
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
	if err := sm.db.First(&app, release.AppID).Error; err != nil {
		return 0, nil, fmt.Errorf("app record not found: %w", err)
	}

	// 3. 幂等创建 Deployment
	clusters := []string{"default"} // todo: 从 app 配置读取
	var failed []string
	for _, cluster := range clusters {
		dep := model.Deployment{
			BatchID:        release.BatchID,
			AppID:          release.AppID,
			ReleaseID:      release.ID,
			DeploymentName: app.Name,
			Environment:    constants.EnvTypePre,
			Cluster:        cluster,
			ImageTag:       build.ImageTag,
			Status:         "pending",
			RetryCount:     0,
		}

		// 正确使用 FirstOrCreate
		result := sm.db.Where("release_id = ? AND environment = ? AND cluster = ?", release.ID, constants.EnvTypePre, cluster).FirstOrCreate(&dep)

		if result.Error != nil {
			failed = append(failed, cluster)
			log.Error("创建 Deployment 失败", zap.String("cluster", cluster), zap.Error(result.Error))
		}
	}

	// 4. 记录失败信息
	if len(failed) > 0 {
		release.Reason = fmt.Sprintf("failed clusters: %v", failed)
		return 0, nil, fmt.Errorf("部分集群创建失败")
	}

	log.Info("PreDeploy 触发成功", zap.String("image", build.ImageTag))
	return constants.ReleaseAppStatusPreTriggered, nil, nil
}

// HandlePreTriggered handle PreTriggered:12 -> PreDeployed:13, check deployments record
func (sm *ReleaseStateMachine) HandlePreTriggered(ctx context.Context, release *model.ReleaseApp) (int8, func(*model.ReleaseApp), error) {
	log := sm.logger.With(zap.Int64("release_id", release.ID), zap.Int64("batch_id", release.BatchID))

	// 1. 查询该 ReleaseApp 下的所有 Deployment
	var deployments []model.Deployment
	if err := sm.db.Where("release_id = ? AND environment = ?", release.ID, constants.EnvTypePre).Find(&deployments).Error; err != nil {
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
	if err := sm.db.First(&app, release.AppID).Error; err != nil {
		return 0, nil, fmt.Errorf("app record not found: %w", err)
	}

	// 3. 幂等创建 Deployment
	clusters := []string{"default"} // todo: 从 app 配置读取生产集群
	var failed []string
	for _, cluster := range clusters {
		dep := model.Deployment{
			BatchID:        release.BatchID,
			AppID:          release.AppID,
			ReleaseID:      release.ID,
			DeploymentName: app.Name,
			Environment:    constants.EnvTypeProd, // 生产环境
			Cluster:        cluster,
			ImageTag:       build.ImageTag,
			Status:         "pending",
			RetryCount:     0,
		}

		result := sm.db.Where("release_id = ? AND environment = ? AND cluster = ?", release.ID, constants.EnvTypeProd, cluster).
			FirstOrCreate(&dep)

		if result.Error != nil {
			failed = append(failed, cluster)
			log.Error("创建 Deployment 失败", zap.String("cluster", cluster), zap.Error(result.Error))
		}
	}

	// 4. 记录失败信息
	if len(failed) > 0 {
		return 0, func(r *model.ReleaseApp) {
			r.Reason = fmt.Sprintf("生产部署触发失败: %v", failed)
		}, fmt.Errorf("部分集群创建失败")
	}

	log.Info("ProdDeploy 触发成功", zap.String("image", build.ImageTag))
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
