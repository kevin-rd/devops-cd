package handler

import (
	"devops-cd/pkg/responses"
	"net/http"

	"github.com/gin-gonic/gin"

	"devops-cd/internal/dto"
	"devops-cd/internal/service"
)

type UserHandler struct {
	service service.UserService
}

func NewUserHandler(service service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// Search 用户搜索
func (h *UserHandler) Search(c *gin.Context) {
	var req dto.UserSearchQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		responses.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}

	users, total, err := h.service.Search(&req)
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, dto.NewPageResponse(users, total, req.GetPage(), req.GetPageSize()))
}

// ListRoles 获取系统角色列表
func (h *UserHandler) ListRoles(c *gin.Context) {
	responses.Success(c, h.service.ListRoles())
}
