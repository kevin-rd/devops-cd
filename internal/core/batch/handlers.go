package batch

import (
	"context"
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
	"time"
)

type Handler interface {
	Handle(ctx context.Context, batch *model.Batch) (int8, func(*model.Batch), error)
}

type HandlerFunc func(ctx context.Context, batch *model.Batch) (int8, func(*model.Batch), error)

func (h HandlerFunc) Handle(ctx context.Context, batch *model.Batch) (int8, func(*model.Batch), error) {
	return h(ctx, batch)
}

func (sm *StateMachine) registerHandlers() {
	sm.handlers[constants.BatchStatusDraft] = HandlerFunc(HandleDraft)
	sm.handlers[constants.BatchStatusSealed] = HandlerFunc(HandleSealed)
	sm.handlers[constants.BatchStatusPreWaiting] = HandlerFunc(sm.HandlePreWaiting)
	sm.handlers[constants.BatchStatusPreDeploying] = HandlerFunc(sm.HandlePreDeploying)
	sm.handlers[constants.BatchStatusPreDeployed] = HandlerFunc(sm.HandlePreDeployed)
	sm.handlers[constants.BatchStatusProdWaiting] = HandlerFunc(sm.HandleProdWaiting)
	sm.handlers[constants.BatchStatusProdDeploying] = HandlerFunc(sm.HandleProdDeploying)
	sm.handlers[constants.BatchStatusProdDeployed] = HandlerFunc(sm.HandleProdDeployed)
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
	batchName := fmt.Sprintf("%s[%v]", batch.BatchNumber, batch.ID)

	// 更新 all releaseApp.status -> PreWaiting
	result := sm.db.Model(&model.ReleaseApp{}).
		Where("batch_id = ?", batch.ID).
		Where("status < ?", constants.ReleaseAppStatusPreWaiting).
		Update("status", constants.ReleaseAppStatusPreWaiting)

	if result.Error != nil {
		return 0, nil, fmt.Errorf("更新发布记录状态失败: %w", result.Error)
	}

	sm.logger.Info(fmt.Sprintf("Batch:%s -> %d 条发布记录更新为 PreWaiting", batchName, result.RowsAffected))
	return constants.BatchStatusPreDeploying, nil, nil
}

// HandlePreDeploying handle StatusPreDeploying:21 -> StatusPreDeployed:22
func (sm *StateMachine) HandlePreDeploying(ctx context.Context, batch *model.Batch) (int8, func(*model.Batch), error) {
	batchName := fmt.Sprintf("%s[%v]", batch.BatchNumber, batch.ID)

	// 统计status < PreDeployed的release_app数量
	var notDeployedCount int64
	if err := sm.db.Model(&model.ReleaseApp{}).
		Where("batch_id = ? AND status < ?", batch.ID, constants.ReleaseAppStatusPreDeployed).
		Count(&notDeployedCount).Error; err != nil {
		return 0, nil, fmt.Errorf("查询未部署应用失败: %w", err)
	}
	if notDeployedCount > 0 {
		sm.logger.Debug(fmt.Sprintf("Batch:%s -> PreDeploying 进行中，剩余 %d 条", batchName, notDeployedCount))
		return 0, nil, nil // 继续等待
	}

	// 检查是否有release, 防止空批次误判
	var total int64
	sm.db.Model(&model.ReleaseApp{}).Where("batch_id = ?", batch.ID).Count(&total)
	if total == 0 {
		return 0, nil, fmt.Errorf("批次无发布记录")
	}

	return constants.BatchStatusPreDeployed, func(b *model.Batch) {
		now := time.Now()
		b.PreDeployFinishedAt = &now
	}, nil
}

// HandlePreDeployed handle StatusPreDeployed:22
func (sm *StateMachine) HandlePreDeployed(ctx context.Context, batch *model.Batch) (int8, func(*model.Batch), error) {
	return 0, nil, nil
}

// HandleProdWaiting handle StatusProdWaiting:30 -> StatusProdDeploying:31
func (sm *StateMachine) HandleProdWaiting(ctx context.Context, batch *model.Batch) (int8, func(*model.Batch), error) {
	batchName := fmt.Sprintf("%s[%v]", batch.BatchNumber, batch.ID)

	// 更新 all releaseApp.status -> ProdWaiting
	result := sm.db.Model(&model.ReleaseApp{}).
		Where("batch_id = ?", batch.ID).
		Where("status = ?", constants.ReleaseAppStatusPreDeployed).
		Update("status", constants.ReleaseAppStatusProdWaiting)

	if result.Error != nil {
		return 0, nil, fmt.Errorf("更新发布记录状态失败: %w", result.Error)
	}

	sm.logger.Info(fmt.Sprintf("Batch:%s -> %d 条release_app记录更新为 ProdWaiting", batchName, result.RowsAffected))
	return constants.BatchStatusProdDeploying, nil, nil
}

// HandleProdDeploying handle StatusProdDeploying:31 -> StatusProdDeployed:32
func (sm *StateMachine) HandleProdDeploying(ctx context.Context, batch *model.Batch) (int8, func(*model.Batch), error) {
	batchName := fmt.Sprintf("%s[%v]", batch.BatchNumber, batch.ID)

	// 统计 status < ProdDeployed 的 release_app 数量
	var notDeployedCount int64
	if err := sm.db.Model(&model.ReleaseApp{}).
		Where("batch_id = ? AND status < ?", batch.ID, constants.ReleaseAppStatusProdDeployed).
		Count(&notDeployedCount).Error; err != nil {
		return 0, nil, fmt.Errorf("查询未部署应用失败: %w", err)
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
		b.ProdDeployFinishedAt = &now
	}, nil
}

// HandleProdDeployed handle StatusProdDeployed:32
func (sm *StateMachine) HandleProdDeployed(ctx context.Context, batch *model.Batch) (int8, func(*model.Batch), error) {
	return 0, nil, nil
}
