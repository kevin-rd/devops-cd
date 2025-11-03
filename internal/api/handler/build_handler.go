package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"devops-cd/internal/dto"
	"devops-cd/internal/pkg/logger"
	"devops-cd/internal/service"
	pkgErrors "devops-cd/pkg/errors"
	"devops-cd/pkg/utils"
)

// BuildHandler 构建处理器
type BuildHandler struct {
	buildService service.BuildService
	batchService *service.BatchService // 保留，用于批次相关功能
}

// NewBuildHandler 创建构建处理器
func NewBuildHandler(buildService service.BuildService, batchService *service.BatchService) *BuildHandler {
	return &BuildHandler{
		buildService: buildService,
		batchService: batchService,
	}
}

// Notify 接收构建通知（Drone Webhook）
// @Summary 接收构建通知
// @Description 接收 Drone CI/CD 构建完成通知，记录构建信息并更新应用状态
// @Tags Build
// @Accept json
// @Produce json
// @Param request body dto.BuildNotifyRequest true "构建通知请求"
// @Success 200 {object} utils.Response "成功响应"
// @Router /build/notify [post]
func (h *BuildHandler) Notify(c *gin.Context) {
	var req dto.BuildNotifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, pkgErrors.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	log := logger.Log.Sugar().With(zap.String("repo", req.Repo))
	log.Infof("BuildHandler.Notify %s: %s, app num: %v, build_number: %v", req.Repo, req.BuildStatus, len(req.Apps), req.BuildNumber)

	// 处理构建通知
	if err := h.buildService.ProcessNotify(&req); err != nil {
		// 部分成功的情况也返回成功，但在响应中说明
		if err.(*pkgErrors.AppError).Code == pkgErrors.CodePartialSuccess {
			logger.Warn("构建通知部分处理成功", zap.Error(err))
			utils.Success(c, gin.H{"message": err.Error(), "status": "partial_success"})
			return
		}
		log.Errorf("处理构建通知失败: %v", err)
		utils.Error(c, err)
		return
	}

	utils.Success(c, gin.H{"message": "构建通知处理成功"})
}

// List 查询构建记录列表
// @Summary 查询构建记录列表
// @Tags Build
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param repo_id query int false "仓库ID"
// @Param app_id query int false "应用ID"
// @Param build_status query string false "构建状态"
// @Param build_event query string false "触发事件"
// @Param image_tag query string false "镜像标签"
// @Param commit_sha query string false "Commit SHA"
// @Param environment query string false "环境"
// @Param keyword query string false "关键字"
// @Success 200 {object} utils.Response{data=dto.PageResponse}
// @Router /api/v1/builds [get]
func (h *BuildHandler) List(c *gin.Context) {
	var query dto.BuildListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.ErrorWithDetail(c, pkgErrors.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	data, total, err := h.buildService.List(&query)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, dto.NewPageResponse(data, total, query.GetPage(), query.GetPageSize()))
}

// GetByID 获取构建记录详情
// @Summary 获取构建记录详情
// @Tags Build
// @Accept json
// @Produce json
// @Param id query int true "构建记录ID"
// @Success 200 {object} utils.Response{data=dto.BuildResponse}
// @Router /api/v1/build [get]
func (h *BuildHandler) GetByID(c *gin.Context) {
	var req dto.GetBuildRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ErrorWithDetail(c, pkgErrors.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	resp, err := h.buildService.GetByID(req.ID)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, resp)
}

// GetByAppAndNumber 根据应用和构建号查询
// @Summary 根据应用和构建号查询构建记录
// @Tags Build
// @Accept json
// @Produce json
// @Param app_id query int true "应用ID"
// @Param build_number query int true "构建号"
// @Success 200 {object} utils.Response{data=dto.BuildResponse}
// @Router /api/v1/build/app [get]
func (h *BuildHandler) GetByAppAndNumber(c *gin.Context) {
	var req dto.GetBuildByAppAndNumberRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ErrorWithDetail(c, pkgErrors.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	resp, err := h.buildService.GetByAppAndNumber(req.AppID, req.BuildNumber)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, resp)
}
