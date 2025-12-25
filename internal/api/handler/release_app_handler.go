package handler

import (
	"devops-cd/internal/pkg/logger"
	"devops-cd/pkg/responses"
	"net/http"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"

	"devops-cd/internal/dto"
	"devops-cd/internal/service"
	"devops-cd/pkg/utils"
)

// ReleaseAppHandler 发布应用处理器
type ReleaseAppHandler struct {
	batchService *service.BatchService
}

// NewReleaseAppHandler 创建发布应用处理器
func NewReleaseAppHandler(batchService *service.BatchService) *ReleaseAppHandler {
	return &ReleaseAppHandler{batchService: batchService}
}

// GetByID 获取发布应用详情
// @Summary 获取发布应用详情
// @Tags ReleaseApp
// @Accept json
// @Produce json
// @Param body body dto.GetReleaseAppRequest true "发布应用ID"
// @Success 200 {object} responses.Response{data=dto.ReleaseAppResponse}
// @Router /api/v1/release_app [get]
func (h *ReleaseAppHandler) GetByID(c *gin.Context) {
	var req dto.GetReleaseAppRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		responses.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	resp, err := h.batchService.GetReleaseApp(req.ID)
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, resp)
}

// UpdateBuilds 更新批次发布应用
// @Summary 更新批次发布应用
// @Description 批量更新批次中应用的构建版本等信息（仅草稿状态可修改）
// @Tags 批次管理
// @Accept json
// @Produce json
// @Param request body dto.UpdateBuildsRequest true "更新请求"
// @Success 200 {object} map[string]interface{} "更新成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 403 {object} map[string]interface{} "批次状态不允许修改"
// @Failure 500 {object} map[string]interface{} "更新失败"
// @Router /api/v1/batch/release_app [put]
func (h *ReleaseAppHandler) UpdateBuilds(c *gin.Context) {
	var req dto.UpdateBuildsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	err := h.batchService.UpdateBuilds(&req)
	if err != nil {
		logger.Error("更新批次应用构建失败", zap.Int64("batch_id", req.BatchID), zap.Error(err))

		// 根据错误类型返回不同的HTTP状态码
		if err.Error() == "只能修改草稿状态的批次" {
			responses.ErrorWithCode(c, http.StatusForbidden, err.Error())
		} else {
			responses.ErrorWithCode(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	responses.Success(c, gin.H{
		"message":      "批次应用构建更新成功",
		"batch_id":     req.BatchID,
		"update_count": len(req.BuildChanges),
	})
}

// UpdateDependencies 更新发布应用临时依赖
// @Summary 更新发布应用临时依赖
// @Tags ReleaseApp
// @Accept json
// @Produce json
// @Param id path int true "ReleaseApp ID"
// @Param body body dto.UpdateReleaseDependenciesRequest true "依赖更新请求"
// @Success 200 {object} responses.Response{data=dto.ReleaseDependenciesResponse}
// @Router /api/v1/release_app/{id}/dependencies [put]
func (h *ReleaseAppHandler) UpdateDependencies(c *gin.Context) {
	releaseID, ok := parseIDParam(c.Param("id"))
	if !ok {
		responses.ErrorWithDetail(c, http.StatusBadRequest, "发布应用ID无效", c.Param("id"))
		return
	}

	var req dto.UpdateReleaseDependenciesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	req.ReleaseAppID = releaseID

	resp, err := h.batchService.UpdateReleaseDependencies(&req)
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, resp)
}

// SwitchVersion 切换版本(更新版本)
// @Summary 切换版本
// @Tags ReleaseApp
// @Accept json
// @Produce json
// @Param request body dto.SwitchVersionRequest true "切换请求"
// @Success 200 {object} responses.Response{data=string}
// @Router /api/v1/release_app/trigger_deploy [post]
func (h *BatchHandler) SwitchVersion(c *gin.Context) {
	var req dto.SwitchVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	resp, err := h.coreEngine.SwitchVersion(&req)
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, resp)
}

func (h *BatchHandler) ManualDeploy(c *gin.Context) {
	var req dto.ManualDeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	resp, err := h.coreEngine.ManualDeploy(&req)
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, resp)
}
