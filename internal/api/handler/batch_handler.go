package handler

import (
	pkgErrors "devops-cd/pkg/errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"

	"devops-cd/internal/core"
	"devops-cd/internal/dto"
	"devops-cd/internal/pkg/logger"
	"devops-cd/internal/service"
	"devops-cd/pkg/constants"
	"devops-cd/pkg/utils"
)

// BatchHandler 批次处理器
type BatchHandler struct {
	coreEngine   *core.CoreEngine
	batchService *service.BatchService
}

// NewBatchHandler 创建批次处理器
func NewBatchHandler(coreEngine *core.CoreEngine, batchService *service.BatchService) *BatchHandler {
	return &BatchHandler{
		coreEngine:   coreEngine,
		batchService: batchService,
	}
}

// ProcessActionRequest 处理批次操作请求
type ProcessActionRequest struct {
	BatchID  int64  `json:"batch_id" binding:"required"` // 批次ID
	Action   string `json:"action" binding:"required"`   // 操作类型: seal/start_pre_deploy/finish_pre_deploy/start_prod_deploy/finish_prod_deploy/complete/cancel
	Operator string `json:"operator" binding:"required"` // 操作人
	Reason   string `json:"reason"`                      // 原因（可选）
}

// ProcessAction 处理批次状态操作
// @Summary 批次状态操作
// @Description 处理批次的各种状态操作，如封板、部署、验收等。支持的action: seal(封板)/start_pre_deploy(开始预发布)/finish_pre_deploy(完成预发布)/start_prod_deploy(开始生产部署)/finish_prod_deploy(完成生产部署)/complete(最终验收)/cancel(取消)
// @Tags 批次管理
// @Accept json
// @Produce json
// @Param request body ProcessActionRequest true "操作请求"
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "操作失败"
// @Security BearerAuth
// @Router /api/v1/batch/action [post]
func (h *BatchHandler) ProcessAction(c *gin.Context) {
	var req ProcessActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	// 处理操作
	if err := h.coreEngine.ProcessBatchEvent(req.BatchID, req.Action, req.Operator, req.Reason); err != nil {
		logger.Error("处理批次操作失败",
			zap.Int64("batch_id", req.BatchID),
			zap.String("action", req.Action),
			zap.Error(err))
		utils.ErrorWithCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	logger.Info("批次操作处理成功",
		zap.Int64("batch_id", req.BatchID),
		zap.String("action", req.Action),
		zap.String("operator", req.Operator))

	utils.Success(c, gin.H{
		"message": "操作成功",
		"action":  req.Action,
	})
}

// ApproveRequest 审核通过请求
type ApproveRequest struct {
	BatchID  int64  `json:"batch_id" binding:"required"` // 批次ID
	Operator string `json:"operator" binding:"required"` // 审核人
	Reason   string `json:"reason"`                      // 审核意见
}

// Approve 审批通过批次
// @Summary 批次审批通过
// @Description 审批通过批次，更新审批状态为approved
// @Tags 批次管理
// @Accept json
// @Produce json
// @Param request body ApproveRequest true "审批请求"
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "审批失败"
// @Security BearerAuth
// @Router /api/v1/batch/approve [post]
func (h *BatchHandler) Approve(c *gin.Context) {
	var req ApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	// 调用 service 层处理审批
	if err := h.batchService.ApproveBatch(req.BatchID, req.Operator, req.Reason); err != nil {
		utils.ErrorWithCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "审核通过"})
}

// RejectRequest 拒绝请求
type RejectRequest struct {
	BatchID  int64  `json:"batch_id" binding:"required"` // 批次ID
	Operator string `json:"operator" binding:"required"` // 审核人
	Reason   string `json:"reason" binding:"required"`   // 拒绝原因
}

// Reject 审批拒绝批次
// @Summary 批次审批拒绝
// @Description 审批拒绝批次，更新审批状态为rejected
// @Tags 批次管理
// @Accept json
// @Produce json
// @Param request body RejectRequest true "拒绝请求"
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "拒绝失败"
// @Security BearerAuth
// @Router /api/v1/batch/reject [post]
func (h *BatchHandler) Reject(c *gin.Context) {
	var req RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	// 调用 service 层处理拒绝
	if err := h.batchService.RejectBatch(req.BatchID, req.Operator, req.Reason); err != nil {
		utils.ErrorWithCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "已拒绝"})
}

// ============== 批次管理接口（新增） ==============

// Create 创建批次
// @Summary 创建批次
// @Description 创建新的发布批次，自动扫描指定应用的最新成功构建
// @Tags 批次管理
// @Accept json
// @Produce json
// @Param request body service.CreateBatchRequest true "批次创建请求"
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 409 {object} map[string]interface{} "应用冲突"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Security BearerAuth
// @Router /api/v1/batch [post]
func (h *BatchHandler) Create(c *gin.Context, canAccess func(username string, projectId int64) bool) {
	var req dto.CreateBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	param := req.ToParam()
	param.Operator = c.GetString("username")
	param.ProjectID = req.ProjectID
	param.CanCreate = canAccess

	// 检查权限
	username := c.GetString("username")
	if !canAccess(username, req.ProjectID) {
		utils.Error(c, pkgErrors.ErrForbidden)
		return
	}

	batch, err := h.batchService.CreateBatch(&param)
	if err != nil {
		// 处理应用冲突错误
		if conflictErr, ok := err.(*service.AppConflictError); ok {
			conflicts := make([]gin.H, 0)
			for appID, conflictBatch := range conflictErr.Conflicts {
				app := conflictErr.AppMap[appID]
				appName := ""
				appProject := ""
				if app != nil {
					appName = app.Name
					// 从 Repository 获取 namespace 作为 project
					if app.Repository != nil {
						appProject = app.Repository.Namespace
					}
				}

				statusName := getStatusName(conflictBatch.Status)

				conflicts = append(conflicts, gin.H{
					"app_id":            appID,
					"app_name":          appName,
					"app_project":       appProject,
					"batch_id":          conflictBatch.ID,
					"batch_number":      conflictBatch.BatchNumber,
					"batch_status":      conflictBatch.Status,
					"batch_status_name": statusName,
				})
			}

			// 只返回一次响应，包含详细冲突信息
			c.JSON(http.StatusConflict, gin.H{
				"code":    http.StatusConflict,
				"message": fmt.Sprintf("存在应用冲突，有 %d 个应用已在其他批次中", len(conflicts)),
				"data": gin.H{
					"conflicts": conflicts,
				},
			})
			return
		}

		logger.Error("创建批次失败", zap.Error(err))
		utils.ErrorWithCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, gin.H{
		"batch_id":     batch.ID,
		"batch_number": batch.BatchNumber,
		"message":      "批次创建成功",
	})
}

// Update 更新批次
// @Summary 更新批次
// @Description 更新批次基本信息、添加或删除应用
// @Tags 批次管理
// @Accept json
// @Produce json
// @Param request body service.UpdateBatchRequest true "批次更新请求"
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 403 {object} map[string]interface{} "批次已封板"
// @Failure 409 {object} map[string]interface{} "应用冲突"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Security BearerAuth
// @Router /api/v1/batch [put]
func (h *BatchHandler) Update(c *gin.Context, canAccess func(username string, projectId int64) bool) {
	var req dto.UpdateBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	param := req.ToParam()
	param.Operator = c.GetString("username")
	param.CanUpdate = canAccess

	batch, updatedFields, err := h.batchService.UpdateBatch(&param)
	if err != nil {
		// 处理批次已封板错误
		if sealedErr, ok := err.(*service.BatchSealedError); ok {
			statusName := getStatusName(sealedErr.Status)
			c.JSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "批次已封板，不允许修改",
				"data": gin.H{
					"batch_id":    sealedErr.BatchID,
					"status":      sealedErr.Status,
					"status_name": statusName,
				},
			})
			return
		}

		// 处理应用冲突错误
		if conflictErr, ok := err.(*service.AppConflictError); ok {
			conflicts := make([]gin.H, 0)
			for appID, conflictBatch := range conflictErr.Conflicts {
				app := conflictErr.AppMap[appID]
				appName := ""
				appProject := ""
				if app != nil {
					appName = app.Name
					// 从 Repository 获取 namespace 作为 project
					if app.Repository != nil {
						appProject = app.Repository.Namespace
					}
				}

				statusName := getStatusName(conflictBatch.Status)

				conflicts = append(conflicts, gin.H{
					"app_id":            appID,
					"app_name":          appName,
					"app_project":       appProject,
					"batch_id":          conflictBatch.ID,
					"batch_number":      conflictBatch.BatchNumber,
					"batch_status":      conflictBatch.Status,
					"batch_status_name": statusName,
				})
			}

			c.JSON(http.StatusConflict, gin.H{
				"code":    http.StatusConflict,
				"message": fmt.Sprintf("存在应用冲突，有 %d 个应用已在其他批次中", len(conflicts)),
				"data": gin.H{
					"conflicts": conflicts,
				},
			})
			return
		}

		logger.Error("更新批次失败", zap.Error(err))
		utils.ErrorWithCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, gin.H{
		"batch_id":       batch.ID,
		"batch_number":   batch.BatchNumber,
		"updated_fields": updatedFields,
		"message":        "批次更新成功",
	})
}

// Delete 删除批次
// POST /api/v1/batch/delete
func (h *BatchHandler) Delete(c *gin.Context) {
	var req struct {
		BatchID  int64  `json:"batch_id" binding:"required"`
		Operator string `json:"operator" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	if err := h.batchService.DeleteBatch(req.BatchID, req.Operator); err != nil {
		logger.Error("删除批次失败",
			zap.Int64("batch_id", req.BatchID),
			zap.Error(err))
		utils.ErrorWithCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "批次删除成功"})
}

// Get 获取批次详情
// @Summary 获取批次详情
// @Description 获取批次详细信息和应用列表（支持应用列表分页）。每个应用默认会包含自上次部署以来的最近15条成功构建记录，可通过 with_recent_builds=false 关闭。
// @Tags 批次管理
// @Accept json
// @Produce json
// @Param id query int64 true "批次ID"
// @Param app_page query int false "应用列表页码" default(1)
// @Param app_page_size query int false "应用列表每页数量" default(20)
// @Param with_recent_builds query bool false "是否包含最近构建记录" default(true)
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Security BearerAuth
// @Router /api/v1/batch [get]
func (h *BatchHandler) Get(c *gin.Context) {
	var req dto.BatchGetRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	response, err := h.batchService.GetBatch(req.ID, req.GetAppPage(), req.GetAppPageSize(), req.GetWithRecentBuilds())
	if err != nil {
		logger.Error("获取批次详情失败", zap.Int64("batch_id", req.ID), zap.Error(err))
		utils.ErrorWithCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, response)
}

// List 查询批次列表
// @Summary 查询批次列表
// @Description 分页查询批次列表，支持状态、发起人、审批状态、时间范围、关键字过滤。status支持多值，例如：?status=1&status=2&status=3
// @Tags 批次管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param status query []int8 false "状态过滤（支持多个）"
// @Param initiator query string false "发起人过滤"
// @Param approval_status query string false "审批状态过滤：pending/approved/rejected"
// @Param created_at_start query string false "创建时间起始（RFC3339格式）"
// @Param created_at_end query string false "创建时间结束（RFC3339格式）"
// @Param keyword query string false "关键字搜索（批次编号、发起人、发布说明）"
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Security BearerAuth
// @Router /api/v1/batches [get]
func (h *BatchHandler) List(c *gin.Context) {
	var req dto.BatchListQuery

	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	param := req.ToParam()
	responses, total, err := h.batchService.ListBatches(param)
	if err != nil {
		logger.Error("查询批次列表失败", zap.Error(err))
		utils.ErrorWithCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, dto.NewPageResponse(responses, total, param.Page, param.PageSize))
}

// GetStatus 获取批次状态（轻量级，用于状态轮询）
// @Summary 获取批次状态
// @Description 获取批次状态信息（轻量级接口，专门用于状态轮询）。只返回批次和应用的状态信息，不包含构建历史、依赖等详细数据。
// @Tags 批次管理
// @Accept json
// @Produce json
// @Param id query int64 true "批次ID"
// @Param app_page query int false "应用列表页码" default(1)
// @Param app_page_size query int false "应用列表每页数量" default(20)
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Security BearerAuth
// @Router /api/v1/batch/status [get]
func (h *BatchHandler) GetStatus(c *gin.Context) {
	var req dto.BatchStatusRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	response, err := h.batchService.GetBatchStatus(req.ID, req.GetAppPage(), req.GetAppPageSize())
	if err != nil {
		logger.Error("获取批次状态失败", zap.Int64("batch_id", req.ID), zap.Error(err))
		utils.ErrorWithCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, response)
}

// getStatusName 获取状态名称（仅用于错误响应）
func getStatusName(status int8) string {
	switch status {
	case constants.BatchStatusDraft:
		return "草稿"
	case constants.BatchStatusSealed:
		return "已封板"
	case constants.BatchStatusPreDeploying:
		return "预发布部署中"
	case constants.BatchStatusPreDeployed:
		return "预发布已部署"
	case constants.BatchStatusProdDeploying:
		return "生产部署中"
	case constants.BatchStatusProdDeployed:
		return "生产已部署"
	case constants.BatchStatusCompleted:
		return "已完成"
	case constants.BatchStatusCancelled:
		return "已取消"
	default:
		return "未知状态"
	}
}
