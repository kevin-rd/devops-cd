package service

import (
	"fmt"

	"github.com/go-ldap/ldap/v3"

	"devops-cd/internal/dto"
	"devops-cd/internal/pkg/config"
	pkgErrors "devops-cd/pkg/errors"
)

type LDAPService interface {
	Authenticate(username, password string) (*dto.UserInfo, error)
}

type ldapService struct {
	cfg *config.LDAPConfig
}

func NewLDAPService(cfg *config.LDAPConfig) LDAPService {
	return &ldapService{
		cfg: cfg,
	}
}

func (s *ldapService) Authenticate(username, password string) (*dto.UserInfo, error) {
	if !s.cfg.Enabled {
		return nil, pkgErrors.New(pkgErrors.CodeAuthError, "LDAP认证未启用")
	}

	// 连接LDAP服务器
	conn, err := s.connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// 搜索用户
	userDN, attributes, err := s.searchUser(conn, username)
	if err != nil {
		return nil, err
	}

	// 验证密码
	if err := conn.Bind(userDN, password); err != nil {
		return nil, pkgErrors.ErrInvalidCredentials
	}

	// 构建用户信息
	userInfo := &dto.UserInfo{
		Username:    username,
		Email:       attributes[s.cfg.Attributes.Email],
		DisplayName: attributes[s.cfg.Attributes.DisplayName],
		AuthType:    "ldap",
	}

	return userInfo, nil
}

func (s *ldapService) connect() (*ldap.Conn, error) {
	var conn *ldap.Conn
	var err error

	address := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	if s.cfg.UseSSL {
		conn, err = ldap.DialTLS("tcp", address, nil)
	} else {
		conn, err = ldap.Dial("tcp", address)
	}

	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeAuthError, "LDAP连接失败", err)
	}

	// 使用管理员账号绑定
	if err := conn.Bind(s.cfg.BindDN, s.cfg.BindPassword); err != nil {
		conn.Close()
		return nil, pkgErrors.Wrap(pkgErrors.CodeAuthError, "LDAP绑定失败", err)
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
		[]string{s.cfg.Attributes.Username, s.cfg.Attributes.Email, s.cfg.Attributes.DisplayName},
		nil,
	)

	// 执行搜索
	result, err := conn.Search(searchRequest)
	if err != nil {
		return "", nil, pkgErrors.Wrap(pkgErrors.CodeAuthError, "LDAP搜索失败", err)
	}

	if len(result.Entries) == 0 {
		return "", nil, pkgErrors.ErrUserNotFound
	}

	if len(result.Entries) > 1 {
		return "", nil, pkgErrors.New(pkgErrors.CodeAuthError, "找到多个匹配的用户")
	}

	entry := result.Entries[0]
	attributes := make(map[string]string)
	attributes[s.cfg.Attributes.Username] = entry.GetAttributeValue(s.cfg.Attributes.Username)
	attributes[s.cfg.Attributes.Email] = entry.GetAttributeValue(s.cfg.Attributes.Email)
	attributes[s.cfg.Attributes.DisplayName] = entry.GetAttributeValue(s.cfg.Attributes.DisplayName)

	return entry.DN, attributes, nil
}
