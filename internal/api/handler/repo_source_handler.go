package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"devops-cd/internal/dto"
	"devops-cd/internal/service"
	pkgErrors "devops-cd/pkg/errors"
	"devops-cd/pkg/utils"
)

type RepoSourceHandler struct {
	sourceService *service.RepoSourceService
	syncService   *service.RepoSyncService
}

func NewRepoSourceHandler(sourceService *service.RepoSourceService, syncService *service.RepoSyncService) *RepoSourceHandler {
	return &RepoSourceHandler{
		sourceService: sourceService,
		syncService:   syncService,
	}
}

func (h *RepoSourceHandler) List(c *gin.Context) {
	var query dto.RepoSyncSourceListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.ErrorWithDetail(c, pkgErrors.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	items, total, err := h.sourceService.List(&query)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, dto.NewPageResponse(items, total, query.GetPage(), query.GetPageSize()))
}

func (h *RepoSourceHandler) Create(c *gin.Context) {
	var req dto.CreateRepoSyncSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, pkgErrors.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	resp, err := h.sourceService.Create(&req)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, resp)
}

func (h *RepoSourceHandler) Update(c *gin.Context) {
	var req dto.UpdateRepoSyncSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, pkgErrors.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	resp, err := h.sourceService.Update(&req)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, resp)
}

func (h *RepoSourceHandler) Delete(c *gin.Context) {
	id, err := parseRepoSourceID(c.Param("id"))
	if err != nil {
		utils.ErrorWithDetail(c, pkgErrors.CodeBadRequest, "ID 参数错误", err.Error())
		return
	}

	if err := h.sourceService.Delete(id); err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, nil)
}

func (h *RepoSourceHandler) TestConnection(c *gin.Context) {
	id, err := parseRepoSourceID(c.Param("id"))
	if err != nil {
		utils.ErrorWithDetail(c, pkgErrors.CodeBadRequest, "ID 参数错误", err.Error())
		return
	}

	if err := h.syncService.TestSourceConnection(id); err != nil {
		utils.Error(c, pkgErrors.Wrap(pkgErrors.CodeInternalError, err.Error(), err))
		return
	}

	utils.SuccessWithMessage(c, "连接测试成功", nil)
}

func (h *RepoSourceHandler) SyncNow(c *gin.Context) {
	id, err := parseRepoSourceID(c.Param("id"))
	if err != nil {
		utils.ErrorWithDetail(c, pkgErrors.CodeBadRequest, "ID 参数错误", err.Error())
		return
	}

	success, failed, syncErr := h.syncService.SyncSourceByID(id)
	if syncErr != nil {
		utils.Error(c, pkgErrors.Wrap(pkgErrors.CodeInternalError, syncErr.Error(), syncErr))
		return
	}

	utils.SuccessWithMessage(c, "同步完成", map[string]int{
		"success": success,
		"failed":  failed,
	})
}

func parseRepoSourceID(idStr string) (int64, error) {
	return strconv.ParseInt(idStr, 10, 64)
}
