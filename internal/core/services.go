package core

import (
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
)

// ProcessBatchEvent 处理批次事件
func (e *CoreEngine) ProcessBatchEvent(batchID int64, event string, operator, reason string) error {
	return e.batchSM.ProcessStateChange(batchID, event, operator, reason)
}

// GetBatchStatus 获取批次状态
func (e *CoreEngine) GetBatchStatus(batchID int64) (map[string]interface{}, error) {
	var batch model.Batch
	if err := e.db.First(&batch, batchID).Error; err != nil {
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
