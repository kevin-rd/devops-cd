package transitions

import (
	"devops-cd/internal/pkg/logger"
	"devops-cd/pkg/constants"

	"gorm.io/gorm"
)

func AllTransitions(db *gorm.DB) []StateTransition {
	var transitions = []StateTransition{
		// 草稿 -> 已封板
		{
			From:        constants.BatchStatusDraft,
			To:          constants.BatchStatusSealed,
			Handler:     TriggerSealTransition{db: db, logger: logger.Sugar()},
			AllowSource: SourceOutside,
		},
		// 已封板 -> 触发预发布（需要检查审批状态）
		{
			From:        constants.BatchStatusSealed,
			To:          constants.BatchStatusPreWaiting,
			Handler:     TriggerPreDeployTransition{db: db},
			AllowSource: SourceOutside,
		},
		// Pre 部署中 -> Pre 已部署 todo: internal
		{
			From:        constants.BatchStatusPreDeploying,
			To:          constants.BatchStatusPreDeployed,
			Handler:     OnPreDeployCompletedTransition{db: db},
			AllowSource: SourceInside,
		},
		// Pre 已部署 -> 已验收
		{
			From:        constants.BatchStatusPreDeployed,
			To:          constants.BatchStatusPreAccepted,
			Handler:     &TriggerPreAccepted{db: db},
			AllowSource: SourceOutside,
		},
		// Pre 已验收 -> 触发Prod
		{
			From:        constants.BatchStatusPreAccepted,
			To:          constants.BatchStatusProdWaiting,
			Handler:     TriggerProdDeployTransition{db: db},
			AllowSource: SourceOutside,
		},
		// 无预发布时 -> 触发Prod
		{
			From:        constants.BatchStatusSealed,
			To:          constants.BatchStatusProdWaiting,
			Handler:     TriggerProdDeployTransition{db: db}, // withoutPre
			AllowSource: SourceOutside,
		},
		// Prod 部署中 -> Prod 已部署 // todo: internal
		{
			From:    constants.BatchStatusProdDeploying,
			To:      constants.BatchStatusProdDeployed,
			Handler: OnProdDeployCompletedTransition{db: db},
		},
		// Prod 已部署 -> 已验收
		{
			From:        constants.BatchStatusProdDeployed,
			To:          constants.BatchStatusProdAccepted,
			Handler:     &TriggerProdAccepted{db: db},
			AllowSource: SourceOutside,
		},

		// Prod已验收 -> PM最终验收
		{
			From:        constants.BatchStatusProdAccepted,
			To:          constants.BatchStatusCompleted,
			Handler:     FinalAcceptTransition{db: db},
			AllowSource: SourceOutside,
		},

		// 草稿 -> 取消
		{
			From:        constants.BatchStatusDraft,
			To:          constants.BatchStatusCancelled,
			Handler:     TriggerCancelTransition{db: db},
			AllowSource: SourceOutside,
		},
		// 已封板 -> 取消
		{
			From:        constants.BatchStatusSealed,
			To:          constants.BatchStatusCancelled,
			Handler:     TriggerCancelTransition{db: db},
			AllowSource: SourceOutside,
		},
		// Pre 已部署 -> 取消
		{
			From:        constants.BatchStatusPreDeployed,
			To:          constants.BatchStatusCancelled,
			Handler:     TriggerCancelTransition{db: db},
			AllowSource: SourceOutside,
		},
	}

	return transitions
}
