package core

import (
	"context"
	"devops-cd/internal/adapter/deploy"
	"devops-cd/internal/adapter/notification"
	"devops-cd/internal/core/batch"
	"devops-cd/internal/core/deployment"
	"devops-cd/internal/core/release_app"
	"devops-cd/internal/model"
	"fmt"
	"github.com/samber/lo"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"devops-cd/pkg/constants"
)

// BatchScanner 批次扫描器
type BatchScanner struct {
	db *gorm.DB

	notifier notification.Notifier
	logger   *zap.Logger

	batchTask map[int64]context.CancelFunc

	batchSM      *batch.StateMachine
	releaseSM    *release_app.ReleaseStateMachine
	deploymentSM *deployment.StateMachine
}

// NewBatchScanner 创建批次扫描器
func NewBatchScanner(db *gorm.DB, deployer deploy.Deployer, logger *zap.Logger) *BatchScanner {
	return &BatchScanner{
		db:       db,
		notifier: notification.NewLogNotifier(logger),
		logger:   logger,

		batchTask: make(map[int64]context.CancelFunc, 10),

		batchSM:      batch.NewBatchStateMachine(db, logger),
		releaseSM:    release_app.NewReleaseStateMachine(db, logger),
		deploymentSM: deployment.NewDeploymentStateMachine(db, logger, deployer),
	}
}

func (s *BatchScanner) ScanBatches() {
	var batches []model.Batch
	// 查询 Sealed < status < Completed 并且 create_at < 30 Days
	if err := s.db.Where("status > ? AND status < ?", constants.BatchStatusSealed, constants.BatchStatusCompleted).
		Where("created_at > ?", time.Now().Add(-time.Hour*24*30)).
		Order("id DESC").Find(&batches).Error; err != nil {
		s.logger.Error(fmt.Sprintf("[BatchScaner] 查询批次失败: %v", err))
		return
	}

	ids := lo.Map(batches, func(b model.Batch, i int) int64 { return b.ID })
	s.logger.Debug(fmt.Sprintf("[BatchScaner] 待处理的Batch %v个: %v", len(batches), ids))

	for _, batch := range batches {
		if cancel, exists := s.batchTask[batch.ID]; exists {
			// 如果已经存在, 检查是否需要结束
			if batch.Status == constants.BatchStatusCompleted || batch.Status == constants.BatchStatusCancelled {
				cancel()
				delete(s.batchTask, batch.ID)
			}
			continue
		}

		ctx, cancel := context.WithCancel(context.TODO())
		s.batchTask[batch.ID] = cancel
		go s.batchWork(ctx, batch.ID)
	}
}

func (s *BatchScanner) batchWork(ctx context.Context, batchId int64) {
	ticker := time.NewTicker(10 * time.Second)
	defer func() {
		defer ticker.Stop()
		delete(s.batchTask, batchId)
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			// 0. 每次重新查询batch状态
			batch := model.Batch{}
			if err := s.db.First(&batch, batchId).Error; err != nil {
				s.logger.Error(fmt.Sprintf("[BatchScaner] 查询批次失败: %v", err))
			}

			// 1. 执行Batch
			s.batchSM.Process(ctx, &batch)

			// 2. 执行 releases
			var releases []model.ReleaseApp
			if err := s.db.Where("batch_id = ? AND status > ?", batch.ID, constants.ReleaseAppStatusPending).
				Find(&releases).Error; err != nil {
				s.logger.Error("查询 ReleaseApp 失败", zap.Error(err))
				continue
			}
			for i := range releases {
				s.releaseSM.Process(ctx, &releases[i])
			}

			// 3. 执行 deployments
			s.scamDeployment(ctx, batch.ID)

			// 4. completed -> cancel
			if batch.Status == constants.BatchStatusCompleted {
				return
			}
		}
	}
}

func (s *BatchScanner) scamDeployment(ctx context.Context, batchID int64) {
	var deps []model.Deployment
	if err := s.db.Where("batch_id = ? AND status != ?", batchID, constants.DeploymentStatusSuccess).Find(&deps).Error; err != nil {
		s.logger.Error("扫描 Deployment 失败", zap.Error(err))
		return
	}

	for i := range deps {
		dep := &deps[i]
		s.deploymentSM.Process(ctx, dep)
	}
}

// updateBatchBuilds 更新批次中的构建记录 todo: 是否需要转移
func (s *BatchScanner) updateBatchBuilds(batchID int64) error {
	// 查询该批次的所有应用发布记录
	var releases []model.ReleaseApp
	err := s.db.Where("batch_id = ? AND is_locked = ?", batchID, false).Find(&releases).Error

	if err != nil {
		return fmt.Errorf("查询批次发布记录失败: %w", err)
	}

	if len(releases) == 0 {
		return nil
	}

	s.logger.Info("检查批次构建更新",
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
		err := s.db.Where("app_id = ? AND build_status = ?",
			release.AppID, "success").
			Order("created_at DESC").
			First(&latestBuild).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				s.logger.Debug("没有找到最新构建",
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

			s.logger.Info("发现新构建",
				zap.Int64("app_id", release.AppID),
				zap.Int64("old_build_id", oldBuildID),
				zap.Int64("new_build_id", latestBuild.ID))

			// 更新 build_id（未封板时只更新 build_id，不更新 target_tag）
			release.BuildID = &latestBuild.ID

			if err := s.db.Save(release).Error; err != nil {
				s.logger.Error("更新发布记录失败",
					zap.Int64("release_id", release.ID),
					zap.Error(err))
				continue
			}

			updated++
		}
	}

	if updated > 0 {
		s.logger.Info("批次构建记录已更新",
			zap.Int64("batch_id", batchID),
			zap.Int("updated_count", updated))
	}

	return nil
}

// ProcessBatchStateChange 处理批次状态变更
func (s *BatchScanner) ProcessBatchStateChange(batchID int64, event string, operator string) error {
	return s.batchSM.ProcessStateChange(batchID, event, operator)
}

// GetBatchStatus 获取批次状态信息
func (s *BatchScanner) GetBatchStatus(batchID int64) (map[string]interface{}, error) {
	var batch model.Batch
	if err := s.db.First(&batch, batchID).Error; err != nil {
		return nil, fmt.Errorf("查询批次失败: %w", err)
	}

	stateName := constants.BatchStatusToString(batch.Status)

	return map[string]interface{}{
		"batch_id":           batch.ID,
		"batch_number":       batch.BatchNumber,
		"current_state":      batch.Status,
		"current_state_name": stateName,
		"available_events":   "[]",
	}, nil
}
