package transitions

import (
	"devops-cd/internal/model"
	"devops-cd/pkg/constants"
	"fmt"
	"gorm.io/gorm"
	"time"
)

// TriggerPreDeployTransition 处理开始预发布部署
type TriggerPreDeployTransition struct {
	db *gorm.DB
}

func (h TriggerPreDeployTransition) Handle(batch *model.Batch, from, to int8, options *TransitionOptions) error {
	// 1. 检查前置条件：必须已审批通过
	if batch.ApprovalStatus != constants.ApprovalStatusApproved &&
		batch.ApprovalStatus != constants.ApprovalStatusSkipped {
		return fmt.Errorf("预发布失败: 批次未审批通过，当前审批状态: %s", batch.ApprovalStatus)
	}

	// 2. 检查前置条件：必须已封板
	if batch.Status < constants.BatchStatusSealed || batch.SealedAt == nil {
		return fmt.Errorf("预发布失败: 批次未封板")
	}

	// 3. todo: 所有的app都已经tagged

	// 4. 记录时间
	now := time.Now()
	batch.PreStartedAt = &now
	batch.PreTriggeredBy = &options.operator

	return nil
}

func (h TriggerPreDeployTransition) After(batch *model.Batch, from, to int8, options *TransitionOptions) {

}
