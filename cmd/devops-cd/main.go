package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"devops-cd/internal/api/router"
	"devops-cd/internal/core"
	"devops-cd/internal/pkg/config"
	"devops-cd/internal/pkg/database"
	"devops-cd/internal/pkg/logger"
	"devops-cd/internal/scheduler"

	_ "devops-cd/docs" // Swagger docs
)

// @title DevOps CD API
// @version 1.0
// @description DevOps 持续交付平台 API 文档
// @description 提供代码库管理、应用管理、批次管理、部署管理等功能

// @contact.name API Support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

var (
	configFile = flag.String("config", "", "配置文件路径 (例如: -config=configs/config.yaml)")
	version    = flag.Bool("version", false, "显示版本信息")
)

const (
	appVersion = "1.0.0"
	appName    = "devops-cd-base-service"
)

func main() {
	// 解析命令行参数
	flag.Parse()

	// 显示版本信息
	if *version {
		fmt.Printf("%s version %s\n", appName, appVersion)
		os.Exit(0)
	}

	// init config logger
	var cfg *config.Config
	{
		// 优先级: 命令行参数 > 环境变量 > 默认路径
		configPath := getConfigPath()

		// 加载配置
		c, err := config.Load(configPath)
		if err != nil {
			fmt.Printf("加载配置失败: %v\n", err)
			fmt.Println("\n使用方式:")
			fmt.Println("  1. 命令行参数指定:")
			fmt.Println("     ./base -config=configs/base.yaml")
			fmt.Println("  2. 环境变量指定:")
			fmt.Println("     export CONFIG_FILE=configs/base.yaml")
			fmt.Println("     ./base")
			fmt.Println("  3. 使用默认配置:")
			fmt.Println("     ./base  (将使用 configs/base.yaml)")
			os.Exit(1)
		}
		cfg = c

		// 初始化日志
		if err := logger.Init(&cfg.Log); err != nil {
			fmt.Printf("初始化日志失败: %v\n", err)
			os.Exit(1)
		}
		logger.Info(fmt.Sprintf("Load config file: %s of %s", configPath, getConfigSource()))

		defer func() {
			_ = logger.Close()
		}()
	}

	logger.Info(fmt.Sprintf("服务 %s 启动中...", appName), zap.String("version", appVersion))

	// 初始化数据库
	if err := database.Init(&cfg.Database); err != nil {
		logger.Fatal("初始化数据库失败", zap.Error(err))
	}
	defer func() {
		_ = database.Close()
	}()

	logger.Info(fmt.Sprintf("数据库连接成功 %s:%v", cfg.Database.Host, cfg.Database.Port), zap.String("database", cfg.Database.Database))

	// 注入数据库连接到配置
	cfg.DB = database.GetDB()

	// 初始化Core引擎（状态机）
	coreEngine := core.NewCoreEngine(database.GetDB(), logger.Log, &cfg.Core)

	// 解析扫描间隔
	scanInterval, err := time.ParseDuration(cfg.Core.ScanInterval)
	if err != nil {
		logger.Warn("解析扫描间隔失败，使用默认值30秒", zap.Error(err))
		scanInterval = 30 * time.Second
	}

	// 启动Core引擎
	coreEngine.Start(scanInterval)
	logger.Info("Core引擎启动成功", zap.Duration("scan_interval", scanInterval))

	// 初始化并启动定时任务调度器
	taskScheduler := scheduler.NewScheduler(database.GetDB(), logger.Log, cfg)
	if err := taskScheduler.Start(cfg); err != nil {
		logger.Warn("定时任务调度器启动失败", zap.Error(err))
	}

	// 设置路由
	r := router.Setup(cfg, coreEngine, logger.Log)

	// 创建HTTP服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// 启动服务器
	go func() {
		logger.Info(fmt.Sprintf("%s 服务启动成功", cfg.Server.Name),
			zap.String("address", addr),
			zap.String("mode", cfg.Server.Mode),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("服务器启动失败", zap.Error(err))
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("服务正在关闭...")

	// 关闭定时任务调度器
	taskScheduler.Stop()
	logger.Info("定时任务调度器已停止")

	// 关闭Core引擎
	coreEngine.Stop()
	logger.Info("Core引擎已停止")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("服务器关闭异常", zap.Error(err))
	}

	logger.Info("服务已关闭")
}

// getConfigPath 获取配置文件路径
// 优先级: 命令行参数 > 环境变量 > 默认路径
func getConfigPath() string {
	// 1. 命令行参数
	if *configFile != "" {
		return *configFile
	}

	// 2. 环境变量
	if envConfig := os.Getenv("CONFIG_FILE"); envConfig != "" {
		return envConfig
	}

	// 3. 默认路径
	return "configs/config.yaml"
}

// getConfigSource 获取配置来源说明
func getConfigSource() string {
	if *configFile != "" {
		return "命令行参数"
	}
	if os.Getenv("CONFIG_FILE") != "" {
		return "环境变量"
	}
	return "默认配置"
}
