package handler

import (
	"devops-cd/pkg/responses"
	"github.com/gin-gonic/gin"

	"devops-cd/internal/dto"
	"devops-cd/internal/service"
	"devops-cd/pkg/utils"
)

type RepositoryHandler struct {
	service service.RepositoryService
}

func NewRepositoryHandler(service service.RepositoryService) *RepositoryHandler {
	return &RepositoryHandler{
		service: service,
	}
}

// Create 创建代码库
// @Summary 创建代码库
// @Tags Repository
// @Accept json
// @Produce json
// @Param body body dto.CreateRepositoryRequest true "创建代码库请求"
// @Success 200 {object} utils.Response{data=dto.RepositoryResponse}
// @Router /api/v1/repository [post]
func (h *RepositoryHandler) Create(c *gin.Context) {
	var req dto.CreateRepositoryRequest
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

// GetByID 获取代码库详情
// @Summary 获取代码库详情
// @Tags Repository
// @Accept json
// @Produce json
// @Param id query int true "代码库ID"
// @Success 200 {object} utils.Response{data=dto.RepositoryResponse}
// @Router /api/v1/repository [get]
func (h *RepositoryHandler) GetByID(c *gin.Context) {
	var req dto.GetRepositoryRequest
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

// List 获取代码库列表
// @Summary 获取代码库列表
// @Tags Repository
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param project query string false "项目名称"
// @Param team_id query int false "团队ID"
// @Param git_type query string false "Git类型"
// @Param keyword query string false "关键字"
// @Param status query int false "状态"
// @Param with_applications query bool false "是否包含应用列表"
// @Success 200 {object} utils.Response{data=dto.PageResponse}
// @Router /api/v1/repositories [get]
func (h *RepositoryHandler) List(c *gin.Context) {
	var query dto.RepositoryListQuery
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

// Update 更新代码库
// @Summary 更新代码库
// @Tags Repository
// @Accept json
// @Produce json
// @Param body body dto.UpdateRepositoryRequest true "更新代码库请求"
// @Success 200 {object} utils.Response{data=dto.RepositoryResponse}
// @Router /api/v1/repository [put]
func (h *RepositoryHandler) Update(c *gin.Context) {
	var req dto.UpdateRepositoryRequest
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

// Delete 删除代码库（软删除）
// @Summary 删除代码库
// @Tags Repository
// @Accept json
// @Produce json
// @Param body body dto.DeleteRepositoryRequest true "删除代码库请求"
// @Success 200 {object} utils.Response
// @Router /api/v1/repository/delete [post]
func (h *RepositoryHandler) Delete(c *gin.Context) {
	var req dto.DeleteRepositoryRequest
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
