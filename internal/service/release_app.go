package service

import (
	"devops-cd/internal/dto"
	"devops-cd/internal/pkg/logger"
	"fmt"
	"go.uber.org/zap"
)

// GetReleaseApp 获取单个发布应用详情
func (s *BatchService) GetReleaseApp(releaseAppID int64) (*dto.ReleaseAppResponse, error) {
	log := logger.Log.With(zap.Int64("release_app_id", releaseAppID)).Sugar()

	// 1. 获取 release_app 记录（包含关联信息）
	release, err := s.batchRepo.GetReleaseAppByID(releaseAppID)
	if err != nil {
		return nil, fmt.Errorf("发布应用不存在: %w", err)
	}

	releaseResp := &dto.ReleaseAppResponse{
		ID:      release.ID,
		BatchID: release.BatchID,
		AppID:   release.AppID,
		BuildID: release.BuildID,

		// 版本信息
		LatestBuildID:       release.LatestBuildID,
		PreviousDeployedTag: release.PreviousDeployedTag,
		TargetTag:           release.TargetTag,

		// 发布信息
		ReleaseNotes: release.ReleaseNotes,
		IsLocked:     release.IsLocked,
		Reason:       release.Reason,
		Status:       release.Status,

		CreatedAt: release.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: release.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// 1.1 填充应用信息
	if release.Application != nil {
		releaseResp.AppName = release.Application.Name
		releaseResp.AppDisplayName = release.Application.DisplayName
		releaseResp.AppType = release.Application.AppType
		releaseResp.AppProject = release.Application.Project
		releaseResp.TeamID = release.Application.TeamID
		releaseResp.DeployedTag = release.Application.DeployedTag // 当前部署的标签
		releaseResp.DefaultDependsOn = release.Application.DefaultDependsOn

		// 1.2 填充仓库信息
		if release.Application.Repository != nil {
			releaseResp.RepoID = release.Application.RepoID
			releaseResp.RepoName = release.Application.Repository.Name
			releaseResp.RepoFullName = release.Application.Repository.Project + "/" + release.Application.Repository.Name
		}

		// 1.3 填充团队信息
		if release.Application.Team != nil {
			releaseResp.TeamName = &release.Application.Team.Name
		}
	}

	// 2. 加载构建信息
	if release.BuildID != nil {
		build, err := s.batchRepo.GetBuildByID(*release.BuildID)
		if err != nil {
			log.Errorf("查询构建失败: %v", err)
		}

		if build != nil {
			releaseResp.BuildNumber = &build.BuildNumber
			releaseResp.BuildStatus = &build.BuildStatus
			buildTime := build.BuildCreated.Format("2006-01-02T15:04:05Z07:00")
			releaseResp.BuildTime = &buildTime
			releaseResp.ImageURL = &build.ImageURL
			releaseResp.CommitSHA = &build.CommitSHA
			releaseResp.CommitMessage = &build.CommitMessage
			releaseResp.CommitBranch = &build.CommitBranch

			// 2.1 加载最新的构建记录
			if builds, err := s.batchRepo.GetBuildsSinceTime(release.AppID, build.CreatedAt, 10); err != nil {
				log.Errorf("查询最近构建失败: %v", err)
			} else {
				releaseResp.RecentBuilds = s.toBuildSummaries(builds)
			}
		} else {
			// 2.1 加载最近的构建记录
			if builds, err := s.batchRepo.GetRecentBuilds(release.AppID, 10); err != nil {
				log.Errorf("查询最近构建失败: %v", err)
			} else {
				releaseResp.RecentBuilds = s.toBuildSummaries(builds)
			}
		}
	}

	return releaseResp, nil
}
