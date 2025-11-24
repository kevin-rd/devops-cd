package model

import (
	"time"
)

// Batch 发布批次
type Batch struct {
	BaseModel
	BatchNumber  string  `gorm:"size:200;not null" json:"batch_number"` // 用户填写的批次编号/标题
	ProjectID    int64   `gorm:"index;not null" json:"project_id"`      // 关联的项目ID
	Initiator    string  `gorm:"size:50" json:"initiator"`
	ReleaseNotes *string `gorm:"type:text" json:"release_notes"` // 批次级发布说明

	// 审批信息（独立于部署流程）
	ApprovalStatus string     `gorm:"size:20;index;not null;default:pending" json:"approval_status"` // pending/approved/rejected/skipped
	ApprovedBy     *string    `gorm:"size:50" json:"approved_by"`
	ApprovedAt     *time.Time `json:"approved_at"`
	RejectReason   *string    `gorm:"type:text" json:"reject_reason"`

	// 部署流程状态
	Status int8 `gorm:"index;not null;default:0" json:"status"` // pkg/constants:BatchStatus (0:草稿 10:已封板 21:预发布中...)

	// 封板/触发
	SealedBy        *string    `gorm:"size:50" json:"sealed_by"`                               // 封板人
	SealedAt        *time.Time `gorm:"column:sealed_at" json:"tagged_at"`                      // 封板时间
	PreTriggeredBy  *string    `gorm:"size:50" json:"pre_triggered_by"`                        // 预发布触发人
	PreStartedAt    *time.Time `gorm:"column:pre_started_at" json:"pre_deploy_started_at"`     // 预发布开始时间
	PreFinishedAt   *time.Time `gorm:"column:pre_finished_at" json:"pre_deploy_finished_at"`   // 预发布完成时间
	ProdTriggeredBy *string    `gorm:"size:50" json:"prod_triggered_by"`                       // 生产部署触发人
	ProdStartedAt   *time.Time `gorm:"column:prod_started_at" json:"prod_deploy_started_at"`   // 生产部署开始时间
	ProdFinishedAt  *time.Time `gorm:"column:prod_finished_at" json:"prod_deploy_finished_at"` // 生产部署完成时间

	// 验收和取消
	FinalAcceptedAt *time.Time `json:"final_accepted_at"`                // 最终验收时间
	FinalAcceptedBy *string    `gorm:"size:50" json:"final_accepted_by"` // 最终验收人
	CancelledAt     *time.Time `json:"cancelled_at"`                     // 取消时间
	CancelledBy     *string    `gorm:"size:50" json:"cancelled_by"`      // 取消人
	CancelReason    *string    `gorm:"type:text" json:"cancel_reason"`   // 取消原因
}

// TableName 指定表名
func (Batch) TableName() string {
	return "release_batches"
}

// ReleaseApp 批次中的应用发布记录（简约版）
type ReleaseApp struct {
	ID      int64 `gorm:"primaryKey" json:"id"`
	BatchID int64 `gorm:"index;not null" json:"batch_id"`
	AppID   int64 `gorm:"index;not null" json:"app_id"`

	// 构建关联（可空：允许无构建应用，封板时校验）
	BuildID *int64 `gorm:"index" json:"build_id"` // 关联的构建ID（封板时固定）

	// 版本信息
	PreviousDeployedTag *string `gorm:"column:previous_deployed_tag;size:100" json:"previous_deployed_tag"` // 部署前的版本（封板时从 applications.deployed_tag 获取）
	TargetTag           *string `gorm:"column:target_tag;size:100" json:"target_tag"`                       // 目标部署版本（封板时从 build.image_tag 获取并固定，部署期间代表期望版本，部署完成后代表已部署版本）
	LatestBuildID       *int64  `gorm:"column:latest_build_id" json:"latest_build_id"`                      // 最新检测到的构建ID（新tag到达时更新）

	// 业务字段
	ReleaseNotes  *string   `gorm:"type:text" json:"release_notes"`    // 应用级发布说明（可选）
	IsLocked      bool      `gorm:"default:false" json:"is_locked"`    // 是否已锁定（封板后为true）
	SkipPreEnv    bool      `gorm:"default:false" json:"skip_pre_env"` // 是否跳过预发布环境(封板时从 app_env_configs 计算得出)
	Status        int8      `gorm:"index;not null;default:0" json:"status"`
	Reason        string    `gorm:"type:text" json:"reason"`
	TempDependsOn Int64List `gorm:"column:temp_depends_on;type:json;default:[]" json:"temp_depends_on"` // 批次内临时依赖（JSON 数组，记录应用 ID）

	// 系统字段
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联关系（用于 JOIN 查询时获取完整构建信息）
	Batch       *Batch       `gorm:"foreignKey:BatchID" json:"batch,omitempty"`
	Application *Application `gorm:"foreignKey:AppID" json:"application,omitempty"`
	Build       *Build       `gorm:"foreignKey:BuildID" json:"build,omitempty"` // 通过 JOIN 获取完整构建信息
}

// TableName 指定表名
func (ReleaseApp) TableName() string {
	return "release_apps"
}
