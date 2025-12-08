package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"devops-cd/internal/pkg/config"
	"devops-cd/pkg/constants"
	pkgErrors "devops-cd/pkg/errors"
)

// UserClaims 用户Claims
type UserClaims struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	AuthType    string `json:"auth_type"` // ldap or local
	UID         string `json:"uid"`
	Phone       string `json:"phone"`
	Type        string `json:"type"` // access or refresh
	jwt.RegisteredClaims
}

// GenerateAccessToken 生成访问Token
func GenerateAccessToken(username, email, displayName, authType, uid, phone string) (string, error) {
	cfg := config.GlobalConfig.Auth.JWT

	claims := UserClaims{
		Username:    username,
		Email:       email,
		DisplayName: displayName,
		AuthType:    authType,
		UID:         uid,
		Phone:       phone,
		Type:        constants.JWTTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   username,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.AccessTokenExpire) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

// GenerateRefreshToken 生成刷新Token
func GenerateRefreshToken(username, email, displayName, authType, uid, phone string) (string, error) {
	cfg := config.GlobalConfig.Auth.JWT

	claims := UserClaims{
		Username:    username,
		Email:       email,
		DisplayName: displayName,
		AuthType:    authType,
		UID:         uid,
		Phone:       phone,
		Type:        constants.JWTTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   username,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.RefreshTokenExpire) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

// ParseToken 解析Token
func ParseToken(tokenString string) (*UserClaims, error) {
	cfg := config.GlobalConfig.Auth.JWT

	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.Secret), nil
	})

	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeUnauthorized, "解析Token失败", err)
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, pkgErrors.ErrInvalidToken
}

// ValidateToken 验证Token有效性
func ValidateToken(tokenString string) (*UserClaims, error) {
	claims, err := ParseToken(tokenString)
	if err != nil {
		return nil, err
	}

	// 检查是否过期
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, pkgErrors.ErrTokenExpired
	}

	return claims, nil
}
