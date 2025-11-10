package core

import (
	"context"
	"devops-cd/internal/adapter/deploy"
	"devops-cd/internal/adapter/notification"
	"devops-cd/internal/core/batch"
	"devops-cd/internal/core/dependency"
	"devops-cd/internal/core/deployment"
	"devops-cd/internal/core/release_app"
	"devops-cd/internal/model"
	"devops-cd/internal/pkg/config"
	"devops-cd/pkg/constants"
	"fmt"
	"github.com/samber/lo"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CoreEngine CD核心引擎
type CoreEngine struct {
	db       *gorm.DB
	notifier notification.Notifier
	logger   *zap.Logger

	running  bool
	stopChan chan struct{}

	batchSM      *batch.StateMachine
	releaseSM    *release_app.ReleaseStateMachine
	deploymentSM *deployment.StateMachine

	batchTask map[int64]context.CancelFunc
}

// NewCoreEngine 创建核心引擎
func NewCoreEngine(db *gorm.DB, logger *zap.Logger, coreCfg *config.CoreConfig) *CoreEngine {

	// 创建部署服务
	// deployService := deploy.NewK8sDeployer( deploy.NewMockK8sDeployClient(), notification.NewLogNotifier(logger), db, nil, logger)
	deployService := deploy.NewMockDeployer()

	depCfg := dependency.Config{}
	if coreCfg != nil && len(coreCfg.AppTypes) > 0 {
		depCfg.AppTypeDepends = make(map[string][]string, len(coreCfg.AppTypes))
		for appType, appTypeCfg := range coreCfg.AppTypes {
			if len(appTypeCfg.Dependencies) == 0 {
				continue
			}
			deps := make([]string, len(appTypeCfg.Dependencies))
			copy(deps, appTypeCfg.Dependencies)
			depCfg.AppTypeDepends[appType] = deps
		}
	}
	resolver := dependency.NewResolver(db, logger, depCfg)

	return &CoreEngine{
		db:       db,
		notifier: notification.NewLogNotifier(logger),
		logger:   logger,
		stopChan: make(chan struct{}),

		batchSM:      batch.NewBatchStateMachine(db, logger),
		releaseSM:    release_app.NewReleaseStateMachine(db, logger, resolver),
		deploymentSM: deployment.NewDeploymentStateMachine(db, logger, deployService, resolver),

		batchTask: make(map[int64]context.CancelFunc, 10),
	}
}

// Start 启动核心引擎
func (e *CoreEngine) Start(scanInterval time.Duration) {
	if e.running {
		e.logger.Warn("核心引擎已在运行中")
		return
	}

	e.running = true
	e.logger.Info("CoreEngine starting...", zap.Duration("scan_interval", scanInterval))

	// 启动定时扫描
	go e.runScanner(scanInterval)
}

// Stop 停止核心引擎
func (e *CoreEngine) Stop() {
	if !e.running {
		return
	}

	e.logger.Info("正在停止核心引擎...")
	close(e.stopChan)
	e.running = false
	e.logger.Info("核心引擎已停止")
}

// runScanner 运行扫描器
func (e *CoreEngine) runScanner(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.ScanBatches()
		case <-e.stopChan:
			return
		}
	}
}

func (e *CoreEngine) ScanBatches() {
	var batches []model.Batch
	// 查询 Sealed < status < Completed 并且 create_at < 30 Days
	if err := e.db.Where("status > ? AND status < ?", constants.BatchStatusDraft, constants.BatchStatusCompleted).
		Where("created_at > ?", time.Now().Add(-time.Hour*24*30)).
		Order("id DESC").Find(&batches).Error; err != nil {
		e.logger.Error(fmt.Sprintf("[BatchScaner] 查询批次失败: %v", err))
		return
	}

	ids := lo.Map(batches, func(b model.Batch, i int) int64 { return b.ID })
	e.logger.Debug(fmt.Sprintf("[BatchScaner] 待处理的Batch %v个: %v", len(batches), ids))

	for _, b := range batches {
		if cancel, exists := e.batchTask[b.ID]; exists {
			// 如果已经存在, 检查是否需要结束
			if b.Status == constants.BatchStatusCompleted || b.Status == constants.BatchStatusCancelled {
				cancel()
				delete(e.batchTask, b.ID)
			}
			continue
		}

		ctx, cancel := context.WithCancel(context.TODO())
		e.batchTask[b.ID] = cancel
		go e.batchWork(ctx, b.ID)
	}
}

func (e *CoreEngine) batchWork(ctx context.Context, batchId int64) {
	ticker := time.NewTicker(10 * time.Second)
	defer func() {
		defer ticker.Stop()
		delete(e.batchTask, batchId)
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			// 0. 每次重新查询batch状态
			b := model.Batch{}
			if err := e.db.First(&b, batchId).Error; err != nil {
				e.logger.Error(fmt.Sprintf("[BatchScaner] 查询批次失败: %v", err))
			}

			// 1. 执行Batch
			e.batchSM.Process(ctx, &b)

			// 2. 执行 releases
			var releases []model.ReleaseApp
			if err := e.db.Where("batch_id = ? AND status > ?", b.ID, constants.ReleaseAppStatusPending).
				Find(&releases).Error; err != nil {
				e.logger.Error("查询 ReleaseApp 失败", zap.Error(err))
				continue
			}
			for i := range releases {
				e.releaseSM.Process(ctx, &releases[i])
			}

			// 3. 执行 deployments
			e.scamDeployment(ctx, b.ID)

			// 4. completed -> cancel
			if b.Status == constants.BatchStatusCompleted {
				return
			}
		}
	}
}

func (e *CoreEngine) scamDeployment(ctx context.Context, batchID int64) {
	var deps []model.Deployment
	if err := e.db.Where("batch_id = ? AND status != ?", batchID, constants.DeploymentStatusSuccess).Find(&deps).Error; err != nil {
		e.logger.Error("扫描 Deployment 失败", zap.Error(err))
		return
	}

	for i := range deps {
		dep := &deps[i]
		e.deploymentSM.Process(ctx, dep)
	}
}

// updateBatchBuilds 更新批次中的构建记录 todo: 是否需要转移
func (e *CoreEngine) updateBatchBuilds(batchID int64) error {
	// 查询该批次的所有应用发布记录
	var releases []model.ReleaseApp
	err := e.db.Where("batch_id = ? AND is_locked = ?", batchID, false).Find(&releases).Error

	if err != nil {
		return fmt.Errorf("查询批次发布记录失败: %w", err)
	}

	if len(releases) == 0 {
		return nil
	}

	e.logger.Info("检查批次构建更新",
		zap.Int64("batch_id", batchID),
		zap.Int("release_count", len(releases)))

	updated := 0
	for i := range releases {
		release := &releases[i]

		// 跳过已锁定的应用（已封板）
		if release.IsLocked {
			continue
		}

		// 查询该应用的最新成功构建
		var latestBuild model.Build
		err := e.db.Where("app_id = ? AND build_status = ?",
			release.AppID, "success").
			Order("created_at DESC").
			First(&latestBuild).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				e.logger.Debug("没有找到最新构建",
					zap.Int64("app_id", release.AppID))
				continue
			}
			return fmt.Errorf("查询最新构建失败: %w", err)
		}

		// 如果有更新的构建，更新记录
		if release.BuildID == nil || latestBuild.ID != *release.BuildID {
			oldBuildID := int64(0)
			if release.BuildID != nil {
				oldBuildID = *release.BuildID
			}

			e.logger.Info("发现新构建",
				zap.Int64("app_id", release.AppID),
				zap.Int64("old_build_id", oldBuildID),
				zap.Int64("new_build_id", latestBuild.ID))

			// 更新 build_id（未封板时只更新 build_id，不更新 target_tag）
			release.BuildID = &latestBuild.ID

			if err := e.db.Save(release).Error; err != nil {
				e.logger.Error("更新发布记录失败",
					zap.Int64("release_id", release.ID),
					zap.Error(err))
				continue
			}

			updated++
		}
	}

	if updated > 0 {
		e.logger.Info("批次构建记录已更新",
			zap.Int64("batch_id", batchID),
			zap.Int("updated_count", updated))
	}

	return nil
}
