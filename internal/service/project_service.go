package service

import (
	"encoding/json"
	"fmt"
	"github.com/samber/lo"
	"time"

	"devops-cd/internal/dto"
	"devops-cd/internal/model"
	"devops-cd/internal/repository"
	pkgErrors "devops-cd/pkg/errors"
)

type ProjectService interface {
	Create(req *dto.CreateProjectRequest) (*dto.ProjectResponse, error)
	GetByID(id int64, withTeams bool) (*dto.ProjectResponse, error)
	List(query *dto.ProjectListQuery) ([]*dto.ProjectResponse, int64, error)
	ListAll() ([]*dto.ProjectSimpleResponse, error)
	Update(id int64, req *dto.UpdateProjectRequest) (*dto.ProjectResponse, error)
	Delete(id int64) error
	GetAvailableEnvClusters(projectID int64, env string) (*dto.ProjectAvailableEnvClustersResponse, error)

	GetEnvConfigs(projectID int64) ([]*dto.ProjectEnvConfigResponse, error)
	UpdateEnvConfigs(projectID int64, configs map[string]*dto.ProjectEnvConfigRequest) error
}

type projectService struct {
	repo          repository.ProjectRepository
	teamRepo      repository.TeamRepository
	envConfigRepo repository.ProjectEnvConfigRepository
}

func NewProjectService(repo repository.ProjectRepository, teamRepo repository.TeamRepository, envConfigRepo repository.ProjectEnvConfigRepository) ProjectService {
	return &projectService{
		repo:          repo,
		teamRepo:      teamRepo,
		envConfigRepo: envConfigRepo,
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

	if err := s.repo.Create(project); err != nil {
		return nil, err
	}

	// 处理环境配置：将 map 格式转换为 project_env_configs 表记录
	if req.AllowedEnvClusters != nil || req.DefaultEnvClusters != nil {
		envConfigs, err := s.convertMapToEnvConfigs(project.ID, req.AllowedEnvClusters, req.DefaultEnvClusters)
		if err != nil {
			// 回滚项目创建
			_ = s.repo.Delete(project.ID)
			return nil, err
		}
		if len(envConfigs) > 0 {
			if err := s.envConfigRepo.BatchCreate(envConfigs); err != nil {
				// 回滚项目创建
				_ = s.repo.Delete(project.ID)
				return nil, err
			}
		}
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

func (s *projectService) GetByID(id int64, withTeams bool) (*dto.ProjectResponse, error) {
	project, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	resp := s.toResponse(project)

	// 如果需要包含团队列表
	if withTeams {
		teams, err := s.teamRepo.ListByProjectIDs([]int64{id})
		if err != nil {
			return nil, err
		}

		resp.Teams = make([]*dto.TeamResponse, len(teams))
		for i, team := range teams {
			resp.Teams[i] = s.toTeamResponse(team)
		}
	}

	return resp, nil
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

	// 保存项目基本信息
	if err := s.repo.Update(project); err != nil {
		return nil, err
	}

	// 处理环境配置更新：如果提供了环境配置，先删除旧的，再创建新的
	if req.AllowedEnvClusters != nil || req.DefaultEnvClusters != nil {
		// 删除该项目的所有环境配置
		if err := s.envConfigRepo.DeleteByProjectID(id); err != nil {
			return nil, err
		}

		// 创建新的环境配置
		envConfigs, err := s.convertMapToEnvConfigs(id, req.AllowedEnvClusters, req.DefaultEnvClusters)
		if err != nil {
			return nil, err
		}
		if len(envConfigs) > 0 {
			if err := s.envConfigRepo.BatchCreate(envConfigs); err != nil {
				return nil, err
			}
		}
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

	// 从 project_env_configs 表读取环境配置并转换为 map 格式
	envConfigs, err := s.envConfigRepo.FindByProjectID(project.ID)
	if err == nil && len(envConfigs) > 0 {
		allowedEnvClusters, defaultEnvClusters := s.convertEnvConfigsToMap(envConfigs)
		// 即使 map 中有空数组的环境配置，也要返回（只要 map 不为空）
		if len(allowedEnvClusters) > 0 {
			resp.AllowedEnvClusters = &allowedEnvClusters
		}
		if len(defaultEnvClusters) > 0 {
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
	_, err := s.repo.FindByID(projectID)
	if err != nil {
		return nil, pkgErrors.Wrap(pkgErrors.CodeNotFound, "项目不存在", err)
	}

	resp := &dto.ProjectAvailableEnvClustersResponse{
		AllowedEnvClusters: make(map[string][]string),
		AvailableClusters:  []string{},
	}

	// 从 project_env_configs 表读取环境配置
	envConfigs, err := s.envConfigRepo.FindByProjectID(projectID)
	if err == nil && len(envConfigs) > 0 {
		allowedEnvClusters, _ := s.convertEnvConfigsToMap(envConfigs)
		resp.AllowedEnvClusters = allowedEnvClusters

		// 如果指定了环境,返回该环境下可用的集群列表
		if env != "" {
			if clusters, ok := allowedEnvClusters[env]; ok {
				resp.AvailableClusters = clusters
			}
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

// convertMapToEnvConfigs 将 map 格式的环境集群配置转换为 project_env_configs 表记录
// allowedEnvClusters: {"pre": ["cluster-a"], "prod": ["cluster-b"]}
// defaultEnvClusters: {"pre": ["cluster-a"]}
// 注意：即使某个环境的 clusters 为空数组 []，也会创建记录并保存空数组，不会删除该环境配置
func (s *projectService) convertMapToEnvConfigs(projectID int64, allowedEnvClusters, defaultEnvClusters *map[string][]string) ([]*model.ProjectEnvConfig, error) {
	var envConfigs []*model.ProjectEnvConfig

	// 收集所有环境（从 allowedEnvClusters 中获取）
	envSet := make(map[string]bool)
	if allowedEnvClusters != nil {
		for env := range *allowedEnvClusters {
			envSet[env] = true
		}
	}
	if defaultEnvClusters != nil {
		for env := range *defaultEnvClusters {
			envSet[env] = true
		}
	}

	// 为每个环境创建一条记录
	for env := range envSet {
		config := &model.ProjectEnvConfig{
			ProjectID: projectID,
			Env:       env,
		}

		// 设置 allow_clusters
		if allowedEnvClusters != nil {
			if clusters, ok := (*allowedEnvClusters)[env]; ok {
				allowClustersJSON, err := json.Marshal(clusters)
				if err != nil {
					return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest, "环境集群配置格式错误", err)
				}
				config.AllowClusters = string(allowClustersJSON)
			} else {
				// 如果没有该环境的 allowed 配置，设置为空数组
				config.AllowClusters = "[]"
			}
		} else {
			config.AllowClusters = "[]"
		}

		// 设置 default_clusters
		if defaultEnvClusters != nil {
			if clusters, ok := (*defaultEnvClusters)[env]; ok {
				defaultClustersJSON, err := json.Marshal(clusters)
				if err != nil {
					return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest, "默认环境集群配置格式错误", err)
				}
				config.DefaultClusters = string(defaultClustersJSON)
			} else {
				// 如果没有该环境的 default 配置，设置为空数组
				config.DefaultClusters = "[]"
			}
		} else {
			config.DefaultClusters = "[]"
		}

		// namespace 和 deployment_name_template 暂时设置为空字符串
		config.Namespace = ""
		config.DeploymentNameTemplate = ""

		envConfigs = append(envConfigs, config)
	}

	return envConfigs, nil
}

// convertEnvConfigsToMap 将 project_env_configs 表记录转换为 map 格式
// 返回: allowedEnvClusters, defaultEnvClusters
// 注意：即使 clusters 为空数组 []，也会保留在返回的 map 中，不会过滤掉
func (s *projectService) convertEnvConfigsToMap(envConfigs []*model.ProjectEnvConfig) (map[string][]string, map[string][]string) {
	allowedEnvClusters := make(map[string][]string)
	defaultEnvClusters := make(map[string][]string)

	for _, config := range envConfigs {
		// 解析 allow_clusters（即使为空数组也要保留）
		var allowClusters []string
		if err := json.Unmarshal([]byte(config.AllowClusters), &allowClusters); err == nil {
			// 即使为空数组也要添加到 map 中
			allowedEnvClusters[config.Env] = allowClusters
		}

		// 解析 default_clusters（即使为空数组也要保留）
		var defaultClusters []string
		if err := json.Unmarshal([]byte(config.DefaultClusters), &defaultClusters); err == nil {
			// 即使为空数组也要添加到 map 中
			defaultEnvClusters[config.Env] = defaultClusters
		}
	}

	return allowedEnvClusters, defaultEnvClusters
}

// GetEnvConfigs 获取项目的所有环境配置
func (s *projectService) GetEnvConfigs(projectID int64) ([]*dto.ProjectEnvConfigResponse, error) {
	// 检查项目是否存在
	_, err := s.repo.FindByID(projectID)
	if err != nil {
		return nil, err
	}

	// 查询环境配置
	configs, err := s.envConfigRepo.FindByProjectID(projectID)
	if err != nil {
		return nil, err
	}

	// 转换为响应格式
	responses := make([]*dto.ProjectEnvConfigResponse, len(configs))
	for i, config := range configs {
		responses[i] = s.toEnvConfigResponse(config)
	}

	return responses, nil
}

// UpdateEnvConfigs 批量更新项目的环境配置
// configs: map[env]ConfigData，如 {"pre": {...}, "prod": {...}}
func (s *projectService) UpdateEnvConfigs(projectID int64, configs map[string]*dto.ProjectEnvConfigRequest) error {
	// 检查项目是否存在
	_, err := s.repo.FindByID(projectID)
	if err != nil {
		return err
	}

	// 查询现有配置
	existingConfigs, err := s.envConfigRepo.FindByProjectID(projectID)
	if err != nil {
		return err
	}
	// 转为Map形式，key=item
	existingMap := lo.Associate(existingConfigs, func(config *model.ProjectEnvConfig) (string, *model.ProjectEnvConfig) {
		return config.Env, config
	})

	// 遍历请求中的配置，执行创建或更新
	for env, reqConfig := range configs {
		existing, ok := existingMap[env]
		if !ok {
			existing = &model.ProjectEnvConfig{
				ProjectID: projectID,
				Env:       env,
			}
		}

		// 序列化集群列表
		if reqConfig.AllowClusters != nil {
			allowClustersJSON, err := json.Marshal(reqConfig.AllowClusters)
			if err != nil {
				return pkgErrors.Wrap(pkgErrors.CodeBadRequest, fmt.Sprintf("环境 %s 的 allow_clusters 格式错误", env), err)
			}
			existing.AllowClusters = string(allowClustersJSON)
		}

		if reqConfig.DefaultClusters != nil {
			defaultClustersJSON, err := json.Marshal(reqConfig.DefaultClusters)
			if err != nil {
				return pkgErrors.Wrap(pkgErrors.CodeBadRequest, fmt.Sprintf("环境 %s 的 default_clusters 格式错误", env), err)
			}
			existing.DefaultClusters = string(defaultClustersJSON)
		}

		if reqConfig.Namespace != nil {
			existing.Namespace = *reqConfig.Namespace
		}

		if reqConfig.DeploymentNameTemplate != nil {
			existing.DeploymentNameTemplate = *reqConfig.DeploymentNameTemplate
		}

		if reqConfig.ChartRepoURL != nil {
			existing.ChartRepoURL = *reqConfig.ChartRepoURL
		}

		if reqConfig.ValuesRepoURL != nil {
			existing.ValuesRepoURL = reqConfig.ValuesRepoURL
		}

		if reqConfig.ValuesPathTemplate != nil {
			existing.ValuesPathTemplate = reqConfig.ValuesPathTemplate
		}

		// update or create
		if ok {
			if err := s.envConfigRepo.Update(existing); err != nil {
				return err
			}
		} else {
			if err := s.envConfigRepo.Create(existing); err != nil {
				return err
			}
		}
	}

	return nil
}

// toEnvConfigResponse 转换环境配置为响应格式
func (s *projectService) toEnvConfigResponse(config *model.ProjectEnvConfig) *dto.ProjectEnvConfigResponse {
	resp := &dto.ProjectEnvConfigResponse{
		ID:                     config.ID,
		ProjectID:              config.ProjectID,
		Env:                    config.Env,
		Namespace:              config.Namespace,
		DeploymentNameTemplate: config.DeploymentNameTemplate,
		ChartRepoURL:           config.ChartRepoURL,
		ValuesRepoURL:          config.ValuesRepoURL,
		ValuesPathTemplate:     config.ValuesPathTemplate,
		CreatedAt:              config.CreatedAt.Format(time.RFC3339),
		UpdatedAt:              config.UpdatedAt.Format(time.RFC3339),
	}

	// 反序列化集群列表
	var allowClusters []string
	if err := json.Unmarshal([]byte(config.AllowClusters), &allowClusters); err == nil {
		resp.AllowClusters = allowClusters
	}

	var defaultClusters []string
	if err := json.Unmarshal([]byte(config.DefaultClusters), &defaultClusters); err == nil {
		resp.DefaultClusters = defaultClusters
	}

	return resp
}
