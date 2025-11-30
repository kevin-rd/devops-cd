package service

import (
	"fmt"

	"devops-cd/internal/model"
	"devops-cd/internal/pkg/git"
	"devops-cd/internal/pkg/git/api"
	"devops-cd/internal/repository"
	"devops-cd/pkg/utils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RepoSyncService 代码库同步服务
type RepoSyncService struct {
	repoRepo   repository.RepositoryRepository
	sourceRepo repository.RepoSyncSourceRepository
	teamRepo   repository.TeamRepository
	logger     *zap.Logger
	aesKey     string
}

// NewRepoSyncService 创建代码库同步服务
func NewRepoSyncService(db *gorm.DB, logger *zap.Logger, aesKey string) *RepoSyncService {
	return &RepoSyncService{
		repoRepo:   repository.NewRepositoryRepository(db),
		sourceRepo: repository.NewRepoSyncSourceRepository(db),
		teamRepo:   repository.NewTeamRepository(db),
		logger:     logger,
		aesKey:     aesKey,
	}
}

// SyncAllSources 同步所有启用的代码库源
func (s *RepoSyncService) SyncAllSources() error {
	sources, err := s.sourceRepo.ListEnabled()
	if err != nil {
		return err
	}

	if len(sources) == 0 {
		s.logger.Info("没有启用的仓库源，跳过同步")
		return nil
	}

	s.logger.Info("开始同步仓库源", zap.Int("source_count", len(sources)))

	for _, source := range sources {
		success, failed, syncErr := s.SyncFromSource(source)
		status := "success"
		message := fmt.Sprintf("同步完成: 成功 %d 个, 失败 %d 个", success, failed)

		if syncErr != nil || failed > 0 {
			status = "failed"
			if syncErr != nil {
				message = fmt.Sprintf("同步失败: %v", syncErr)
			}
		}

		if err := s.sourceRepo.UpdateSyncResult(source.ID, status, &message); err != nil {
			s.logger.Warn("更新同步结果失败", zap.Int64("source_id", source.ID), zap.Error(err))
		}
	}

	return nil
}

// SyncFromSource 从单个源同步代码库
func (s *RepoSyncService) SyncFromSource(source *model.RepoSource) (int, int, error) {
	s.logger.Info("开始同步代码库源",
		zap.Int64("source_id", source.ID),
		zap.String("platform", source.Platform),
		zap.String("namespace", source.Namespace))

	gitClient, err := s.buildGitClient(source)
	if err != nil {
		return 0, 0, fmt.Errorf("创建 Git 客户端失败: %w", err)
	}

	repos, err := gitClient.ListRepositories(source.Namespace)
	if err != nil {
		return 0, 0, fmt.Errorf("获取仓库列表失败: %w", err)
	}

	successCount := 0
	failedCount := 0

	for _, repoInfo := range repos {
		if err := s.syncRepository(&repoInfo, source); err != nil {
			s.logger.Error("同步仓库失败", zap.String("repo", repoInfo.FullName), zap.Error(err))
			failedCount++
			continue
		}
		successCount++
	}

	return successCount, failedCount, nil
}

func (s *RepoSyncService) syncRepository(repoInfo *api.RepositoryInfo, source *model.RepoSource) error {
	repo := &model.Repository{
		Namespace:   repoInfo.Owner,
		Name:        repoInfo.Name,
		Description: &repoInfo.Description,
		GitURL:      repoInfo.CloneURL,
		GitType:     source.Platform,
		Language:    &repoInfo.Language,
	}

	// 自动设置默认项目和团队
	if source.DefaultProjectID != nil && *source.DefaultProjectID > 0 {
		repo.ProjectID = source.DefaultProjectID

		// 如果设置了默认团队，需要验证团队是否属于该项目
		if source.DefaultTeamID != nil && *source.DefaultTeamID > 0 {
			belongs, err := s.teamRepo.VerifyTeamBelongsToProject(*source.DefaultTeamID, *source.DefaultProjectID)
			if err != nil {
				s.logger.Warn("验证团队归属失败",
					zap.Int64("team_id", *source.DefaultTeamID),
					zap.Int64("project_id", *source.DefaultProjectID),
					zap.Error(err))
			} else if !belongs {
				s.logger.Warn("团队不属于指定项目，跳过团队设置",
					zap.Int64("team_id", *source.DefaultTeamID),
					zap.Int64("project_id", *source.DefaultProjectID),
					zap.String("repo", repoInfo.FullName))
			} else {
				// 团队验证通过，设置团队ID
				repo.TeamID = source.DefaultTeamID
			}
		}
	} else if source.DefaultTeamID != nil && *source.DefaultTeamID > 0 {
		// 如果只设置了团队没设置项目，记录警告
		s.logger.Warn("未设置默认项目，跳过团队设置",
			zap.Int64("source_id", source.ID),
			zap.String("repo", repoInfo.FullName))
	}

	return s.repoRepo.Upsert(repo)
}

// SyncSourceByID 手动同步某个源
func (s *RepoSyncService) SyncSourceByID(id int64) (int, int, error) {
	source, err := s.sourceRepo.GetByID(id)
	if err != nil {
		return 0, 0, err
	}

	success, failed, syncErr := s.SyncFromSource(source)
	status := "success"
	message := fmt.Sprintf("同步完成: 成功 %d 个, 失败 %d 个", success, failed)

	if syncErr != nil || failed > 0 {
		status = "failed"
		if syncErr != nil {
			message = fmt.Sprintf("同步失败: %v", syncErr)
		}
	}

	if err := s.sourceRepo.UpdateSyncResult(source.ID, status, &message); err != nil {
		s.logger.Warn("更新同步结果失败", zap.Int64("source_id", source.ID), zap.Error(err))
	}

	return success, failed, syncErr
}

// TestSourceConnection 测试某个源的连接
func (s *RepoSyncService) TestSourceConnection(id int64) error {
	source, err := s.sourceRepo.GetByID(id)
	if err != nil {
		return err
	}

	client, err := s.buildGitClient(source)
	if err != nil {
		return err
	}

	if err := client.TestConnection(); err != nil {
		return fmt.Errorf("连接测试失败: %w", err)
	}

	if _, err := client.ListRepositories(source.Namespace); err != nil {
		return fmt.Errorf("命名空间仓库列表获取失败: %w", err)
	}

	return nil
}

func (s *RepoSyncService) buildGitClient(source *model.RepoSource) (*git.Client, error) {
	token, err := utils.DecryptSecret(s.aesKey, source.AuthTokenEnc)
	if err != nil {
		return nil, err
	}

	return git.NewClient(source.BaseURL, token, source.Platform)
}
