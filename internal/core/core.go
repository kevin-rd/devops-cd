package core

import (
	"devops-cd/internal/adapter/deploy"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CoreEngine CD核心引擎
type CoreEngine struct {
	db       *gorm.DB
	scanner  *BatchScanner
	logger   *zap.Logger
	running  bool
	stopChan chan struct{}
}

// NewCoreEngine 创建核心引擎
func NewCoreEngine(db *gorm.DB, logger *zap.Logger) *CoreEngine {

	// 创建部署服务
	// deployService := deploy.NewK8sDeployer( deploy.NewMockK8sDeployClient(), notification.NewLogNotifier(logger), db, nil, logger)
	deployService := deploy.NewMockDeployer()

	// 创建扫描器
	scanner := NewBatchScanner(db, deployService, logger)

	return &CoreEngine{
		db:       db,
		scanner:  scanner,
		logger:   logger,
		stopChan: make(chan struct{}),
	}
}

// Start 启动核心引擎
func (e *CoreEngine) Start(scanInterval time.Duration) {
	if e.running {
		e.logger.Warn("核心引擎已在运行中")
		return
	}

	e.running = true
	e.logger.Info("CoreEngine starting...", zap.Duration("scan_interval", scanInterval))

	// 启动定时扫描
	go e.runScanner(scanInterval)
}

// Stop 停止核心引擎
func (e *CoreEngine) Stop() {
	if !e.running {
		return
	}

	e.logger.Info("正在停止核心引擎...")
	close(e.stopChan)
	e.running = false
	e.logger.Info("核心引擎已停止")
}

// runScanner 运行扫描器
func (e *CoreEngine) runScanner(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.scanner.ScanBatches()
		case <-e.stopChan:
			return
		}
	}
}

// ProcessBatchEvent 处理批次事件
func (e *CoreEngine) ProcessBatchEvent(batchID int64, event string, operator string) error {
	return e.scanner.ProcessBatchStateChange(batchID, event, operator)
}

// GetBatchStatus 获取批次状态
func (e *CoreEngine) GetBatchStatus(batchID int64) (map[string]interface{}, error) {
	return e.scanner.GetBatchStatus(batchID)
}
