package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"devops-cd/internal/dto"
	"devops-cd/internal/service"
	pkgErrors "devops-cd/pkg/errors"
	"devops-cd/pkg/utils"
)

type AppEnvConfigHandler struct {
	service service.AppEnvConfigService
}

func NewAppEnvConfigHandler(service service.AppEnvConfigService) *AppEnvConfigHandler {
	return &AppEnvConfigHandler{
		service: service,
	}
}

// Create 创建应用环境配置
// @Summary 创建应用环境配置
// @Tags AppEnvConfig
// @Accept json
// @Produce json
// @Param body body dto.CreateAppEnvConfigRequest true "创建请求"
// @Success 200 {object} utils.Response{data=dto.AppEnvConfigResponse}
// @Router /api/v1/app-env-configs [post]
func (h *AppEnvConfigHandler) Create(c *gin.Context) {
	var req dto.CreateAppEnvConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, pkgErrors.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	resp, err := h.service.Create(&req)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, resp)
}

// Update 更新应用环境配置
// @Summary 更新应用环境配置
// @Tags AppEnvConfig
// @Accept json
// @Produce json
// @Param id path int64 true "配置ID"
// @Param body body dto.UpdateAppEnvConfigRequest true "更新请求"
// @Success 200 {object} utils.Response{data=dto.AppEnvConfigResponse}
// @Router /api/v1/app-env-configs/{id} [put]
func (h *AppEnvConfigHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorWithDetail(c, pkgErrors.CodeBadRequest, "无效的配置ID", err.Error())
		return
	}

	var req dto.UpdateAppEnvConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, pkgErrors.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	req.ID = id
	resp, err := h.service.Update(id, &req)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, resp)
}

// Delete 删除应用环境配置
// @Summary 删除应用环境配置
// @Tags AppEnvConfig
// @Accept json
// @Produce json
// @Param id path int64 true "配置ID"
// @Success 200 {object} utils.Response
// @Router /api/v1/app-env-configs/{id} [delete]
func (h *AppEnvConfigHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorWithDetail(c, pkgErrors.CodeBadRequest, "无效的配置ID", err.Error())
		return
	}

	if err := h.service.Delete(id); err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, nil)
}

// GetByID 获取应用环境配置详情
// @Summary 获取应用环境配置详情
// @Tags AppEnvConfig
// @Accept json
// @Produce json
// @Param id path int64 true "配置ID"
// @Success 200 {object} utils.Response{data=dto.AppEnvConfigResponse}
// @Router /api/v1/app-env-configs/{id} [get]
func (h *AppEnvConfigHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorWithDetail(c, pkgErrors.CodeBadRequest, "无效的配置ID", err.Error())
		return
	}

	resp, err := h.service.GetByID(id)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, resp)
}

// List 查询应用环境配置列表
// @Summary 查询应用环境配置列表
// @Tags AppEnvConfig
// @Accept json
// @Produce json
// @Param app_id query int64 true "应用ID"
// @Param env query string false "环境名称(pre/prod/dev/test/uat)"
// @Success 200 {object} utils.Response{data=[]dto.AppEnvConfigResponse}
// @Router /api/v1/app-env-configs [get]
func (h *AppEnvConfigHandler) List(c *gin.Context) {
	var query dto.ListAppEnvConfigsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.ErrorWithDetail(c, pkgErrors.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	configs, err := h.service.List(&query)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, configs)
}

// BatchCreate 批量创建应用环境配置
// @Summary 批量创建应用环境配置
// @Tags AppEnvConfig
// @Accept json
// @Produce json
// @Param body body dto.BatchCreateAppEnvConfigsRequest true "批量创建请求"
// @Success 200 {object} utils.Response{data=[]dto.AppEnvConfigResponse}
// @Router /api/v1/app-env-configs/batch [post]
func (h *AppEnvConfigHandler) BatchCreate(c *gin.Context) {
	var req dto.BatchCreateAppEnvConfigsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, pkgErrors.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	configs, err := h.service.BatchCreate(&req)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, configs)
}
