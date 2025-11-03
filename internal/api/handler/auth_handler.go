package handler

import (
	"github.com/gin-gonic/gin"

	"devops-cd/internal/dto"
	"devops-cd/internal/service"
	"devops-cd/pkg/utils"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Login 登录
// @Summary 用户登录
// @Description 支持LDAP和本地用户登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "登录请求"
// @Success 200 {object} dto.LoginResponse
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, 400, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	resp, err := h.authService.Login(&req)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, resp)
}

// Refresh 刷新Token
// @Summary 刷新访问Token
// @Description 使用RefreshToken获取新的AccessToken
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "刷新Token请求"
// @Success 200 {object} dto.LoginResponse
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithDetail(c, 400, "请求参数错误", utils.FormatValidationError(err))
		return
	}

	resp, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		utils.Error(c, err)
		return
	}

	utils.Success(c, resp)
}

// GetMe 获取当前用户信息
// @Summary 获取当前用户信息
// @Description 从JWT Token中获取当前登录用户信息
// @Tags 认证
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} dto.UserInfo
// @Router /api/v1/auth/me [get]
func (h *AuthHandler) GetMe(c *gin.Context) {
	// 从context中获取用户信息(由认证中间件设置)
	userInfo, exists := c.Get("user")
	if !exists {
		utils.ErrorWithCode(c, 401, "未登录")
		return
	}

	utils.Success(c, userInfo)
}

// Verify 验证Token
// @Summary 验证Token有效性
// @Description 验证Token是否有效(内部API)
// @Tags 认证
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} dto.UserInfo
// @Router /api/v1/auth/verify [get]
func (h *AuthHandler) Verify(c *gin.Context) {
	// 由认证中间件已验证,直接返回用户信息
	userInfo, exists := c.Get("user")
	if !exists {
		utils.ErrorWithCode(c, 401, "未登录")
		return
	}

	utils.Success(c, userInfo)
}
