package service

import (
	"crypto/tls"
	"devops-cd/internal/pkg/logger"
	pkgErrors "devops-cd/pkg/responses"
	"fmt"
	"net"
	"time"

	"github.com/go-ldap/ldap/v3"

	"devops-cd/internal/dto"
	"devops-cd/internal/pkg/config"
	"devops-cd/pkg/constants"
)

type LDAPService interface {
	Authenticate(username, password string) (*dto.UserInfo, error)
}

type ldapService struct {
	cfg  *config.LDAPConfig
	conn *ldap.Conn
}

func NewLDAPService(cfg *config.LDAPConfig) LDAPService {
	if !cfg.Enabled {
		return nil
	}

	conn, err := connect(cfg)
	if err != nil {
		logger.Fatalf("LDAP连接失败, %v", err)
		return nil
	}

	return &ldapService{
		cfg:  cfg,
		conn: conn,
	}
}

func (s *ldapService) Authenticate(username, password string) (*dto.UserInfo, error) {
	if !s.cfg.Enabled {
		return nil, pkgErrors.New(pkgErrors.CodeAuthError, "LDAP认证未启用")
	}

	if s.conn == nil {
		conn, err := connect(s.cfg)
		if err != nil {
			return nil, err
		}
		s.conn = conn
	}

	// 搜索用户
	userDN, attributes, err := s.searchUser(s.conn, username)
	if err != nil {
		return nil, err
	}

	// 验证密码
	if err = s.conn.Bind(userDN, password); err != nil {
		return nil, pkgErrors.ErrInvalidCredentials
	}

	// 构建用户信息
	displayName := attributes[s.cfg.Attributes.DisplayName]
	if displayName == "" {
		displayName = username
	}

	userInfo := &dto.UserInfo{
		Username:    username,
		Email:       attributes[s.cfg.Attributes.Email],
		DisplayName: displayName,
		UID:         attributes[s.cfg.Attributes.UID],
		Phone:       attributes[s.cfg.Attributes.Phone],
		AuthType:    constants.AuthTypeLDAP,
	}

	return userInfo, nil
}

func connect(cfg *config.LDAPConfig) (*ldap.Conn, error) {
	var conn *ldap.Conn
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	if cfg.UseSSL {
		if cfg.ClientCertPath == "" || cfg.ClientKeyPath == "" {
			return nil, pkgErrors.New(pkgErrors.CodeAuthError, "LDAP客户端证书路径不能为空")
		}
		cert, err := tls.LoadX509KeyPair(cfg.ClientCertPath, cfg.ClientKeyPath)
		if err != nil {
			return nil, pkgErrors.Wrap(pkgErrors.CodeAuthError, "LDAP客户端证书加载失败", err)
		}

		tlsConfig := &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: cfg.SkipTLSVerify,
			Certificates:       []tls.Certificate{cert},
		}
		conn, err = ldap.DialURL(fmt.Sprintf("ldaps://%s", address), ldap.DialWithTLSDialer(tlsConfig,
			&net.Dialer{
				Timeout:   timeout,
				KeepAlive: 30 * time.Second,
			}))
		if err != nil {
			return nil, pkgErrors.Wrap(pkgErrors.CodeAuthError, "LDAP连接失败", err)
		}
	} else {
		var err error
		conn, err = ldap.DialURL(fmt.Sprintf("ldap://%s", address))
		if err != nil {
			return nil, pkgErrors.Wrap(pkgErrors.CodeAuthError, "LDAP连接失败", err)
		}
	}

	// 使用管理员账号绑定（如果配置)
	if cfg.BindDN != "" {
		if err := conn.Bind(cfg.BindDN, cfg.BindPassword); err != nil {
			return nil, pkgErrors.Wrap(pkgErrors.CodeAuthError, "LDAP绑定失败", err)
		}
	}

	return conn, nil
}

func (s *ldapService) searchUser(conn *ldap.Conn, username string) (string, map[string]string, error) {
	// 构建搜索过滤器
	filter := fmt.Sprintf(s.cfg.UserFilter, ldap.EscapeFilter(username))

	// 搜索请求
	searchRequest := ldap.NewSearchRequest(
		s.cfg.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		filter,
		[]string{
			s.cfg.Attributes.Username,
			s.cfg.Attributes.Email,
			s.cfg.Attributes.DisplayName,
			s.cfg.Attributes.UID,
			s.cfg.Attributes.Phone,
		},
		nil,
	)

	// 执行搜索
	result, err := conn.Search(searchRequest)
	if err != nil {
		logger.Sugar().Errorf("LDAP搜索失败: %v", err)
		return "", nil, pkgErrors.Wrap(pkgErrors.CodeAuthError, "LDAP搜索失败", err)
	}

	if len(result.Entries) == 0 {
		return "", nil, pkgErrors.ErrUserNotFound
	}

	if len(result.Entries) > 1 {
		logger.Sugar().Warnf("找到多个匹配的用户: %v", result.Entries)
		return "", nil, pkgErrors.New(pkgErrors.CodeAuthError, "找到多个匹配的用户")
	}

	entry := result.Entries[0]
	attributes := make(map[string]string)
	for _, attr := range []string{
		s.cfg.Attributes.Username,
		s.cfg.Attributes.Email,
		s.cfg.Attributes.DisplayName,
		s.cfg.Attributes.UID,
		s.cfg.Attributes.Phone,
	} {
		if attr == "" {
			continue
		}
		attributes[attr] = entry.GetAttributeValue(attr)
	}

	return entry.DN, attributes, nil
}
