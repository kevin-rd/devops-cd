package handler

import (
	"devops-cd/internal/dto"
	"devops-cd/internal/service"
	"devops-cd/pkg/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ClusterHandler struct {
	clusterService *service.ClusterService
}

func NewClusterHandler(clusterService *service.ClusterService) *ClusterHandler {
	return &ClusterHandler{
		clusterService: clusterService,
	}
}

// Create 创建集群
// @Summary 创建集群
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param request body dto.ClusterCreateRequest true "创建集群请求"
// @Success 200 {object} dto.ClusterResponse
// @Router /api/v1/clusters [post]
func (h *ClusterHandler) Create(c *gin.Context) {
	var req dto.ClusterCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithCode(c, 400, "请求参数错误: "+err.Error())
		return
	}

	resp, err := h.clusterService.Create(&req)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, resp)
}

// Update 更新集群
// @Summary 更新集群
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param request body dto.ClusterUpdateRequest true "更新集群请求"
// @Success 200 {object} dto.ClusterResponse
// @Router /api/v1/clusters/{id} [put]
func (h *ClusterHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorWithCode(c, 400, "无效的集群ID")
		return
	}

	var req dto.ClusterUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithCode(c, 400, "请求参数错误: "+err.Error())
		return
	}

	resp, err := h.clusterService.Update(id, &req)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, resp)
}

// Get 获取集群详情
// @Summary 获取集群详情
// @Tags 集群管理
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} dto.ClusterResponse
// @Router /api/v1/clusters/{id} [get]
func (h *ClusterHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorWithCode(c, 400, "无效的集群ID")
		return
	}

	resp, err := h.clusterService.Get(id)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, resp)
}

// List 获取集群列表
// @Summary 获取集群列表
// @Tags 集群管理
// @Produce json
// @Param name query string false "集群名称(模糊搜索)"
// @Param status query int false "状态(0:禁用 1:启用)"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} dto.PageResponse
// @Router /api/v1/clusters [get]
func (h *ClusterHandler) List(c *gin.Context) {
	var req dto.ClusterListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ErrorWithCode(c, 400, "请求参数错误: "+err.Error())
		return
	}

	// 设置默认分页
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	data, total, err := h.clusterService.List(&req)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, dto.NewPageResponse(data, total, req.Page, req.PageSize))
}

// Delete 删除集群
// @Summary 删除集群
// @Tags 集群管理
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/clusters/{id} [delete]
func (h *ClusterHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorWithCode(c, 400, "无效的集群ID")
		return
	}

	if err := h.clusterService.Delete(id); err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, gin.H{"message": "删除成功"})
}
