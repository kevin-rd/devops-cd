package service

import (
	"devops-cd/internal/pkg/git/api"
	"fmt"
	"strings"

	"devops-cd/internal/model"
	"devops-cd/internal/pkg/config"
	"devops-cd/internal/pkg/git"
	"devops-cd/internal/repository"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RepoSyncService 代码库同步服务
type RepoSyncService struct {
	db       *gorm.DB
	repoRepo repository.RepositoryRepository
	logger   *zap.Logger
	config   *config.RepoConfig
}

// NewRepoSyncService 创建代码库同步服务
func NewRepoSyncService(db *gorm.DB, logger *zap.Logger, cfg *config.RepoConfig) *RepoSyncService {
	return &RepoSyncService{
		db:       db,
		repoRepo: repository.NewRepositoryRepository(db),
		logger:   logger,
		config:   cfg,
	}
}

// SyncAllSources 同步所有启用的代码库源
func (s *RepoSyncService) SyncAllSources() error {
	log := s.logger.Sugar()
	if s.config == nil || len(s.config.Sources) == 0 {
		s.logger.Info("没有配置代码库同步源，跳过同步")
		return nil
	}

	log.Infof("开始同步代码库 sources_count=%d", len(s.config.Sources))

	totalSuccess := 0
	totalFailed := 0

	for _, source := range s.config.Sources {
		if !source.Enabled {
			log.Debugf("跳过未启用的同步源 %s", source.Name)
			continue
		}

		success, failed, err := s.SyncFromSource(&source)
		if err != nil {
			s.logger.Error("同步代码库源失败", zap.String("source", source.Name), zap.Error(err))
			totalFailed += failed
			continue
		}

		totalSuccess += success
		totalFailed += failed

		log.Infof("代码库源: %s同步完成: %d 成功, %d 失败", source.Name, success, failed)
	}

	log.Infof("所有代码库同步完成 %d 成功, %d 失败", totalSuccess, totalFailed)
	return nil
}

// SyncFromSource 从单个源同步代码库
func (s *RepoSyncService) SyncFromSource(source *config.RepoSourceConfig) (int, int, error) {
	s.logger.Info("开始同步代码库源",
		zap.String("source", source.Name),
		zap.String("platform", source.Platform),
		zap.String("sync_mode", source.SyncMode))

	// 创建 Git 客户端
	gitClient, err := git.NewClient(source.BaseURL, source.Token, source.Platform)
	if err != nil {
		return 0, 0, fmt.Errorf("创建Git客户端失败: %w", err)
	}

	// 获取仓库列表
	allRepos, err := s.fetchRepositories(gitClient, source)
	if err != nil {
		return 0, 0, fmt.Errorf("获取仓库列表失败: %w", err)
	}

	s.logger.Info("获取到仓库列表", zap.Int("count", len(allRepos)))

	// 同步到数据库
	successCount := 0
	failedCount := 0

	for _, repoInfo := range allRepos {
		if err := s.syncRepository(&repoInfo, source); err != nil {
			s.logger.Error("同步仓库失败", zap.String("repo", repoInfo.FullName), zap.Error(err))
			failedCount++
			continue
		}
		successCount++
	}

	return successCount, failedCount, nil
}

// fetchRepositories 根据同步模式获取仓库列表
func (s *RepoSyncService) fetchRepositories(gitClient *git.Client, source *config.RepoSourceConfig) ([]api.RepositoryInfo, error) {
	var allRepos []api.RepositoryInfo

	switch strings.ToLower(source.SyncMode) {
	case "namespaces":
		// 同步指定的命名空间列表
		if len(source.Namespaces) == 0 {
			return nil, fmt.Errorf("sync_mode=namespaces 时必须指定 namespaces 列表")
		}

		for _, namespace := range source.Namespaces {
			repos, err := gitClient.ListRepositories(namespace)
			if err != nil {
				s.logger.Error("获取命名空间仓库失败", zap.String("namespace", namespace), zap.Error(err))
				continue
			}
			allRepos = append(allRepos, repos...)
		}

	case "user":
		// 同步当前用户的仓库
		user, err := gitClient.GetCurrentUser()
		if err != nil {
			return nil, fmt.Errorf("获取当前用户失败: %w", err)
		}

		s.logger.Info("获取当前用户", zap.String("username", user.Username), zap.String("email", user.Email))

		repos, err := gitClient.ListRepositories(user.Username)
		if err != nil {
			return nil, fmt.Errorf("获取当前用户仓库失败: %w", err)
		}
		allRepos = repos

	case "all":
		// 同步所有可访问的仓库
		repos, err := gitClient.ListAllAccessibleRepositories()
		if err != nil {
			return nil, fmt.Errorf("获取所有可访问仓库失败: %w", err)
		}
		allRepos = repos

	default:
		return nil, fmt.Errorf("不支持的同步模式: %s", source.SyncMode)
	}

	return allRepos, nil
}

// syncRepository 同步单个仓库到数据库
func (s *RepoSyncService) syncRepository(repoInfo *api.RepositoryInfo, source *config.RepoSourceConfig) error {
	// 转换为数据库模型
	repo := &model.Repository{
		Project:     repoInfo.Owner,
		Name:        repoInfo.Name,
		Description: &repoInfo.Description,
		GitURL:      repoInfo.CloneURL,
		GitType:     source.Platform,
		Language:    &repoInfo.Language,
	}

	// 使用 Upsert 插入或更新
	return s.repoRepo.Upsert(repo)
}
