package handler

import (
	"devops-cd/pkg/responses"
	"strconv"

	"github.com/gin-gonic/gin"

	"devops-cd/internal/dto"
	"devops-cd/internal/service"
	"devops-cd/pkg/utils"
)

type ApplicationHandler struct {
	service service.ApplicationService
}

func NewApplicationHandler(service service.ApplicationService) *ApplicationHandler {
	return &ApplicationHandler{
		service: service,
	}
}

// Create 创建应用
// @Summary 创建应用
// @Tags Application
// @Accept json
// @Produce json
// @Param body body dto.CreateApplicationRequest true "创建应用请求"
// @Success 200 {object} responses.Response{data=dto.ApplicationResponse}
// @Router /api/v1/applications [post]
func (h *ApplicationHandler) Create(c *gin.Context) {
	var req dto.CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.ErrorWithDetail(c, responses.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	resp, err := h.service.Create(&req)
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, resp)
}

// GetByID 获取应用详情
// @Summary 获取应用详情
// @Tags Application
// @Accept json
// @Produce json
// @Param id query int true "应用ID"
// @Success 200 {object} responses.Response{data=dto.ApplicationResponse}
// @Router /api/v1/application [get]
func (h *ApplicationHandler) GetByID(c *gin.Context) {
	var req dto.GetApplicationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		responses.ErrorWithDetail(c, responses.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	resp, err := h.service.GetByID(req.ID)
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, resp)
}

// List 获取应用列表
// @Summary 获取应用列表
// @Tags Application
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param repo_id query int false "代码库ID"
// @Param team_id query int false "团队ID"
// @Param app_type query string false "应用类型"
// @Param keyword query string false "关键字"
// @Param status query int false "状态"
// @Success 200 {object} responses.Response{data=dto.PageResponse}
// @Router /api/v1/applications [get]
func (h *ApplicationHandler) List(c *gin.Context) {
	var query dto.ApplicationListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		responses.ErrorWithDetail(c, responses.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	data, total, err := h.service.List(&query)
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, dto.NewPageResponse(data, total, query.GetPage(), query.GetPageSize()))
}

// Update 更新应用
// @Summary 更新应用
// @Tags Application
// @Accept json
// @Produce json
// @Param body body dto.UpdateApplicationRequest true "更新应用请求"
// @Success 200 {object} responses.Response{data=dto.ApplicationResponse}
// @Router /api/v1/application [put]
func (h *ApplicationHandler) Update(c *gin.Context) {
	var req dto.UpdateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.ErrorWithDetail(c, responses.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	resp, err := h.service.Update(req.ID, &req)
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, resp)
}

// Delete 删除应用（软删除）
// @Summary 删除应用
// @Tags Application
// @Accept json
// @Produce json
// @Param body body dto.DeleteApplicationRequest true "删除应用请求"
// @Success 200 {object} responses.Response
// @Router /api/v1/application/delete [post]
func (h *ApplicationHandler) Delete(c *gin.Context) {
	var req dto.DeleteApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.ErrorWithDetail(c, responses.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	if err := h.service.Delete(req.ID); err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, nil)
}

// GetBuilds 获取应用的构建历史
// @Summary 获取应用的构建历史
// @Tags Application
// @Accept json
// @Produce json
// @Param id query int true "应用ID"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} responses.Response{data=dto.PageResponse}
// @Router /api/v1/application/builds [get]
func (h *ApplicationHandler) GetBuilds(c *gin.Context) {
	var req dto.GetApplicationBuildsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		responses.ErrorWithDetail(c, responses.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	builds, total, err := h.service.GetBuilds(req.ID, req.GetPage(), req.GetPageSize())
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, dto.NewPageResponse(builds, total, req.GetPage(), req.GetPageSize()))
}

// GetAppTypes 获取应用类型列表
// @Summary 获取应用类型列表
// @Description 获取所有可用的应用类型及其元数据（显示名称、描述、图标、颜色等）
// @Tags Application
// @Accept json
// @Produce json
// @Success 200 {object} responses.Response{data=dto.AppTypesResponse}
// @Router /api/v1/application/types [get]
func (h *ApplicationHandler) GetAppTypes(c *gin.Context) {
	resp, err := h.service.GetAppTypes()
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, resp)
}

// SearchWithBuilds 搜索应用（包含构建信息，支持模糊查询）
// @Summary 搜索应用（包含构建信息）
// @Description 搜索应用列表，支持模糊查询 app、repo、commit、tag 等字段，返回包含最新构建信息的应用列表，按构建时间倒序排序
// @Tags Application
// @Accept json
// @Produce json
// @Param keyword query string false "模糊查询关键字（支持查询 app、repo、commit、tag 等字段）"
// @Param page query int false "页码（默认1）"
// @Param page_size query int false "每页数量（默认20，最大100）"
// @Param repo_id query int false "按代码库ID过滤"
// @Param team_id query int false "按团队ID过滤"
// @Param app_type query string false "按应用类型过滤"
// @Param status query int false "按状态过滤（0/1）"
// @Success 200 {object} responses.Response{data=dto.PageResponse}
// @Router /api/v1/application_builds [get]
func (h *ApplicationHandler) SearchWithBuilds(c *gin.Context) {
	var query dto.ApplicationSearchQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		responses.ErrorWithDetail(c, responses.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	param := query.ToParam()
	data, total, err := h.service.SearchWithBuilds(&param)
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, dto.NewPageResponse(data, total, query.GetPage(), query.GetPageSize()))
}

// GetDependencies 获取应用默认依赖
// @Summary 获取应用默认依赖
// @Tags Application
// @Produce json
// @Param id path int true "应用ID"
// @Success 200 {object} responses.Response{data=dto.ApplicationDependenciesResponse}
// @Router /api/v1/application/{id}/dependencies [get]
func (h *ApplicationHandler) GetDependencies(c *gin.Context) {
	id, ok := parseIDParam(c.Param("id"))
	if !ok {
		responses.ErrorWithDetail(c, responses.CodeBadRequest, "应用ID无效", c.Param("id"))
		return
	}

	resp, err := h.service.GetDefaultDependencies(id)
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, resp)
}

// UpdateDependencies 更新应用默认依赖
// @Summary 更新应用默认依赖
// @Tags Application
// @Accept json
// @Produce json
// @Param id path int true "应用ID"
// @Param body body dto.UpdateAppDependenciesRequest true "更新依赖请求"
// @Success 200 {object} responses.Response{data=dto.ApplicationDependenciesResponse}
// @Router /api/v1/application/{id}/dependencies [put]
func (h *ApplicationHandler) UpdateDependencies(c *gin.Context) {
	id, ok := parseIDParam(c.Param("id"))
	if !ok {
		responses.ErrorWithDetail(c, responses.CodeBadRequest, "应用ID无效", c.Param("id"))
		return
	}

	var req dto.UpdateAppDependenciesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.ErrorWithDetail(c, responses.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	resp, err := h.service.UpdateDefaultDependencies(id, &req)
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, resp)
}

func parseIDParam(raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
