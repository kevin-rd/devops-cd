package handler

import (
	"devops-cd/internal/dto"
	"devops-cd/internal/pkg/logger"
	"devops-cd/internal/service"
	"devops-cd/pkg/responses"
	"devops-cd/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DeploymentHandler 部署任务处理器
type DeploymentHandler struct {
	batchService *service.BatchService
}

func NewDeploymentHandler(batchService *service.BatchService) *DeploymentHandler {
	return &DeploymentHandler{batchService: batchService}
}

// Retry 手动重试 deployment（仅 failed 可重试）
// @Summary 手动重试 deployment
// @Tags Deployment
// @Accept json
// @Produce json
// @Param id path int true "Deployment ID"
// @Param body body dto.RetryDeploymentRequest true "重试请求"
// @Success 200 {object} responses.Response{data=map[string]string}
// @Router /api/v1/deployment/{id}/retry [post]
func (h *DeploymentHandler) Retry(c *gin.Context) {
	deploymentID, ok := parseIDParam(c.Param("id"))
	if !ok {
		responses.ErrorWithDetail(c, http.StatusBadRequest, "deployment_id 无效", c.Param("id"))
		return
	}

	var req dto.RetryDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.ErrorWithDetail(c, http.StatusBadRequest, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	if err := h.batchService.RetryDeployment(deploymentID, req.Operator, req.Reason); err != nil {
		logger.Error("手动重试 deployment 失败", zap.Int64("deployment_id", deploymentID), zap.Error(err))
		responses.ErrorWithCode(c, http.StatusBadRequest, err.Error())
		return
	}

	responses.Success(c, gin.H{"message": "已触发重试"})
}
