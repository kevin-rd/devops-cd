package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"devops-cd/internal/dto"
	"devops-cd/internal/service"
	"devops-cd/pkg/utils"
)

type ProjectHandler struct {
	projectService service.ProjectService
}

func NewProjectHandler(projectService service.ProjectService) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
	}
}

// Create 创建项目
// @Summary 创建项目
// @Tags Project
// @Accept json
// @Produce json
// @Param request body dto.CreateProjectRequest true "创建项目请求"
// @Success 200 {object} utils.Response{data=dto.ProjectResponse}
// @Router /api/v1/projects [post]
func (h *ProjectHandler) Create(c *gin.Context) {
	var req dto.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	project, err := h.projectService.Create(&req)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, project)
}

// GetByID 获取项目详情
// @Summary 获取项目详情
// @Tags Project
// @Accept json
// @Produce json
// @Param id query int64 true "项目ID"
// @Success 200 {object} utils.Response{data=dto.ProjectResponse}
// @Router /api/v1/projects/detail [get]
func (h *ProjectHandler) GetByID(c *gin.Context) {
	var req dto.GetProjectRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	project, err := h.projectService.GetByID(req.ID)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, project)
}

// List 获取项目列表
// @Summary 获取项目列表
// @Tags Project
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param keyword query string false "关键字搜索"
// @Success 200 {object} utils.Response{data=dto.PaginatedResponse}
// @Router /api/v1/projects [get]
func (h *ProjectHandler) List(c *gin.Context) {
	var query dto.ProjectListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	projects, total, err := h.projectService.List(&query)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.PageSuccess(c, projects, total, query.GetPage(), query.GetPageSize())
}

// ListAll 获取所有项目（用于下拉选择）
// @Summary 获取所有项目
// @Tags Project
// @Accept json
// @Produce json
// @Success 200 {object} utils.Response{data=[]dto.ProjectSimpleResponse}
// @Router /api/v1/projects/all [get]
func (h *ProjectHandler) ListAll(c *gin.Context) {
	projects, err := h.projectService.ListAll()
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, projects)
}

// Update 更新项目
// @Summary 更新项目
// @Tags Project
// @Accept json
// @Produce json
// @Param request body dto.UpdateProjectRequest true "更新项目请求"
// @Success 200 {object} utils.Response{data=dto.ProjectResponse}
// @Router /api/v1/projects [put]
func (h *ProjectHandler) Update(c *gin.Context) {
	var req dto.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	project, err := h.projectService.Update(req.ID, &req)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, project)
}

// Delete 删除项目
// @Summary 删除项目
// @Tags Project
// @Accept json
// @Produce json
// @Param id path int64 true "项目ID"
// @Success 200 {object} utils.Response
// @Router /api/v1/projects/{id} [delete]
func (h *ProjectHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "无效的项目ID", err.Error())
		return
	}

	if err := h.projectService.Delete(id); err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, nil)
}
