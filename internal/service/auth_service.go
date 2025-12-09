package service

import (
	"devops-cd/internal/dto"
	"devops-cd/internal/model"
	"devops-cd/internal/pkg/config"
	"devops-cd/internal/pkg/crypto"
	"devops-cd/internal/pkg/jwt"
	"devops-cd/internal/repository"
	"devops-cd/pkg/constants"
	pkgErrors "devops-cd/pkg/responses"
	"devops-cd/pkg/utils/strings"
)

type AuthService interface {
	Login(req *dto.LoginRequest) (*dto.LoginResponse, error)
	RefreshToken(refreshToken string) (*dto.LoginResponse, error)
	VerifyToken(token string) (*dto.UserInfo, error)
}

type authService struct {
	cfg         *config.AuthConfig
	userRepo    repository.UserRepository
	ldapService LDAPService
}

func NewAuthService(
	cfg *config.AuthConfig,
	userRepo repository.UserRepository,
	ldapService LDAPService,
) AuthService {
	return &authService{
		cfg:         cfg,
		userRepo:    userRepo,
		ldapService: ldapService,
	}
}

func (s *authService) Login(req *dto.LoginRequest) (*dto.LoginResponse, error) {
	var userInfo *dto.UserInfo
	var err error

	switch req.AuthType {
	case constants.AuthTypeLDAP:
		if !s.cfg.LDAP.Enabled {
			return nil, pkgErrors.New(pkgErrors.CodeAuthError, "LDAP认证未启用")
		}
		userInfo, err = s.ldapService.Authenticate(req.Username, req.Password)
		if err != nil {
			return nil, err
		}
		if err := s.syncLDAPUser(userInfo); err != nil {
			return nil, err
		}

	case constants.AuthTypeLocal:
		if !s.cfg.Local.Enabled {
			return nil, pkgErrors.New(pkgErrors.CodeAuthError, "本地认证未启用")
		}
		userInfo, err = s.authenticateLocal(req.Username, req.Password)
		if err != nil {
			return nil, err
		}

	default:
		return nil, pkgErrors.New(pkgErrors.CodeBadRequest, "不支持的认证类型")
	}

	// 生成Token
	accessToken, err := jwt.GenerateAccessToken(
		userInfo.Username,
		userInfo.Email,
		userInfo.DisplayName,
		userInfo.AuthType,
		userInfo.UID,
		userInfo.Phone,
	)
	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeInternalError, "生成AccessToken失败", err)
	}

	refreshToken, err := jwt.GenerateRefreshToken(
		userInfo.Username,
		userInfo.Email,
		userInfo.DisplayName,
		userInfo.AuthType,
		userInfo.UID,
		userInfo.Phone,
	)
	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeInternalError, "生成RefreshToken失败", err)
	}

	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.cfg.JWT.AccessTokenExpire,
		User:         userInfo,
	}, nil
}

func (s *authService) authenticateLocal(username, password string) (*dto.UserInfo, error) {
	// 查询用户
	user, err := s.userRepo.FindByUsername(username, constants.AuthTypeLocal)
	if err != nil {
		if err == pkgErrors.ErrRecordNotFound {
			return nil, pkgErrors.ErrInvalidCredentials
		}
		return nil, err
	}

	// 检查状态
	if user.Status != constants.StatusEnabled {
		return nil, pkgErrors.ErrUserDisabled
	}

	// 验证密码
	if !crypto.CheckPassword(password, user.Password) {
		return nil, pkgErrors.ErrInvalidCredentials
	}

	// 更新最后登录时间
	_ = s.userRepo.UpdateLastLogin(user.ID)

	// 构建用户信息
	email := ""
	if user.Email != nil {
		email = *user.Email
	}
	displayName := username
	if user.DisplayName != nil {
		displayName = *user.DisplayName
	}
	phone := ""
	if user.Phone != nil {
		phone = *user.Phone
	}
	uid := ""
	if user.ExternalUID != nil {
		uid = *user.ExternalUID
	}

	return &dto.UserInfo{
		Username:    user.Username,
		Email:       email,
		DisplayName: displayName,
		AuthType:    constants.AuthTypeLocal,
		UID:         uid,
		Phone:       phone,
	}, nil
}

func (s *authService) syncLDAPUser(userInfo *dto.UserInfo) error {
	user, err := s.userRepo.FindByUsername(userInfo.Username, constants.AuthTypeLDAP)
	if err != nil {
		if err == pkgErrors.ErrRecordNotFound {
			user = &model.User{
				AuthProvider: constants.AuthTypeLDAP,
				Username:     userInfo.Username,
				Password:     "",
				DisplayName:  strings.StringPtr(userInfo.DisplayName),
				Email:        strings.StringPtr(userInfo.Email),
				Phone:        strings.StringPtr(userInfo.Phone),
				ExternalUID:  strings.StringPtr(userInfo.UID),
				BaseStatus:   model.BaseStatus{Status: constants.StatusEnabled},
			}
			if err = s.userRepo.Create(user); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return s.userRepo.UpdateLastLogin(user.ID)
}

func (s *authService) RefreshToken(refreshToken string) (*dto.LoginResponse, error) {
	// 验证RefreshToken
	claims, err := jwt.ParseToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// 检查Token类型
	if claims.Type != constants.JWTTypeRefresh {
		return nil, pkgErrors.New(pkgErrors.CodeUnauthorized, "无效的RefreshToken")
	}

	// 生成新的AccessToken
	accessToken, err := jwt.GenerateAccessToken(
		claims.Username,
		claims.Email,
		claims.DisplayName,
		claims.AuthType,
		claims.UID,
		claims.Phone,
	)
	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeInternalError, "生成AccessToken失败", err)
	}

	// 生成新的RefreshToken
	newRefreshToken, err := jwt.GenerateRefreshToken(
		claims.Username,
		claims.Email,
		claims.DisplayName,
		claims.AuthType,
		claims.UID,
		claims.Phone,
	)
	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeInternalError, "生成RefreshToken失败", err)
	}

	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    s.cfg.JWT.AccessTokenExpire,
		User: &dto.UserInfo{
			Username:    claims.Username,
			Email:       claims.Email,
			DisplayName: claims.DisplayName,
			AuthType:    claims.AuthType,
			UID:         claims.UID,
			Phone:       claims.Phone,
		},
	}, nil
}

func (s *authService) VerifyToken(token string) (*dto.UserInfo, error) {
	claims, err := jwt.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	return &dto.UserInfo{
		Username:    claims.Username,
		Email:       claims.Email,
		DisplayName: claims.DisplayName,
		AuthType:    claims.AuthType,
		UID:         claims.UID,
		Phone:       claims.Phone,
	}, nil
}
