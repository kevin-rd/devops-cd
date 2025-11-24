package service

import (
	"encoding/json"
	"fmt"
	"time"

	"devops-cd/internal/dto"
	"devops-cd/internal/model"
	"devops-cd/internal/repository"
	pkgErrors "devops-cd/pkg/errors"
)

type ProjectService interface {
	Create(req *dto.CreateProjectRequest) (*dto.ProjectResponse, error)
	GetByID(id int64) (*dto.ProjectResponse, error)
	List(query *dto.ProjectListQuery) ([]*dto.ProjectResponse, int64, error)
	ListAll() ([]*dto.ProjectSimpleResponse, error)
	Update(id int64, req *dto.UpdateProjectRequest) (*dto.ProjectResponse, error)
	Delete(id int64) error
	GetAvailableEnvClusters(projectID int64, env string) (*dto.ProjectAvailableEnvClustersResponse, error)
}

type projectService struct {
	repo     repository.ProjectRepository
	teamRepo repository.TeamRepository
}

func NewProjectService(repo repository.ProjectRepository, teamRepo repository.TeamRepository) ProjectService {
	return &projectService{
		repo:     repo,
		teamRepo: teamRepo,
	}
}

func (s *projectService) Create(req *dto.CreateProjectRequest) (*dto.ProjectResponse, error) {
	// 检查项目名称是否已存在
	existing, _ := s.repo.FindByName(req.Name)
	if existing != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest,
			fmt.Sprintf("项目 %s 已存在", req.Name), nil)
	}

	// 校验 default_env_clusters 是否是 allowed_env_clusters 的子集
	if err := s.validateDefaultEnvClusters(req.AllowedEnvClusters, req.DefaultEnvClusters); err != nil {
		return nil, err
	}

	// 创建项目
	project := &model.Project{
		Name:        req.Name,
		Description: req.Description,
		OwnerName:   req.OwnerName,
	}

	// 处理 allowed_env_clusters
	if req.AllowedEnvClusters != nil {
		jsonData, err := json.Marshal(req.AllowedEnvClusters)
		if err != nil {
			return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest, "环境集群配置格式错误", err)
		}
		jsonStr := string(jsonData)
		project.AllowedEnvClusters = &jsonStr
	}

	// 处理 default_env_clusters
	if req.DefaultEnvClusters != nil {
		jsonData, err := json.Marshal(req.DefaultEnvClusters)
		if err != nil {
			return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest, "默认环境集群配置格式错误", err)
		}
		jsonStr := string(jsonData)
		project.DefaultEnvClusters = &jsonStr
	}

	if err := s.repo.Create(project); err != nil {
		return nil, err
	}

	if s.shouldCreateDefaultTeam(req) {
		team := &model.Team{
			Name:      project.Name,
			ProjectID: project.ID,
		}
		if err := s.teamRepo.Create(team); err != nil {
			// 回滚项目创建
			_ = s.repo.Delete(project.ID)
			return nil, pkgErrors.Wrap(pkgErrors.CodeInternalError, "创建默认团队失败", err)
		}
	}

	return s.toResponse(project), nil
}

func (s *projectService) GetByID(id int64) (*dto.ProjectResponse, error) {
	project, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return s.toResponse(project), nil
}

func (s *projectService) List(query *dto.ProjectListQuery) ([]*dto.ProjectResponse, int64, error) {
	projects, total, err := s.repo.List(
		query.GetPage(),
		query.GetPageSize(),
		query.Keyword,
	)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*dto.ProjectResponse, len(projects))

	var teamMap map[int64][]*dto.TeamResponse
	if query.WithTeams && len(projects) > 0 {
		projectIDs := make([]int64, len(projects))
		for i, project := range projects {
			projectIDs[i] = project.ID
		}
		teams, err := s.teamRepo.ListByProjectIDs(projectIDs)
		if err != nil {
			return nil, 0, err
		}
		teamMap = make(map[int64][]*dto.TeamResponse)
		for _, team := range teams {
			teamMap[team.ProjectID] = append(teamMap[team.ProjectID], s.toTeamResponse(team))
		}
	}

	for i, project := range projects {
		resp := s.toResponse(project)
		if teamMap != nil {
			resp.Teams = teamMap[project.ID]
		}
		responses[i] = resp
	}

	return responses, total, nil
}

func (s *projectService) ListAll() ([]*dto.ProjectSimpleResponse, error) {
	projects, err := s.repo.ListAll()
	if err != nil {
		return nil, err
	}

	responses := make([]*dto.ProjectSimpleResponse, len(projects))
	for i, project := range projects {
		responses[i] = &dto.ProjectSimpleResponse{
			ID:   project.ID,
			Name: project.Name,
		}
	}

	return responses, nil
}

func (s *projectService) Update(id int64, req *dto.UpdateProjectRequest) (*dto.ProjectResponse, error) {
	// 查询项目
	project, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// 检查名称是否冲突
	if req.Name != nil && *req.Name != project.Name {
		existing, _ := s.repo.FindByName(*req.Name)
		if existing != nil {
			return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest,
				fmt.Sprintf("项目 %s 已存在", *req.Name), nil)
		}
		project.Name = *req.Name
	}

	// 校验 default_env_clusters 是否是 allowed_env_clusters 的子集
	if err := s.validateDefaultEnvClusters(req.AllowedEnvClusters, req.DefaultEnvClusters); err != nil {
		return nil, err
	}

	// 更新字段
	if req.Description != nil {
		project.Description = req.Description
	}
	if req.OwnerName != nil {
		project.OwnerName = req.OwnerName
	}
	if req.AllowedEnvClusters != nil {
		jsonData, err := json.Marshal(req.AllowedEnvClusters)
		if err != nil {
			return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest, "环境集群配置格式错误", err)
		}
		jsonStr := string(jsonData)
		project.AllowedEnvClusters = &jsonStr
	}
	if req.DefaultEnvClusters != nil {
		jsonData, err := json.Marshal(req.DefaultEnvClusters)
		if err != nil {
			return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest, "默认环境集群配置格式错误", err)
		}
		jsonStr := string(jsonData)
		project.DefaultEnvClusters = &jsonStr
	}

	// 保存更新
	if err := s.repo.Update(project); err != nil {
		return nil, err
	}

	return s.toResponse(project), nil
}

func (s *projectService) Delete(id int64) error {
	// 检查项目是否存在
	_, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	// TODO: 检查是否有代码库关联此项目，如果有则提示用户先解除关联

	// 软删除项目
	return s.repo.Delete(id)
}

// toResponse 转换为响应对象
func (s *projectService) toResponse(project *model.Project) *dto.ProjectResponse {
	resp := &dto.ProjectResponse{
		ID:          project.ID,
		Name:        project.Name,
		Description: project.Description,
		OwnerName:   project.OwnerName,
		CreatedAt:   project.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   project.UpdatedAt.Format(time.RFC3339),
	}

	// 解析 allowed_env_clusters
	if project.AllowedEnvClusters != nil && *project.AllowedEnvClusters != "" {
		var allowedEnvClusters map[string][]string
		if err := json.Unmarshal([]byte(*project.AllowedEnvClusters), &allowedEnvClusters); err == nil {
			resp.AllowedEnvClusters = &allowedEnvClusters
		}
	}

	// 解析 default_env_clusters
	if project.DefaultEnvClusters != nil && *project.DefaultEnvClusters != "" {
		var defaultEnvClusters map[string][]string
		if err := json.Unmarshal([]byte(*project.DefaultEnvClusters), &defaultEnvClusters); err == nil {
			resp.DefaultEnvClusters = &defaultEnvClusters
		}
	}

	return resp
}

func (s *projectService) toTeamResponse(team *model.Team) *dto.TeamResponse {
	return &dto.TeamResponse{
		ID:          team.ID,
		Name:        team.Name,
		ProjectID:   team.ProjectID,
		Description: team.Description,
		LeaderName:  team.LeaderName,
		CreatedAt:   team.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   team.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *projectService) shouldCreateDefaultTeam(req *dto.CreateProjectRequest) bool {
	if req.CreateDefaultTeam == nil {
		return true
	}
	return *req.CreateDefaultTeam
}

// GetAvailableEnvClusters 获取项目可用的环境集群配置
func (s *projectService) GetAvailableEnvClusters(projectID int64, env string) (*dto.ProjectAvailableEnvClustersResponse, error) {
	// 查询项目
	project, err := s.repo.FindByID(projectID)
	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeNotFound, "项目不存在", err)
	}

	resp := &dto.ProjectAvailableEnvClustersResponse{
		AllowedEnvClusters: make(map[string][]string),
		AvailableClusters:  []string{},
	}

	// 解析 allowed_env_clusters
	if project.AllowedEnvClusters != nil && *project.AllowedEnvClusters != "" {
		if err := json.Unmarshal([]byte(*project.AllowedEnvClusters), &resp.AllowedEnvClusters); err != nil {
			return nil, pkgErrors.Wrap(pkgErrors.CodeInternalError, "解析环境集群配置失败", err)
		}
	}

	// 如果指定了环境,返回该环境下可用的集群列表
	if env != "" {
		if clusters, ok := resp.AllowedEnvClusters[env]; ok {
			resp.AvailableClusters = clusters
		}
	}

	return resp, nil
}

// validateDefaultEnvClusters 校验 default_env_clusters 是否是 allowed_env_clusters 的子集
func (s *projectService) validateDefaultEnvClusters(allowedEnvClusters, defaultEnvClusters *map[string][]string) error {
	// 如果没有设置 default_env_clusters,不需要校验
	if defaultEnvClusters == nil {
		return nil
	}

	// 如果设置了 default_env_clusters 但没有设置 allowed_env_clusters,报错
	if allowedEnvClusters == nil {
		return pkgErrors.Wrap(pkgErrors.CodeBadRequest, "设置默认环境集群配置前,必须先设置允许的环境集群配置", nil)
	}

	// 校验每个环境和集群是否在 allowed_env_clusters 中
	for env, clusters := range *defaultEnvClusters {
		// 检查环境是否在 allowed_env_clusters 中
		allowedClusters, exists := (*allowedEnvClusters)[env]
		if !exists {
			return pkgErrors.Wrap(pkgErrors.CodeBadRequest,
				fmt.Sprintf("默认环境集群配置中的环境 '%s' 不在允许的环境列表中", env), nil)
		}

		// 检查每个集群是否在该环境的 allowed_clusters 中
		for _, cluster := range clusters {
			if !s.contains(allowedClusters, cluster) {
				return pkgErrors.Wrap(pkgErrors.CodeBadRequest,
					fmt.Sprintf("默认环境集群配置中的集群 '%s' 不在环境 '%s' 的允许集群列表中", cluster, env), nil)
			}
		}
	}

	return nil
}

// contains 检查字符串切片中是否包含指定字符串
func (s *projectService) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
