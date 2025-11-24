package service

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"devops-cd/internal/dto"
	"devops-cd/internal/model"
	"devops-cd/internal/repository"
	"devops-cd/pkg/constants"
	pkgErrors "devops-cd/pkg/errors"
)

type AppEnvConfigService interface {
	Create(req *dto.CreateAppEnvConfigRequest) (*dto.AppEnvConfigResponse, error)
	Update(id int64, req *dto.UpdateAppEnvConfigRequest) (*dto.AppEnvConfigResponse, error)
	Delete(id int64) error
	GetByID(id int64) (*dto.AppEnvConfigResponse, error)
	List(query *dto.ListAppEnvConfigsQuery) ([]*dto.AppEnvConfigResponse, error)
	BatchCreate(req *dto.BatchCreateAppEnvConfigsRequest) ([]*dto.AppEnvConfigResponse, error)

	// 内部方法:供其他 service 调用
	GetEnvConfigs(appID int64, env string) ([]*model.AppEnvConfig, error)
	CheckAppHasEnv(appID int64, env string) (bool, error)
}

type appEnvConfigService struct {
	repo    repository.AppEnvConfigRepository
	appRepo repository.ApplicationRepository
	db      *gorm.DB
}

func NewAppEnvConfigService(
	repo repository.AppEnvConfigRepository,
	appRepo repository.ApplicationRepository,
	db *gorm.DB,
) AppEnvConfigService {
	return &appEnvConfigService{
		repo:    repo,
		appRepo: appRepo,
		db:      db,
	}
}

func (s *appEnvConfigService) Create(req *dto.CreateAppEnvConfigRequest) (*dto.AppEnvConfigResponse, error) {
	// 1. 检查应用是否存在
	_, err := s.appRepo.FindByID(req.AppID)
	if err != nil {
		if err == pkgErrors.ErrRecordNotFound {
			return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest, "应用不存在", nil)
		}
		return nil, err
	}

	// 2. 检查是否已存在相同配置
	exists, err := s.repo.CheckExists(req.AppID, req.Env, req.Cluster)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest,
			fmt.Sprintf("应用在 %s 环境的 %s 集群已存在配置", req.Env, req.Cluster), nil)
	}

	// 3. 创建配置
	config := &model.AppEnvConfig{
		AppID:      req.AppID,
		Env:        req.Env,
		Cluster:    req.Cluster,
		Replicas:   req.Replicas,
		ConfigData: req.ConfigData,
		BaseStatus: model.BaseStatus{Status: constants.StatusEnabled},
	}

	if err := s.repo.Create(config); err != nil {
		return nil, err
	}

	return s.toResponse(config), nil
}

func (s *appEnvConfigService) Update(id int64, req *dto.UpdateAppEnvConfigRequest) (*dto.AppEnvConfigResponse, error) {
	// 1. 查询配置
	config, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// 2. 更新字段
	if req.Cluster != nil {
		// 检查新集群名是否冲突
		if *req.Cluster != config.Cluster {
			exists, err := s.repo.CheckExists(config.AppID, config.Env, *req.Cluster)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest,
					fmt.Sprintf("应用在 %s 环境的 %s 集群已存在配置", config.Env, *req.Cluster), nil)
			}
			config.Cluster = *req.Cluster
		}
	}

	if req.Replicas != nil {
		config.Replicas = *req.Replicas
	}

	if req.ConfigData != nil {
		config.ConfigData = req.ConfigData
	}

	if req.Status != nil {
		config.Status = *req.Status
	}

	// 3. 保存更新
	if err := s.repo.Update(config); err != nil {
		return nil, err
	}

	return s.toResponse(config), nil
}

func (s *appEnvConfigService) Delete(id int64) error {
	// 检查配置是否存在
	_, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	// 软删除
	return s.repo.Delete(id)
}

func (s *appEnvConfigService) GetByID(id int64) (*dto.AppEnvConfigResponse, error) {
	config, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return s.toResponse(config), nil
}

func (s *appEnvConfigService) List(query *dto.ListAppEnvConfigsQuery) ([]*dto.AppEnvConfigResponse, error) {
	var configs []*model.AppEnvConfig
	var err error

	if query.Env != nil {
		configs, err = s.repo.FindByAppIDAndEnv(query.AppID, *query.Env)
	} else {
		configs, err = s.repo.FindByAppID(query.AppID)
	}

	if err != nil {
		return nil, err
	}

	responses := make([]*dto.AppEnvConfigResponse, len(configs))
	for i, config := range configs {
		responses[i] = s.toResponse(config)
	}

	return responses, nil
}

func (s *appEnvConfigService) BatchCreate(req *dto.BatchCreateAppEnvConfigsRequest) ([]*dto.AppEnvConfigResponse, error) {
	// 1. 检查应用是否存在
	_, err := s.appRepo.FindByID(req.AppID)
	if err != nil {
		if err == pkgErrors.ErrRecordNotFound {
			return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest, "应用不存在", nil)
		}
		return nil, err
	}

	// 2. 检查是否有重复配置
	for i, item := range req.Configs {
		exists, err := s.repo.CheckExists(req.AppID, item.Env, item.Cluster)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, pkgErrors.Wrap(pkgErrors.CodeBadRequest,
				fmt.Sprintf("第 %d 项配置冲突: 应用在 %s 环境的 %s 集群已存在配置", i+1, item.Env, item.Cluster), nil)
		}
	}

	// 3. 批量创建
	configs := make([]*model.AppEnvConfig, len(req.Configs))
	for i, item := range req.Configs {
		configs[i] = &model.AppEnvConfig{
			AppID:      req.AppID,
			Env:        item.Env,
			Cluster:    item.Cluster,
			Replicas:   item.Replicas,
			ConfigData: item.ConfigData,
			BaseStatus: model.BaseStatus{Status: constants.StatusEnabled},
		}
	}

	if err := s.repo.BatchCreate(configs); err != nil {
		return nil, err
	}

	// 4. 返回创建的配置列表
	responses := make([]*dto.AppEnvConfigResponse, len(configs))
	for i, config := range configs {
		responses[i] = s.toResponse(config)
	}

	return responses, nil
}

// GetEnvConfigs 获取应用在某环境的所有集群配置(供其他服务调用)
func (s *appEnvConfigService) GetEnvConfigs(appID int64, env string) ([]*model.AppEnvConfig, error) {
	return s.repo.FindByAppIDAndEnv(appID, env)
}

// CheckAppHasEnv 检查应用是否配置了某环境(供其他服务调用)
func (s *appEnvConfigService) CheckAppHasEnv(appID int64, env string) (bool, error) {
	configs, err := s.repo.FindByAppIDAndEnv(appID, env)
	if err != nil {
		return false, err
	}
	return len(configs) > 0, nil
}

// toResponse 转换为响应对象
func (s *appEnvConfigService) toResponse(config *model.AppEnvConfig) *dto.AppEnvConfigResponse {
	return &dto.AppEnvConfigResponse{
		ID:         config.ID,
		AppID:      config.AppID,
		Env:        config.Env,
		Cluster:    config.Cluster,
		Replicas:   config.Replicas,
		ConfigData: config.ConfigData,
		Status:     config.Status,
		CreatedAt:  config.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  config.UpdatedAt.Format(time.RFC3339),
	}
}
