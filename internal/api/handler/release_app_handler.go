package handler

import (
	"net/http"

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
// @Success 200 {object} utils.Response{data=dto.ReleaseAppResponse}
// @Router /api/v1/release_app [get]
func (h *ReleaseAppHandler) GetByID(c *gin.Context) {
	var req dto.GetReleaseAppRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	resp, err := h.batchService.GetReleaseApp(req.ID)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, resp)
}

// UpdateDependencies 更新发布应用临时依赖
// @Summary 更新发布应用临时依赖
// @Tags ReleaseApp
// @Accept json
// @Produce json
// @Param id path int true "ReleaseApp ID"
// @Param body body dto.UpdateReleaseDependenciesRequest true "依赖更新请求"
// @Success 200 {object} utils.Response{data=dto.ReleaseDependenciesResponse}
// @Router /api/v1/release_app/{id}/dependencies [put]
func (h *ReleaseAppHandler) UpdateDependencies(c *gin.Context) {
	releaseID, ok := parseIDParam(c.Param("id"))
	if !ok {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "发布应用ID无效", c.Param("id"))
		return
	}

	var req dto.UpdateReleaseDependenciesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	req.ReleaseAppID = releaseID

	resp, err := h.batchService.UpdateReleaseDependencies(&req)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, resp)
}

// SwitchVersion 切换版本(更新版本)
// @Summary 切换版本
// @Tags ReleaseApp
// @Accept json
// @Produce json
// @Param body dto.SwitchVersionRequest true "切换请求"
// @Success 200 {object} utils.Response{data=dto.TriggerDeployResponse}
// @Router /api/v1/release_app/trigger_deploy [post]
func (h *BatchHandler) SwitchVersion(c *gin.Context) {
	var req dto.SwitchVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	resp, err := h.coreEngine.SwitchVersion(&req)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, resp)
}

func (h *BatchHandler) ManualDeploy(c *gin.Context) {
	var req dto.ManualDeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	resp, err := h.coreEngine.ManualDeploy(&req)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, resp)
}
