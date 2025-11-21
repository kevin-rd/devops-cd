package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"devops-cd/internal/dto"
	"devops-cd/internal/service"
	"devops-cd/pkg/utils"
)

type TeamHandler struct {
	teamService service.TeamService
}

func NewTeamHandler(teamService service.TeamService) *TeamHandler {
	return &TeamHandler{
		teamService: teamService,
	}
}

// Create 创建团队
// @Summary 创建团队
// @Tags Team
// @Accept json
// @Produce json
// @Param request body dto.CreateTeamRequest true "创建团队请求"
// @Success 200 {object} utils.Response{data=dto.TeamResponse}
// @Router /api/v1/team [post]
func (h *TeamHandler) Create(c *gin.Context) {
	var req dto.CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	team, err := h.teamService.Create(&req)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, team)
}

// GetByID 获取团队详情
// @Summary 获取团队详情
// @Tags Team
// @Accept json
// @Produce json
// @Param id query int64 true "团队ID"
// @Success 200 {object} utils.Response{data=dto.TeamResponse}
// @Router /api/v1/team [get]
func (h *TeamHandler) GetByID(c *gin.Context) {
	var req dto.GetTeamRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	team, err := h.teamService.GetByID(req.ID)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, team)
}

// List 获取团队列表
// @Summary 获取团队列表（返回所有团队，可按项目过滤，用于下拉选择）
// @Tags Team
// @Accept json
// @Produce json
// @Param project_id query int false "项目ID"
// @Success 200 {object} utils.Response{data=[]dto.TeamSimpleResponse}
// @Router /api/v1/teams [get]
func (h *TeamHandler) List(c *gin.Context) {
	var projectID *int64
	if idStr := c.Query("project_id"); idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			utils.ErrorWithDetail(c, http.StatusBadRequest, "无效的项目ID", err.Error())
			return
		}
		projectID = &id
	}

	teams, err := h.teamService.List(projectID)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, teams)
}

// Update 更新团队
// @Summary 更新团队
// @Tags Team
// @Accept json
// @Produce json
// @Param request body dto.UpdateTeamRequest true "更新团队请求"
// @Success 200 {object} utils.Response{data=dto.TeamResponse}
// @Router /api/v1/team [put]
func (h *TeamHandler) Update(c *gin.Context) {
	var req dto.UpdateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	team, err := h.teamService.Update(req.ID, &req)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, team)
}

// Delete 删除团队
// @Summary 删除团队
// @Tags Team
// @Accept json
// @Produce json
// @Param id path int64 true "团队ID"
// @Success 200 {object} utils.Response
// @Router /api/v1/team/{id} [delete]
func (h *TeamHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, "无效的团队ID", err.Error())
		return
	}

	if err := h.teamService.Delete(id); err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, nil)
}
