package service

import (
	"strings"

	"devops-cd/internal/dto"
	"devops-cd/internal/model"
	"devops-cd/internal/repository"
	pkgErrors "devops-cd/pkg/errors"
	"devops-cd/pkg/utils"
)

type RepoSourceService struct {
	repo     repository.RepoSyncSourceRepository
	teamRepo repository.TeamRepository
	aesKey   string
}

func NewRepoSourceService(repo repository.RepoSyncSourceRepository, teamRepo repository.TeamRepository, aesKey string) *RepoSourceService {
	return &RepoSourceService{
		repo:     repo,
		teamRepo: teamRepo,
		aesKey:   aesKey,
	}
}

func (s *RepoSourceService) List(query *dto.RepoSyncSourceListQuery) ([]*dto.RepoSyncSourceResponse, int64, error) {
	sources, total, err := s.repo.List(query.GetPage(), query.GetPageSize(), query.Keyword, query.Platform, query.BaseURL, query.Namespace, query.Enabled)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]*dto.RepoSyncSourceResponse, 0, len(sources))
	for _, source := range sources {
		resp = append(resp, s.toResponse(source))
	}
	return resp, total, nil
}

func (s *RepoSourceService) Create(req *dto.CreateRepoSyncSourceRequest) (*dto.RepoSyncSourceResponse, error) {
	if strings.TrimSpace(req.Token) == "" {
		return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest, "Token 不能为空", nil)
	}

	// 验证团队是否属于项目
	if err := s.validateTeamProject(req.DefaultTeamID, req.DefaultProjectID); err != nil {
		return nil, err
	}

	enc, err := utils.EncryptSecret(s.aesKey, req.Token)
	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeInternalError, "加密 Token 失败", err)
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	source := &model.RepoSource{
		Platform:         req.Platform,
		BaseURL:          req.BaseURL,
		Namespace:        req.Namespace,
		AuthTokenEnc:     enc,
		Enabled:          enabled,
		DefaultProjectID: req.DefaultProjectID,
		DefaultTeamID:    req.DefaultTeamID,
		CreatedBy:        req.CreatedBy,
	}

	if err := s.repo.Create(source); err != nil {
		return nil, err
	}

	return s.toResponse(source), nil
}

func (s *RepoSourceService) Update(req *dto.UpdateRepoSyncSourceRequest) (*dto.RepoSyncSourceResponse, error) {
	source, err := s.repo.GetByID(req.ID)
	if err != nil {
		return nil, err
	}

	// 验证团队是否属于项目
	if err := s.validateTeamProject(req.DefaultTeamID, req.DefaultProjectID); err != nil {
		return nil, err
	}

	source.Platform = req.Platform
	source.BaseURL = req.BaseURL
	source.Namespace = req.Namespace
	source.UpdatedBy = req.UpdatedBy
	source.DefaultProjectID = req.DefaultProjectID
	source.DefaultTeamID = req.DefaultTeamID

	if req.Enabled != nil {
		source.Enabled = *req.Enabled
	}

	if req.Token != nil {
		if strings.TrimSpace(*req.Token) == "" {
			return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest, "Token 不能为空", nil)
		}

		enc, err := utils.EncryptSecret(s.aesKey, *req.Token)
		if err != nil {
			return nil, pkgErrors.Wrap(pkgErrors.CodeInternalError, "加密 Token 失败", err)
		}
		source.AuthTokenEnc = enc
	}

	if err := s.repo.Update(source); err != nil {
		return nil, err
	}

	return s.toResponse(source), nil
}

func (s *RepoSourceService) Delete(id int64) error {
	return s.repo.Delete(id)
}

func (s *RepoSourceService) toResponse(source *model.RepoSource) *dto.RepoSyncSourceResponse {
	hasToken := source.AuthTokenEnc != ""

	resp := &dto.RepoSyncSourceResponse{
		ID:               source.ID,
		Platform:         source.Platform,
		BaseURL:          source.BaseURL,
		Namespace:        source.Namespace,
		Enabled:          source.Enabled,
		DefaultProjectID: source.DefaultProjectID,
		DefaultTeamID:    source.DefaultTeamID,
		LastSyncedAt:     source.LastSyncedAt,
		LastStatus:       source.LastStatus,
		LastMessage:      source.LastMessage,
		CreatedAt:        source.CreatedAt,
		UpdatedAt:        source.UpdatedAt,
		HasToken:         hasToken,
	}

	// 添加关联的项目和团队名称
	if source.DefaultProject != nil {
		resp.DefaultProjectName = &source.DefaultProject.Name
	}
	if source.DefaultTeam != nil {
		resp.DefaultTeamName = &source.DefaultTeam.Name
	}

	return resp
}

// validateTeamProject 验证团队是否属于项目
func (s *RepoSourceService) validateTeamProject(teamID *int64, projectID *int64) error {
	// 如果设置了团队但没有设置项目，返回错误
	if teamID != nil && *teamID > 0 {
		if projectID == nil || *projectID <= 0 {
			return pkgErrors.Wrap(pkgErrors.CodeBadRequest, "设置默认团队时必须先设置默认项目", nil)
		}

		// 验证团队是否属于该项目
		belongs, err := s.teamRepo.VerifyTeamBelongsToProject(*teamID, *projectID)
		if err != nil {
			return pkgErrors.Wrap(pkgErrors.CodeInternalError, "验证团队归属失败", err)
		}
		if !belongs {
			return pkgErrors.Wrap(pkgErrors.CodeBadRequest, "团队不属于指定的项目", nil)
		}
	}
	return nil
}

func (s *RepoSourceService) GetByID(id int64) (*model.RepoSource, error) {
	return s.repo.GetByID(id)
}

// ReplaceToken 用于在明文 token 缺失时提示
