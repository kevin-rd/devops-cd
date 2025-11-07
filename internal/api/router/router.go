package router

import (
	"devops-cd/internal/api/handler"
	"devops-cd/internal/api/middleware"
	"devops-cd/internal/core"
	"devops-cd/internal/pkg/config"
	"devops-cd/internal/repository"
	"devops-cd/internal/service"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// Setup 设置路由
func Setup(cfg *config.Config, coreEngine *core.CoreEngine) *gin.Engine {
	// 设置Gin模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// 全局中间件
	r.Use(gin.Recovery())
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.CORSMiddleware())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Swagger API 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 获取数据库连接
	db := cfg.DB.(*gorm.DB)

	// 初始化Repository
	userRepo := repository.NewUserRepository()
	repositoryRepo := repository.NewRepositoryRepository(db)
	applicationRepo := repository.NewApplicationRepository(db)
	buildRepo := repository.NewBuildRepository(db)

	// 初始化Service
	ldapService := service.NewLDAPService(&cfg.Auth.LDAP)
	authService := service.NewAuthService(&cfg.Auth, userRepo, ldapService)
	repositoryService := service.NewRepositoryService(repositoryRepo, applicationRepo)
	applicationService := service.NewApplicationService(applicationRepo, repositoryRepo, db)
	batchService := service.NewBatchService(db)
	buildService := service.NewBuildService(db, buildRepo, repositoryRepo, applicationRepo)

	// 初始化Handler
	authHandler := handler.NewAuthHandler(authService)
	repositoryHandler := handler.NewRepositoryHandler(repositoryService)
	applicationHandler := handler.NewApplicationHandler(applicationService)
	batchHandler := handler.NewBatchHandler(coreEngine, batchService)
	buildHandler := handler.NewBuildHandler(buildService, batchService)
	releaseAppHandler := handler.NewReleaseAppHandler(batchService)

	// API v1
	v1 := r.Group("/api/v1")
	{
		// 认证相关(无需token)
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/refresh", authHandler.Refresh)
		}

		// 需要认证的路由
		authed := v1.Group("")
		authed.Use(middleware.AuthMiddleware())
		{
			// 认证信息
			authed.GET("/auth/me", authHandler.GetMe)
			authed.GET("/auth/verify", authHandler.Verify)

			// 代码库管理
			groupRepository := authed.Group("/repository")
			groupRepositories := authed.Group("/repositories")
			{
				groupRepository.POST("", repositoryHandler.Create)        // 创建代码库
				groupRepositories.GET("", repositoryHandler.List)         // 列表查询
				groupRepository.GET("", repositoryHandler.GetByID)        // 获取详情（query参数id，包含应用列表）
				groupRepository.PUT("", repositoryHandler.Update)         // 更新代码库（JSON包含id）
				groupRepository.POST("/delete", repositoryHandler.Delete) // 删除代码库（软删除，JSON包含id）
			}

			// 应用管理
			groupApplication := authed.Group("/application")
			groupApplications := authed.Group("/applications")
			{
				groupApplication.POST("", applicationHandler.Create)                             // 创建应用
				groupApplications.GET("", applicationHandler.List)                               // 列表查询
				groupApplication.GET("", applicationHandler.GetByID)                             // 获取详情（query参数id）
				groupApplication.PUT("", applicationHandler.Update)                              // 更新应用（JSON包含id）
				groupApplication.POST("/delete", applicationHandler.Delete)                      // 删除应用（软删除，JSON包含id）
				groupApplication.GET("/builds", applicationHandler.GetBuilds)                    // 获取构建历史（query参数id）
				groupApplication.GET("/types", applicationHandler.GetAppTypes)                   // 获取应用类型列表
				groupApplication.GET("/:id/dependencies", applicationHandler.GetDependencies)    // 获取默认依赖
				groupApplication.PUT("/:id/dependencies", applicationHandler.UpdateDependencies) // 更新默认依赖
				authed.GET("/application_builds", applicationHandler.SearchWithBuilds)           // 搜索应用（包含构建信息，支持模糊查询）
			}

			// 批次管理
			groupBatch := authed.Group("/batch")
			groupBatches := authed.Group("/batches")
			{
				// 写操作（POST/PUT）
				groupBatch.POST("", batchHandler.Create)                  // 创建批次
				groupBatch.PUT("", batchHandler.Update)                   // 更新批次
				groupBatch.POST("/delete", batchHandler.Delete)           // 软删除
				groupBatch.PUT("/release_app", batchHandler.UpdateBuilds) // 更新发布应用（构建版本等）

				// 读操作（GET）
				groupBatch.GET("", batchHandler.Get)       // 获取详情（query: id）
				groupBatch.GET("/status", batchHandler.GetStatus) // 获取批次状态（轻量级，用于轮询）
				groupBatches.GET("", batchHandler.List)    // 列表查询（query: page, page_size, status, initiator）

				// 审批操作
				groupBatch.POST("/approve", batchHandler.Approve) // 审批通过
				groupBatch.POST("/reject", batchHandler.Reject)   // 审批拒绝

				// 状态操作
				groupBatch.POST("/action", batchHandler.ProcessAction) // 状态流转
			}

			// 发布应用配置
			releaseAppGroup := authed.Group("/release_app")
			{
				releaseAppGroup.PUT(":id/dependencies", releaseAppHandler.UpdateDependencies)
			}

			// 构建记录管理
			groupBuild := authed.Group("/build")
			groupBuilds := authed.Group("/builds")
			{
				groupBuilds.GET("", buildHandler.List)                 // 列表查询
				groupBuild.GET("", buildHandler.GetByID)               // 获取详情（query参数id）
				groupBuild.GET("/app", buildHandler.GetByAppAndNumber) // 按应用和构建号查询
			}
		}

		// 构建通知（无需认证，由Drone调用）
		v1.POST("/build/notify", buildHandler.Notify)
	}

	return r
}
