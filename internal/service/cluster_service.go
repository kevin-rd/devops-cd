package service

import (
	"devops-cd/internal/dto"
	"devops-cd/internal/model"
	"devops-cd/internal/repository"
	"devops-cd/pkg/responses"
	"fmt"

	"gorm.io/gorm"
)

type ClusterService struct {
	clusterRepo *repository.ClusterRepository
	db          *gorm.DB
}

func NewClusterService(db *gorm.DB) *ClusterService {
	return &ClusterService{
		clusterRepo: repository.NewClusterRepository(db),
		db:          db,
	}
}

// Create 创建集群
func (s *ClusterService) Create(req *dto.ClusterCreateRequest) (*dto.ClusterResponse, error) {
	// 1. 检查集群名称是否已存在
	exists, err := s.clusterRepo.CheckNameExists(req.Name, nil)
	if err != nil {
		return nil, responses.Wrap(responses.CodeInternalError, "检查集群名称失败", err)
	}
	if exists {
		return nil, responses.New(responses.CodeBadRequest, fmt.Sprintf("集群名称 '%s' 已存在", req.Name))
	}

	// 2. 创建集群
	cluster := &model.Cluster{
		Name:        req.Name,
		Description: req.Description,
		Region:      req.Region,
	}

	if err := s.clusterRepo.Create(cluster); err != nil {
		return nil, responses.Wrap(responses.CodeInternalError, "创建集群失败", err)
	}

	return s.toClusterResponse(cluster), nil
}

// Update 更新集群
func (s *ClusterService) Update(id int64, req *dto.ClusterUpdateRequest) (*dto.ClusterResponse, error) {
	// 1. 查询集群
	cluster, err := s.clusterRepo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, responses.Wrap(responses.CodeNotFound, "集群不存在", err)
		}
		return nil, responses.Wrap(responses.CodeInternalError, "查询集群失败", err)
	}

	// 2. 检查名称是否重复
	if req.Name != nil && *req.Name != cluster.Name {
		exists, err := s.clusterRepo.CheckNameExists(*req.Name, &id)
		if err != nil {
			return nil, responses.Wrap(responses.CodeInternalError, "检查集群名称失败", err)
		}
		if exists {
			return nil, responses.New(responses.CodeBadRequest, fmt.Sprintf("集群名称 '%s' 已存在", *req.Name))
		}
		cluster.Name = *req.Name
	}

	// 3. 更新字段
	if req.Description != nil {
		cluster.Description = req.Description
	}
	if req.Region != nil {
		cluster.Region = req.Region
	}

	// 4. 保存更新
	if err := s.clusterRepo.Update(cluster); err != nil {
		return nil, responses.Wrap(responses.CodeInternalError, "更新集群失败", err)
	}

	return s.toClusterResponse(cluster), nil
}

// Get 获取集群详情
func (s *ClusterService) Get(id int64) (*dto.ClusterResponse, error) {
	cluster, err := s.clusterRepo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, responses.Wrap(responses.CodeNotFound, "集群不存在", err)
		}
		return nil, responses.Wrap(responses.CodeInternalError, "查询集群失败", err)
	}
	return s.toClusterResponse(cluster), nil
}

// List 获取集群列表
func (s *ClusterService) List(req *dto.ClusterListRequest) ([]dto.ClusterResponse, int64, error) {
	clusters, total, err := s.clusterRepo.List(req)
	if err != nil {
		return nil, 0, responses.Wrap(responses.CodeInternalError, "查询集群列表失败", err)
	}

	responses := make([]dto.ClusterResponse, len(clusters))
	for i, cluster := range clusters {
		responses[i] = *s.toClusterResponse(&cluster)
	}

	return responses, total, nil
}

// Delete 删除集群
func (s *ClusterService) Delete(id int64) error {
	// 1. 检查集群是否存在
	_, err := s.clusterRepo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return responses.Wrap(responses.CodeNotFound, "集群不存在", err)
		}
		return responses.Wrap(responses.CodeInternalError, "查询集群失败", err)
	}

	// 2. 检查是否有应用正在使用该集群
	var count int64
	if err := s.db.Model(&model.AppEnvConfig{}).
		Joins("JOIN clusters ON app_env_configs.cluster = clusters.name").
		Where("clusters.id = ? AND app_env_configs.deleted_at IS NULL", id).
		Count(&count).Error; err != nil {
		return responses.Wrap(responses.CodeInternalError, "检查集群使用情况失败", err)
	}

	if count > 0 {
		return responses.New(responses.CodeBadRequest, fmt.Sprintf("集群正在被 %d 个应用配置使用,无法删除", count))
	}

	// 3. 执行软删除
	if err := s.clusterRepo.Delete(id); err != nil {
		return responses.Wrap(responses.CodeInternalError, "删除集群失败", err)
	}

	return nil
}

// toClusterResponse 转换为响应DTO
func (s *ClusterService) toClusterResponse(cluster *model.Cluster) *dto.ClusterResponse {
	return &dto.ClusterResponse{
		ID:          cluster.ID,
		Name:        cluster.Name,
		Description: cluster.Description,
		Region:      cluster.Region,
		CreatedAt:   cluster.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   cluster.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
