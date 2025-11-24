package repository

import (
	"devops-cd/internal/dto"
	"devops-cd/internal/model"

	"gorm.io/gorm"
)

type ClusterRepository struct {
	db *gorm.DB
}

func NewClusterRepository(db *gorm.DB) *ClusterRepository {
	return &ClusterRepository{db: db}
}

// Create 创建集群
func (r *ClusterRepository) Create(cluster *model.Cluster) error {
	return r.db.Create(cluster).Error
}

// Update 更新集群
func (r *ClusterRepository) Update(cluster *model.Cluster) error {
	return r.db.Save(cluster).Error
}

// FindByID 根据ID查询集群
func (r *ClusterRepository) FindByID(id int64) (*model.Cluster, error) {
	var cluster model.Cluster
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&cluster).Error
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

// FindByName 根据名称查询集群
func (r *ClusterRepository) FindByName(name string) (*model.Cluster, error) {
	var cluster model.Cluster
	err := r.db.Where("name = ? AND deleted_at IS NULL", name).First(&cluster).Error
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

// List 查询集群列表
func (r *ClusterRepository) List(req *dto.ClusterListRequest) ([]model.Cluster, int64, error) {
	var clusters []model.Cluster
	var total int64

	query := r.db.Model(&model.Cluster{}).Where("deleted_at IS NULL")

	// 过滤条件
	if req.Name != nil && *req.Name != "" {
		query = query.Where("name LIKE ?", "%"+*req.Name+"%")
	}
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	// 计数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	err := query.Order("created_at DESC").
		Limit(req.PageSize).
		Offset(offset).
		Find(&clusters).Error

	return clusters, total, err
}

// Delete 软删除集群
func (r *ClusterRepository) Delete(id int64) error {
	return r.db.Model(&model.Cluster{}).
		Where("id = ?", id).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}

// CheckNameExists 检查集群名称是否已存在
func (r *ClusterRepository) CheckNameExists(name string, excludeID *int64) (bool, error) {
	var count int64
	query := r.db.Model(&model.Cluster{}).Where("name = ? AND deleted_at IS NULL", name)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}
