package router

import (
	"devops-cd/internal/api/handler"
	"devops-cd/internal/api/middleware"
	"devops-cd/internal/core"
	"devops-cd/internal/pkg/auth"
	"devops-cd/internal/pkg/config"
	"devops-cd/internal/repository"
	"devops-cd/internal/service"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Setup 设置路由
func Setup(cfg *config.Config, coreEngine *core.CoreEngine, logger *zap.Logger) *gin.Engine {
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
	repoSyncSourceRepo := repository.NewRepoSyncSourceRepository(db)
	applicationRepo := repository.NewApplicationRepository(db)
	appEnvConfigRepo := repository.NewAppEnvConfigRepository(db)
	buildRepo := repository.NewBuildRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	projectEnvConfigRepo := repository.NewProjectEnvConfigRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	teamMemberRepo := repository.NewTeamMemberRepository(db)
	authz = service.NewAuthorizationService(userRepo, teamMemberRepo)

	// 初始化Service
	ldapService := service.NewLDAPService(&cfg.Auth.LDAP)
	authService := service.NewAuthService(&cfg.Auth, userRepo, ldapService)
	userService := service.NewUserService(userRepo)
	projectService := service.NewProjectService(projectRepo, teamRepo, projectEnvConfigRepo)
	teamService := service.NewTeamService(teamRepo, projectRepo)
	teamMemberService := service.NewTeamMemberService(logger, teamMemberRepo, teamRepo, userRepo)
	repositoryService := service.NewRepositoryService(repositoryRepo, applicationRepo)
	repoSourceService := service.NewRepoSourceService(repoSyncSourceRepo, teamRepo, cfg.Crypto.AESKey)
	repoSyncService := service.NewRepoSyncService(db, logger, cfg.Crypto.AESKey)
	applicationService := service.NewApplicationService(applicationRepo, repositoryRepo, db, logger)
	appEnvConfigService := service.NewAppEnvConfigService(appEnvConfigRepo, applicationRepo, db)
	clusterService := service.NewClusterService(db)
	batchService := service.NewBatchService(db)
	buildService := service.NewBuildService(buildRepo, repositoryRepo, applicationRepo, coreEngine)

	// 初始化Handler
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	projectHandler := handler.NewProjectHandler(projectService)
	teamHandler := handler.NewTeamHandler(teamService)
	teamMemberHandler := handler.NewTeamMemberHandler(teamMemberService)
	repositoryHandler := handler.NewRepositoryHandler(repositoryService)
	repoSourceHandler := handler.NewRepoSourceHandler(repoSourceService, repoSyncService)
	applicationHandler := handler.NewApplicationHandler(applicationService)
	appEnvConfigHandler := handler.NewAppEnvConfigHandler(appEnvConfigService)
	clusterHandler := handler.NewClusterHandler(clusterService)
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
			authed.GET("/users/search", userHandler.Search)
			authed.GET("/roles", userHandler.ListRoles)

			// 项目管理
			groupProject := authed.Group("/project")
			groupProjects := authed.Group("/projects")
			{
				groupProject.POST("", projectHandler.Create)                                         // 创建项目
				groupProjects.GET("", projectHandler.List)                                           // 列表查询（无参数返回全部，有分页参数返回分页数据）
				groupProject.GET("", projectHandler.GetByID)                                         // 获取详情（支持 with_teams 参数）
				groupProject.PUT("", projectHandler.Update)                                          // 更新项目
				groupProject.DELETE("/:id", projectHandler.Delete)                                   // 删除项目
				groupProjects.GET("/available-env-clusters", projectHandler.GetAvailableEnvClusters) // 获取项目可用的环境集群配置

				// 项目环境配置管理（作为项目的附属资源）
				groupProject.GET("/:id/env", projectHandler.GetEnvConfigs)    // 获取项目的环境配置
				groupProject.PUT("/:id/env", projectHandler.UpdateEnvConfigs) // 批量更新项目的环境配置
			}

			// 团队管理
			groupTeam := authed.Group("/team")
			groupTeams := authed.Group("/teams")
			{
				groupTeam.POST("", teamHandler.Create)       // 创建团队
				groupTeams.GET("", teamHandler.List)         // 列表查询（返回所有团队）
				groupTeam.GET("", teamHandler.GetByID)       // 获取详情
				groupTeam.PUT("", teamHandler.Update)        // 更新团队
				groupTeam.DELETE("/:id", teamHandler.Delete) // 删除团队
			}

			teamMemberGroup := authed.Group("/team_members")
			{
				teamMemberGroup.POST("", teamMemberHandler.AddMember)          // 添加成员
				teamMemberGroup.GET("", teamMemberHandler.ListMembers)         // 成员列表
				teamMemberGroup.PUT("/:id/role", teamMemberHandler.UpdateRole) // 更新角色
				teamMemberGroup.DELETE("/:id", teamMemberHandler.DeleteMember) // 移除成员
			}

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

			// 仓库源管理
			repoSourcesGroup := authed.Group("/repo-sources")
			{
				repoSourcesGroup.GET("", repoSourceHandler.List)
				repoSourcesGroup.POST("", repoSourceHandler.Create)
				repoSourcesGroup.PUT("", repoSourceHandler.Update)
				repoSourcesGroup.DELETE("/:id", repoSourceHandler.Delete)
				repoSourcesGroup.POST("/:id/test", repoSourceHandler.TestConnection)
				repoSourcesGroup.POST("/:id/sync", repoSourceHandler.SyncNow)
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

			// 应用环境配置管理
			appEnvConfigGroup := authed.Group("/app-env-configs")
			{
				appEnvConfigGroup.POST("", appEnvConfigHandler.Create)            // 创建应用环境配置
				appEnvConfigGroup.GET("", appEnvConfigHandler.List)               // 查询应用环境配置列表
				appEnvConfigGroup.GET("/:id", appEnvConfigHandler.GetByID)        // 获取应用环境配置详情
				appEnvConfigGroup.PUT("/:id", appEnvConfigHandler.Update)         // 更新应用环境配置
				appEnvConfigGroup.DELETE("/:id", appEnvConfigHandler.Delete)      // 删除应用环境配置
				appEnvConfigGroup.POST("/batch", appEnvConfigHandler.BatchCreate) // 批量创建应用环境配置
			}

			// 集群管理
			clusterGroup := authed.Group("/clusters")
			{
				clusterGroup.POST("", clusterHandler.Create)       // 创建集群
				clusterGroup.GET("", clusterHandler.List)          // 查询集群列表
				clusterGroup.GET("/:id", clusterHandler.Get)       // 获取集群详情
				clusterGroup.PUT("/:id", clusterHandler.Update)    // 更新集群
				clusterGroup.DELETE("/:id", clusterHandler.Delete) // 删除集群
			}

			// 批次管理
			groupBatch := authed.Group("/batch")
			groupBatches := authed.Group("/batches")
			{
				// 写操作（POST/PUT）
				groupBatch.POST("", ProjectAuthWrapper(batchHandler.Create, auth.PermBatchCreate)) // 创建批次
				groupBatch.PUT("", ProjectAuthWrapper(batchHandler.Update, auth.PermBatchUpdate))  // 更新批次
				groupBatch.POST("/delete", batchHandler.Delete)                                    // 软删除
				groupBatch.PUT("/release_app", releaseAppHandler.UpdateBuilds)                     // 更新发布应用（构建版本等）

				// 读操作（GET）
				groupBatch.GET("", batchHandler.Get)              // 获取详情（query: id）
				groupBatch.GET("/status", batchHandler.GetStatus) // 获取批次状态（轻量级，用于轮询）
				groupBatches.GET("", batchHandler.List)           // 列表查询（query: page, page_size, status, initiator）

				// 审批操作
				groupBatch.POST("/approve", batchHandler.Approve) // 审批通过
				groupBatch.POST("/reject", batchHandler.Reject)   // 审批拒绝

				// 状态操作
				groupBatch.POST("/action", batchHandler.ProcessAction) // 状态流转
			}

			// 发布应用配置
			releaseAppGroup := authed.Group("/release_app")
			{
				releaseAppGroup.GET("", releaseAppHandler.GetByID) // 获取发布应用详情
				releaseAppGroup.PUT(":id/dependencies", releaseAppHandler.UpdateDependencies)
				releaseAppGroup.POST("/switch_version", batchHandler.SwitchVersion) // 切换版本
				releaseAppGroup.POST("/manual_deploy", batchHandler.ManualDeploy)   // 手动部署
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
