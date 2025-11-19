package scheduler

import (
	"devops-cd/internal/pkg/config"
	"devops-cd/internal/service"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Scheduler 调度器
type Scheduler struct {
	cron          *cron.Cron
	logger        *zap.Logger
	repoSyncSvc   *service.RepoSyncService
	cronSchedules map[string]cron.EntryID // 存储任务ID，便于管理
}

// NewScheduler 创建调度器
func NewScheduler(db *gorm.DB, logger *zap.Logger, cfg *config.Config) *Scheduler {
	// 创建 cron 实例（带秒级支持）
	c := cron.New(cron.WithSeconds())

	return &Scheduler{
		cron:          c,
		logger:        logger,
		repoSyncSvc:   service.NewRepoSyncService(db, logger, &cfg.Repo),
		cronSchedules: make(map[string]cron.EntryID),
	}
}

// Start 启动调度器
func (s *Scheduler) Start(cfg *config.Config) error {
	log := s.logger.Sugar()

	log.Info("启动定时任务调度器...")

	// 获取配置的 cron 表达式，默认每天凌晨2点执行
	// cron 表达式格式: 秒 分 时 日 月 周
	cronExpr := cfg.Repo.Cron
	if cronExpr == "" {
		cronExpr = "0 0 2 * * *" // 默认: 每天凌晨2点
		log.Warn("未配置repo.cron，使用默认值", zap.String("cron", cronExpr))
	}

	entryID, err := s.cron.AddFunc(cronExpr, func() {
		log.Info("执行定时任务: 代码库同步")
		if err := s.repoSyncSvc.SyncAllSources(); err != nil {
			log.Errorf("代码库同步任务执行失败: %v", err)
		}
	})

	if err != nil {
		log.Errorf("注册代码库: %v 同步任务失败: %v", cronExpr, err)
		return err
	}

	s.cronSchedules["repo_sync"] = entryID
	log.Infof("代码库同步任务已注册: %s entry_id=%d", cronExpr, entryID)

	// 启动 cron
	s.cron.Start()
	log.Info("定时任务调度器启动成功")

	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.logger.Info("正在停止定时任务调度器...")

	// 停止 cron（等待正在执行的任务完成）
	ctx := s.cron.Stop()
	<-ctx.Done()

	s.logger.Info("定时任务调度器已停止")
}

// TriggerRepoSync 手动触发代码库同步（用于测试或手动触发）
func (s *Scheduler) TriggerRepoSync() error {
	s.logger.Info("手动触发代码库同步")
	return s.repoSyncSvc.SyncAllSources()
}
