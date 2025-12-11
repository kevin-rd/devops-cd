package constants

import "fmt"

// BatchStatus 批次状态
const (
	BatchStatusDraft  int8 = 0  // 草稿/准备中
	BatchStatusSealed int8 = 10 // 已封板

	BatchStatusPreWaiting   int8 = 20 // 预发布已触发
	BatchStatusPreDeploying int8 = 21 // 预发布部署中
	BatchStatusPreDeployed  int8 = 22 // 预发布已部署完成, 验收中
	BatchStatusPreFailed    int8 = 24
	BatchStatusPreAccepted  int8 = 25 // 预发布已验收(需要测试验收)

	BatchStatusProdWaiting   int8 = 30 // 生产已触发
	BatchStatusProdDeploying int8 = 31 // 生产部署中
	BatchStatusProdDeployed  int8 = 32 // 生产已部署完成, 验收中
	BatchStatusProdFailed    int8 = 34
	BatchStatusProdAccepted  int8 = 35 // 生产已验收(需要测试验收)

	BatchStatusCompleted     int8 = 40 // 已完成
	BatchStatusFinalAccepted int8 = 40
	BatchStatusCancelled     int8 = 90 // 已取消
)

// int8 → string
var batchStatusName = map[int8]string{
	BatchStatusDraft:         "Draft",
	BatchStatusSealed:        "Sealed",
	BatchStatusPreWaiting:    "PreWaiting",
	BatchStatusPreDeploying:  "PreDeploying",
	BatchStatusPreDeployed:   "PreDeployed",
	BatchStatusProdWaiting:   "ProdWaiting",
	BatchStatusProdDeploying: "ProdDeploying",
	BatchStatusProdDeployed:  "ProdDeployed",
	BatchStatusCompleted:     "Completed",
	BatchStatusCancelled:     "Cancelled",
}

// BatchStatusToString int8 → string
func BatchStatusToString(status int8) string {
	if name, ok := batchStatusName[status]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(%d)", status)
}

const (
	BatchActionSeal      = "seal"
	BatchActionStartPre  = "start_pre_deploy"
	BatchActionStartProd = "start_prod_deploy"
	BatchActionComplete  = "complete"
	BatchActionCancel    = "cancel"
)
