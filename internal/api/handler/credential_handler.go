package handler

import (
	"devops-cd/internal/dto"
	"devops-cd/internal/service"
	"devops-cd/pkg/responses"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CredentialHandler struct {
	svc service.CredentialService
}

func NewCredentialHandler(svc service.CredentialService) *CredentialHandler {
	return &CredentialHandler{svc: svc}
}

// Create 创建凭据
// @Summary 创建凭据
// @Tags Credential
// @Accept json
// @Produce json
// @Param request body dto.CreateCredentialRequest true "创建凭据请求"
// @Success 200 {object} responses.Response{data=dto.CredentialResponse}
// @Router /api/v1/credentials [post]
func (h *CredentialHandler) Create(c *gin.Context) {
	var req dto.CreateCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}
	resp, err := h.svc.Create(&req)
	if err != nil {
		responses.Error(c, err)
		return
	}
	responses.Success(c, resp)
}

// List 列表
// @Summary 凭据列表
// @Tags Credential
// @Produce json
// @Param scope query string false "global/project"
// @Param project_id query int64 false "项目ID(scope=project时)"
// @Success 200 {object} responses.Response{data=[]dto.CredentialResponse}
// @Router /api/v1/credentials [get]
func (h *CredentialHandler) List(c *gin.Context) {
	scope := c.Query("scope")
	var projectID *int64
	if v := c.Query("project_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			responses.ErrorWithDetail(c, http.StatusBadRequest, "无效的 project_id", err.Error())
			return
		}
		projectID = &id
	}
	list, err := h.svc.List(scope, projectID)
	if err != nil {
		responses.Error(c, err)
		return
	}
	responses.Success(c, list)
}

func (h *CredentialHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		responses.ErrorWithDetail(c, http.StatusBadRequest, "无效的 ID", err.Error())
		return
	}
	resp, err := h.svc.GetByID(id)
	if err != nil {
		responses.Error(c, err)
		return
	}
	responses.Success(c, resp)
}

func (h *CredentialHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		responses.ErrorWithDetail(c, http.StatusBadRequest, "无效的 ID", err.Error())
		return
	}
	var req dto.UpdateCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", err.Error())
		return
	}
	resp, err := h.svc.Update(id, &req)
	if err != nil {
		responses.Error(c, err)
		return
	}
	responses.Success(c, resp)
}

func (h *CredentialHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		responses.ErrorWithDetail(c, http.StatusBadRequest, "无效的 ID", err.Error())
		return
	}
	if err := h.svc.Delete(id); err != nil {
		responses.Error(c, err)
		return
	}
	responses.Success(c, nil)
}
