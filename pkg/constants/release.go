package constants

// ReleaseAppStatus 应用发布状态
const (
	ReleaseAppStatusPending        int8 = 0  // 初始化
	ReleaseAppStatusTagged         int8 = 10 // 应用已打tag并发版成功
	ReleaseAppStatusPreWaiting     int8 = 20 // Pre 等待被触发
	ReleaseAppStatusPreCanTrigger  int8 = 21 // Pre 可以触发
	ReleaseAppStatusPreTriggered   int8 = 22 // Pre 均已触发
	ReleaseAppStatusPreDeployed    int8 = 23 // Pre 均部署完成
	ReleaseAppStatusPreFailed      int8 = 24
	ReleaseAppStatusProdWaiting    int8 = 30 // Prod 等待被触发
	ReleaseAppStatusProdCanTrigger int8 = 31 // Prod 可以触发
	ReleaseAppStatusProdTriggered  int8 = 32 // Prod 均已触发
	ReleaseAppStatusProdDeployed   int8 = 33 // Prod 均部署完成
	ReleaseAppStatusProdFailed     int8 = 34
)

// DeploymentStatus 部署状态
const (
	DeploymentStatusPending = "pending"
	DeploymentStatusRunning = "running"
	DeploymentStatusSuccess = "success"
	DeploymentStatusFailed  = "failed"
)
