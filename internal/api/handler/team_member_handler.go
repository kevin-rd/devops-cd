package handler

import (
	"devops-cd/pkg/responses"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"devops-cd/internal/dto"
	"devops-cd/internal/service"
)

type TeamMemberHandler struct {
	service service.TeamMemberService
}

func NewTeamMemberHandler(service service.TeamMemberService) *TeamMemberHandler {
	return &TeamMemberHandler{
		service: service,
	}
}

// AddMember 添加团队成员
func (h *TeamMemberHandler) AddMember(c *gin.Context) {
	var req dto.TeamMemberAddRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	member, err := h.service.Add(&req)
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, member)
}

// ListMembers 获取团队成员列表
func (h *TeamMemberHandler) ListMembers(c *gin.Context) {
	var req dto.TeamMemberListQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		responses.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	members, total, err := h.service.List(&req)
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, dto.NewPageResponse(members, total, req.GetPage(), req.GetPageSize()))
}

// UpdateRole 更新团队成员角色
func (h *TeamMemberHandler) UpdateRole(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		responses.ErrorWithDetail(c, http.StatusBadRequest, "无效的成员ID", err.Error())
		return
	}

	var req dto.TeamMemberUpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	member, err := h.service.UpdateRole(id, &req)
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, member)
}

// DeleteMember 删除团队成员
func (h *TeamMemberHandler) DeleteMember(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		responses.ErrorWithDetail(c, http.StatusBadRequest, "无效的成员ID", err.Error())
		return
	}

	if err := h.service.Remove(id); err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, nil)
}
