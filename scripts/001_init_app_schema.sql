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
  `project` VARCHAR(63) NOT NULL COMMENT '项目名称(继承自repository.project)',    --- todo:
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
  UNIQUE KEY `uk_project_app_name` (`project`, `name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='应用表';

-- =====================================================
-- 3. 环境表 (environments)
-- =====================================================
CREATE TABLE IF NOT EXISTS `environments` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
    `name` VARCHAR(63) NOT NULL UNIQUE COMMENT '环境名称(dev/test/staging/prod)',
    `description` TEXT DEFAULT NULL COMMENT '环境描述',
    `env_type` VARCHAR(63) NOT NULL COMMENT '环境类型(dev/test/staging/prod)',
    `cluster_name` VARCHAR(63) DEFAULT NULL COMMENT '集群名称',
    `cluster_url` VARCHAR(255) DEFAULT NULL COMMENT '集群API地址',
    `cluster_token` VARCHAR(1000) DEFAULT NULL COMMENT '集群访问Token(加密存储)',
    `namespace` VARCHAR(63) DEFAULT NULL COMMENT 'K8s命名空间',
    `priority` INT NOT NULL DEFAULT 0 COMMENT '优先级(用于排序)',
    `require_approval` BOOLEAN NOT NULL DEFAULT FALSE COMMENT '是否需要审批',
    `auto_deploy` BOOLEAN NOT NULL DEFAULT FALSE COMMENT '是否自动部署',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态(1:启用 0:禁用)',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间',
    INDEX `idx_name` (`name`),
    INDEX `idx_env_type` (`env_type`),
    INDEX `idx_priority` (`priority`),
    INDEX `idx_status` (`status`),
    INDEX `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='环境表';

-- 插入默认环境
INSERT INTO `environments` (`name`, `display_name`, `description`, `env_type`, `priority`, `require_approval`, `auto_deploy`, `status`)
VALUES
    ('pre', '预发布环境', '用于上线前的最终验证', 'pre', 3, TRUE, FALSE, 1),
    ('prod', '生产环境', '正式生产环境', 'prod', 4, TRUE, FALSE, 1)
    ON DUPLICATE KEY UPDATE `name` = `name`;

-- =====================================================
-- 4. 表关系说明
-- =====================================================
-- repositories (1) ----< (N) applications: 一个代码库可以有多个应用
-- applications (N) ----< (N) environments: 应用和环境通过app_env_configs表建立多对多关系

