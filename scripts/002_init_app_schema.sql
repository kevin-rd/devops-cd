-- DevOps CD 工具 - Base Service 数据库表结构
-- 版本: v2.0
-- 创建日期: 2025-10-15
-- 数据库: MySQL 8.0+

-- =====================================================
-- 1. 代码库表 (repositories)
-- =====================================================
CREATE TABLE IF NOT EXISTS `repositories` (
  `id` BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT 'ID',
  `project` VARCHAR(63) NOT NULL COMMENT 'user/org 如(my_org)', --- todo: rename to namespace
  `name` VARCHAR(63) NOT NULL COMMENT '仓库名称(如: my_repo)',
  `description` TEXT DEFAULT NULL COMMENT '代码库描述',
  `git_url` VARCHAR(255) NOT NULL COMMENT 'Git仓库地址',
  `git_type` VARCHAR(63) NOT NULL COMMENT 'Git类型(gitlab/github/gitea等)',
  `language` VARCHAR(255) DEFAULT NULL COMMENT '主要编程语言',
  `team_id` BIGINT DEFAULT NULL COMMENT '所属团队ID',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态(1:启用 0:禁用)',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间',
  UNIQUE INDEX `idx_project_name` (`project`, `name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='代码库表';


-- =====================================================
-- 2. 应用表 (applications)
-- =====================================================
CREATE TABLE IF NOT EXISTS `applications` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `name` VARCHAR(63) NOT NULL COMMENT '应用名称',   --- 期望 RFC1035
  `project_id` VARCHAR(63) NOT NULL COMMENT '项目名称(继承自repository.project)',    --- todo:
  `description` TEXT COMMENT '应用描述',
  `repo_id` BIGINT NOT NULL COMMENT '关联的代码库ID',
  `app_type` VARCHAR(63) NOT NULL COMMENT '应用类型(web/api/job/microservice等)',
  `team_id` BIGINT DEFAULT NULL COMMENT '所属团队ID',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态(1:启用 0:禁用)',
  `deployed_tag` VARCHAR(63) DEFAULT NULL COMMENT '当前线上部署的tag（ProdDeployed时更新）',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_project_name` (`project_id`, `name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='应用表';


-- =====================================================
-- 3. 应用环境配置表
-- 用途: 通过记录存在判断应用是否需要部署到某环境
-- 设计:
--   - 每个集群一条记录,支持不同集群独立配置
--   - 有 env='pre' 记录 -> 应用需要部署到 pre
--   - 无 env='pre' 记录 -> 应用跳过 pre,直接到 prod
--
-- 示例:
--   app_id=1, env='prod', cluster='cluster-a', replicas=2
--   app_id=1, env='prod', cluster='cluster-b', replicas=3
-- =====================================================
CREATE TABLE IF NOT EXISTS `app_env_configs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `app_id` BIGINT NOT NULL COMMENT '应用ID,关联 applications.id',
  `env` VARCHAR(20) NOT NULL COMMENT '环境名称: pre/prod/dev/test/uat 等',
  `cluster` VARCHAR(50) NOT NULL DEFAULT 'default' COMMENT '集群名称',

  -- 部署配置
  `replicas` INT DEFAULT 1 COMMENT '副本数量',
  `config_data` JSON DEFAULT NULL COMMENT '环境专属配置(JSON格式,用于存储扩展配置)',

  -- 系统字段
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态(1:启用 0:禁用,用于临时禁用配置)',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_app_env_cluster` (`app_id`, `env`, `cluster`),
  INDEX `idx_app_env` (`app_id`, `env`),
  INDEX `idx_status` (`status`),
  INDEX `idx_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_app_env_configs_cluster` FOREIGN KEY (`cluster`) REFERENCES `clusters` (`name`) ON DELETE RESTRICT ON UPDATE CASCADE
  #CONSTRAINT `fk_app_env_configs_app` FOREIGN KEY (`app_id`) REFERENCES `applications` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='应用环境配置表';
