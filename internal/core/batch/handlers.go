package batch

import (
	"context"
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type StateHandler interface {
	Handle(ctx context.Context, batch *model.Batch) (int8, func(*model.Batch), error)
}

type StateHandlerFunc func(ctx context.Context, batch *model.Batch) (int8, func(*model.Batch), error)

func (h StateHandlerFunc) Handle(ctx context.Context, batch *model.Batch) (int8, func(*model.Batch), error) {
	return h(ctx, batch)
}

func (sm *StateMachine) registerHandlers() {
	sm.handlers[constants.BatchStatusDraft] = StateHandlerFunc(HandleDraft)
	sm.handlers[constants.BatchStatusSealed] = StateHandlerFunc(HandleSealed)
	sm.handlers[constants.BatchStatusPreWaiting] = StateHandlerFunc(sm.HandlePreWaiting)
	sm.handlers[constants.BatchStatusPreDeploying] = StateHandlerFunc(sm.HandlePreDeploying)
	sm.handlers[constants.BatchStatusPreDeployed] = StateHandlerFunc(sm.HandlePreDeployed)
	sm.handlers[constants.BatchStatusProdWaiting] = StateHandlerFunc(sm.HandleProdWaiting)
	sm.handlers[constants.BatchStatusProdDeploying] = StateHandlerFunc(sm.HandleProdDeploying)
	sm.handlers[constants.BatchStatusProdDeployed] = StateHandlerFunc(sm.HandleProdDeployed)
}

// all handlers

// HandleDraft handle StatusDraft:0
func HandleDraft(ctx context.Context, batch *model.Batch) (int8, func(*model.Batch), error) {
	return 0, nil, nil
}

// HandleSealed handle StatusSealed:10
func HandleSealed(ctx context.Context, batch *model.Batch) (int8, func(*model.Batch), error) {
	return 0, nil, nil
}

// HandlePreWaiting handle StatusPreWaiting:20 -> StatusPreDeploying:21
func (sm *StateMachine) HandlePreWaiting(ctx context.Context, batch *model.Batch) (int8, func(*model.Batch), error) {
	batchName := fmt.Sprintf("%d「%s」", batch.ID, batch.BatchNumber)

	// 更新 all releaseApp.status -> PreWaiting (只更新需要 pre 的应用)
	result := sm.db.Model(&model.ReleaseApp{}).
		Where("batch_id = ? AND skip_pre_env = ?", batch.ID, false).
		Where("status < ?", constants.ReleaseAppStatusPreWaiting).
		Update("status", constants.ReleaseAppStatusPreWaiting)

	if result.Error != nil {
		return 0, nil, fmt.Errorf("更新发布记录状态失败: %w", result.Error)
	}

	sm.logger.Info(fmt.Sprintf("[Batch SM: %d]:%s :: %d条ReleaseApp更新为 PreWaiting", batch.ID, batchName, result.RowsAffected))
	return constants.BatchStatusPreDeploying, nil, nil
}

// HandlePreDeploying handle StatusPreDeploying:21
// When all success -> StatusPreDeployed:22
// When any failed -> StatusPreFailed:24
// todo: 需要添加失败情况
func (sm *StateMachine) HandlePreDeploying(ctx context.Context, batch *model.Batch) (int8, func(*model.Batch), error) {
	batchName := fmt.Sprintf("%s[%v]", batch.BatchNumber, batch.ID)

	// 统计status < PreDeployed的release_app数量 (只统计需要 pre 的应用)
	var countInDoing int64
	if err := sm.db.Model(&model.ReleaseApp{}).Where("batch_id = ? AND skip_pre_env = ?", batch.ID, false).
		Scopes(StatusIn(constants.BatchStatusPreWaiting)).
		Where("status != ?", constants.ReleaseAppStatusPreDeployed).
		Count(&countInDoing).Error; err != nil {
		return 0, nil, fmt.Errorf("[db] 查询未部署应用时失败: %w", err)
	}
	if countInDoing > 0 {
		sm.logger.Debug(fmt.Sprintf("Batch:%s -> PreDeploying 进行中，剩余 %d 条", batchName, countInDoing))
		return 0, nil, nil // 继续等待
	}

	// 检查是否有release, 防止空批次误判
	var countInPre int64
	sm.db.Model(&model.ReleaseApp{}).Where("batch_id = ? AND skip_pre_env = ?", batch.ID, false).Count(&countInPre)
	if countInPre == 0 {
		// 不应该到达这里,但保险起见,直接跳到 prod
		sm.logger.Error(fmt.Sprintf("Batch:%s -> PreDeploying 没有需要 pre 的应用, 直接跳转到 ProdWaiting", batchName))
		return constants.BatchStatusPreDeployed, nil, nil
	}

	return constants.BatchStatusPreDeployed, func(b *model.Batch) {
		now := time.Now()
		b.PreFinishedAt = &now
	}, nil
}

// HandlePreDeployed handle StatusPreDeployed:22
func (sm *StateMachine) HandlePreDeployed(ctx context.Context, batch *model.Batch) (int8, func(*model.Batch), error) {
	return 0, nil, nil
}

// HandleProdWaiting handle StatusProdWaiting:30 -> StatusProdDeploying:31
func (sm *StateMachine) HandleProdWaiting(ctx context.Context, batch *model.Batch) (int8, func(*model.Batch), error) {
	batchName := fmt.Sprintf("%s[%v]", batch.BatchNumber, batch.ID)

	// 1. 更新跳过 pre 的应用: Tagged → ProdWaiting
	result1 := sm.db.Model(&model.ReleaseApp{}).
		Where("batch_id = ? AND skip_pre_env = ?", batch.ID, true).
		Where("status = ?", constants.ReleaseAppStatusTagged).
		Update("status", constants.ReleaseAppStatusProdWaiting)

	// 2. 更新经过 pre 的应用: PreDeployed → ProdWaiting
	result2 := sm.db.Model(&model.ReleaseApp{}).
		Where("batch_id = ? AND skip_pre_env = ?", batch.ID, false).
		Where("status = ?", constants.ReleaseAppStatusPreDeployed).
		Update("status", constants.ReleaseAppStatusProdWaiting)

	if result1.Error != nil || result2.Error != nil {
		return 0, nil, fmt.Errorf("更新发布记录状态失败")
	}

	total := result1.RowsAffected + result2.RowsAffected
	sm.logger.Info(fmt.Sprintf("Batch:%s -> %d 条release_app记录更新为 ProdWaiting (跳过pre:%d, 经过pre:%d)",
		batchName, total, result1.RowsAffected, result2.RowsAffected))
	return constants.BatchStatusProdDeploying, nil, nil
}

// HandleProdDeploying handle StatusProdDeploying:31
// When all success -> StatusProdDeployed:32
// When any failed -> StatusProdFailed:34
// todo: 需要添加失败情况
func (sm *StateMachine) HandleProdDeploying(ctx context.Context, batch *model.Batch) (int8, func(*model.Batch), error) {
	batchName := fmt.Sprintf("%s[%v]", batch.BatchNumber, batch.ID)

	// 统计 status < ProdDeployed 的 release_app 数量
	var notDeployedCount int64
	if err := sm.db.Model(&model.ReleaseApp{}).Where("batch_id = ?", batch.ID).
		Scopes(StatusIn(constants.BatchStatusProdWaiting)).Where("status != ?", constants.ReleaseAppStatusProdDeployed).
		Count(&notDeployedCount).Error; err != nil {
		return 0, nil, fmt.Errorf("[db] 统计未部署应用时失败: %w", err)
	}
	if notDeployedCount > 0 {
		sm.logger.Debug(fmt.Sprintf("[Batch SM] Batch:%s -> ProdDeploying 进行中，剩余 %d 条", batchName, notDeployedCount))
		return 0, nil, nil // 继续等待
	}

	// 检查是否有 release，防止空批次误判
	var total int64
	sm.db.Model(&model.ReleaseApp{}).Where("batch_id = ?", batch.ID).Count(&total)
	if total == 0 {
		return 0, nil, fmt.Errorf("批次无发布记录")
	}

	return constants.BatchStatusProdDeployed, func(b *model.Batch) {
		now := time.Now()
		b.ProdFinishedAt = &now
	}, nil
}

// HandleProdDeployed handle StatusProdDeployed:32
func (sm *StateMachine) HandleProdDeployed(ctx context.Context, batch *model.Batch) (int8, func(*model.Batch), error) {
	return 0, nil, nil
}

// ---- common functions -----

// StatusIn 批量查询指定范围内的状态, 左闭右开区间
func StatusIn(status int8) func(db *gorm.DB) *gorm.DB {
	start, end := constants.Range10(status)
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("status >= ? AND status < ?", start, end)
	}
}
