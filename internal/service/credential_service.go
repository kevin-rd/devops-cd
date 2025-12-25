package service

import (
	"devops-cd/internal/dto"
	"devops-cd/internal/model"
	"devops-cd/internal/pkg/crypto"
	"devops-cd/internal/repository"
	pkgErrors "devops-cd/pkg/responses"
	"encoding/json"
	"fmt"
	"gorm.io/datatypes"
	"time"
)

type CredentialService interface {
	Create(req *dto.CreateCredentialRequest) (*dto.CredentialResponse, error)
	GetByID(id int64) (*dto.CredentialResponse, error)
	List(scope string, projectID *int64) ([]*dto.CredentialResponse, error)
	Update(id int64, req *dto.UpdateCredentialRequest) (*dto.CredentialResponse, error)
	Delete(id int64) error
}

type credentialService struct {
	repo *repository.CredentialRepository
}

func NewCredentialService(repo *repository.CredentialRepository) CredentialService {
	return &credentialService{repo: repo}
}

func (s *credentialService) Create(req *dto.CreateCredentialRequest) (*dto.CredentialResponse, error) {
	if req.Scope == string(model.ScopeProject) && req.ProjectID == nil {
		return nil, pkgErrors.New(pkgErrors.CodeBadRequest, "scope=project 时 project_id 必填")
	}
	if req.Scope == string(model.ScopeGlobal) {
		req.ProjectID = nil
	}

	enc, err := crypto.Encrypt(string(req.Data))
	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest, "凭据加密失败，请检查 crypto.aes_key 配置", err)
	}

	c := &model.Credential{
		Scope:         model.Scope(req.Scope),
		ProjectID:     req.ProjectID,
		Name:          req.Name,
		Type:          req.Type,
		EncryptedData: enc,
	}
	if req.Meta != nil && len(req.Meta) > 0 {
		c.MetaJSON = datatypes.JSON(req.Meta)
	}

	if err := s.repo.Create(c); err != nil {
		return nil, err
	}
	return toCredentialResponse(c), nil
}

func (s *credentialService) GetByID(id int64) (*dto.CredentialResponse, error) {
	c, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return toCredentialResponse(c), nil
}

func (s *credentialService) List(scope string, projectID *int64) ([]*dto.CredentialResponse, error) {
	list, err := s.repo.List(scope, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]*dto.CredentialResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toCredentialResponse(c))
	}
	return out, nil
}

func (s *credentialService) Update(id int64, req *dto.UpdateCredentialRequest) (*dto.CredentialResponse, error) {
	c, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	c.Name = req.Name
	if req.Data != nil && len(req.Data) > 0 {
		enc, err := crypto.Encrypt(string(req.Data))
		if err != nil {
			return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest, "凭据加密失败，请检查 crypto.aes_key 配置", err)
		}
		c.EncryptedData = enc
	}
	if req.Meta != nil {
		c.MetaJSON = datatypes.JSON(req.Meta)
	}

	if err := s.repo.Update(c); err != nil {
		return nil, err
	}
	return toCredentialResponse(c), nil
}

func (s *credentialService) Delete(id int64) error {
	return s.repo.Delete(id)
}

func toCredentialResponse(c *model.Credential) *dto.CredentialResponse {
	if c == nil {
		return nil
	}
	resp := &dto.CredentialResponse{
		ID:        c.ID,
		Scope:     string(c.Scope),
		ProjectID: c.ProjectID,
		Name:      c.Name,
		Type:      c.Type,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
		UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
	}
	if len(c.MetaJSON) > 0 {
		resp.Meta = json.RawMessage(c.MetaJSON)
	}
	return resp
}

// DecryptCredentialData 仅供内部使用：解密凭据明文
func DecryptCredentialData(c *model.Credential) (json.RawMessage, error) {
	if c == nil {
		return nil, fmt.Errorf("credential is nil")
	}
	plain, err := crypto.Decrypt(c.EncryptedData)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(plain), nil
}
