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
// @Summary 获取项目列表（无分页参数时返回所有项目，有分页参数时返回分页数据）
// @Tags Project
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param keyword query string false "关键字搜索"
// @Success 200 {object} utils.Response{data=[]dto.ProjectSimpleResponse}
// @Success 200 {object} utils.Response{data=dto.PaginatedResponse}
// @Router /api/v1/projects [get]
func (h *ProjectHandler) List(c *gin.Context) {
	var query dto.ProjectListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	// 如果没有分页参数，返回所有项目简化列表（用于下拉选择）
	if query.Page == 0 && query.PageSize == 0 {
		projects, err := h.projectService.ListAll()
		if err != nil {
			utils.Error(c, err)
			return
		}
		utils.Success(c, projects)
		return
	}

	// 有分页参数，返回分页数据
	projects, total, err := h.projectService.List(&query)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.PageSuccess(c, projects, total, query.GetPage(), query.GetPageSize())
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

// GetAvailableEnvClusters 获取项目可用的环境集群配置
// @Summary 获取项目可用的环境集群配置
// @Tags Project
// @Accept json
// @Produce json
// @Param project_id query int64 true "项目ID"
// @Param env query string false "环境名称(pre/prod),不传返回全部配置"
// @Success 200 {object} utils.Response{data=dto.ProjectAvailableEnvClustersResponse}
// @Router /api/v1/projects/available-env-clusters [get]
func (h *ProjectHandler) GetAvailableEnvClusters(c *gin.Context) {
	var req dto.GetProjectAvailableEnvClustersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	resp, err := h.projectService.GetAvailableEnvClusters(req.ProjectID, req.Env)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, resp)
}
