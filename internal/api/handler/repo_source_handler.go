package handler

import (
	"devops-cd/pkg/responses"
	"strconv"

	"github.com/gin-gonic/gin"

	"devops-cd/internal/dto"
	"devops-cd/internal/service"
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
		responses.ErrorWithDetail(c, responses.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	items, total, err := h.sourceService.List(&query)
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, dto.NewPageResponse(items, total, query.GetPage(), query.GetPageSize()))
}

func (h *RepoSourceHandler) Create(c *gin.Context) {
	var req dto.CreateRepoSyncSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.ErrorWithDetail(c, responses.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	resp, err := h.sourceService.Create(&req)
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, resp)
}

func (h *RepoSourceHandler) Update(c *gin.Context) {
	var req dto.UpdateRepoSyncSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.ErrorWithDetail(c, responses.CodeBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	resp, err := h.sourceService.Update(&req)
	if err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, resp)
}

func (h *RepoSourceHandler) Delete(c *gin.Context) {
	id, err := parseRepoSourceID(c.Param("id"))
	if err != nil {
		responses.ErrorWithDetail(c, responses.CodeBadRequest, "ID 参数错误", err.Error())
		return
	}

	if err := h.sourceService.Delete(id); err != nil {
		responses.Error(c, err)
		return
	}

	responses.Success(c, nil)
}

func (h *RepoSourceHandler) TestConnection(c *gin.Context) {
	id, err := parseRepoSourceID(c.Param("id"))
	if err != nil {
		responses.ErrorWithDetail(c, responses.CodeBadRequest, "ID 参数错误", err.Error())
		return
	}

	if err := h.syncService.TestSourceConnection(id); err != nil {
		responses.Error(c, responses.Wrap(responses.CodeInternalError, err.Error(), err))
		return
	}

	responses.SuccessWithMessage(c, "连接测试成功", nil)
}

func (h *RepoSourceHandler) SyncNow(c *gin.Context) {
	id, err := parseRepoSourceID(c.Param("id"))
	if err != nil {
		responses.ErrorWithDetail(c, responses.CodeBadRequest, "ID 参数错误", err.Error())
		return
	}

	success, failed, syncErr := h.syncService.SyncSourceByID(id)
	if syncErr != nil {
		responses.Error(c, responses.Wrap(responses.CodeInternalError, syncErr.Error(), syncErr))
		return
	}

	responses.SuccessWithMessage(c, "同步完成", map[string]int{
		"success": success,
		"failed":  failed,
	})
}

func parseRepoSourceID(idStr string) (int64, error) {
	return strconv.ParseInt(idStr, 10, 64)
}
