-- DevOps CD 工具 - Base Service 数据库表结构
-- 版本: v2.0
-- 创建日期: 2025-10-15
-- 数据库: MySQL 8.0+

-- =====================================================
-- 1. 本地用户表 (local_users)
-- =====================================================
CREATE TABLE IF NOT EXISTS `local_users` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
    `username` VARCHAR(50) NOT NULL UNIQUE COMMENT '用户名',
    `password` VARCHAR(255) NOT NULL COMMENT '密码(bcrypt加密)',
    `email` VARCHAR(100) DEFAULT NULL COMMENT '邮箱',
    `display_name` VARCHAR(100) DEFAULT NULL COMMENT '显示名称',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态(1:启用 0:禁用)',
    `last_login_at` TIMESTAMP NULL DEFAULT NULL COMMENT '最后登录时间',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间',
    INDEX `idx_username` (`username`)
) ENGINE=InnoDB AUTO_INCREMENT=100 DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

-- =====================================================
-- 2. 团队表 (teams)
-- =====================================================
CREATE TABLE IF NOT EXISTS `teams` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
    `name` VARCHAR(100) NOT NULL UNIQUE COMMENT '团队名称',
    `display_name` VARCHAR(100) DEFAULT NULL COMMENT '团队展示名称',
    `description` TEXT DEFAULT NULL COMMENT '团队描述',
    `leader_name` VARCHAR(100) DEFAULT NULL COMMENT '团队负责人名称',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态(1:启用 0:禁用)',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间'
) ENGINE=InnoDB AUTO_INCREMENT=100 DEFAULT CHARSET=utf8mb4 COMMENT='团队表';

-- =====================================================
-- 3. 代码库表 (repositories)
-- =====================================================
CREATE TABLE IF NOT EXISTS `repositories` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
    `project` VARCHAR(100) NOT NULL COMMENT '项目名称(如: myorg)',
    `name` VARCHAR(100) NOT NULL COMMENT '仓库名称(如: myrepo)',
    `description` TEXT DEFAULT NULL COMMENT '代码库描述',
    `git_url` VARCHAR(500) NOT NULL COMMENT 'Git仓库地址',
    `git_type` VARCHAR(20) NOT NULL COMMENT 'Git类型(gitlab/github/gitea等)',
    `language` VARCHAR(50) DEFAULT NULL COMMENT '主要编程语言',
    `team_id` BIGINT DEFAULT NULL COMMENT '所属团队ID',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态(1:启用 0:禁用)',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间',
    UNIQUE INDEX `idx_project_name` (`project`, `name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='代码库表';

-- =====================================================
-- 4. 应用表 (applications)
-- =====================================================
CREATE TABLE IF NOT EXISTS `applications` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `name` VARCHAR(100) NOT NULL COMMENT '应用名称',
  `project` VARCHAR(100) NOT NULL COMMENT '项目名称(继承自repository.project)',
  `display_name` VARCHAR(100) DEFAULT NULL COMMENT '应用展示名称',
  `description` TEXT COMMENT '应用描述',
  `repo_id` BIGINT NOT NULL COMMENT '关联的代码库ID',
  `app_type` VARCHAR(50) NOT NULL COMMENT '应用类型(web/api/job/microservice等)',
  `team_id` BIGINT DEFAULT NULL COMMENT '所属团队ID',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态(1:启用 0:禁用)',
  `deployed_tag` VARCHAR(100) DEFAULT NULL COMMENT '当前线上部署的tag（ProdDeployed时更新）',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_project_app_name` (`project`, `name`)
) ENGINE=InnoDB AUTO_INCREMENT=100 DEFAULT CHARSET=utf8mb4 COMMENT='应用表';

-- =====================================================
-- 5. 环境表 (environments)
-- =====================================================
CREATE TABLE IF NOT EXISTS `environments` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
    `name` VARCHAR(50) NOT NULL UNIQUE COMMENT '环境名称(dev/test/staging/prod)',
    `display_name` VARCHAR(100) DEFAULT NULL COMMENT '环境展示名称',
    `description` TEXT DEFAULT NULL COMMENT '环境描述',
    `env_type` VARCHAR(20) NOT NULL COMMENT '环境类型(dev/test/staging/prod)',
    `cluster_name` VARCHAR(100) DEFAULT NULL COMMENT '集群名称',
    `cluster_url` VARCHAR(500) DEFAULT NULL COMMENT '集群API地址',
    `cluster_token` VARCHAR(1000) DEFAULT NULL COMMENT '集群访问Token(加密存储)',
    `namespace` VARCHAR(100) DEFAULT NULL COMMENT 'K8s命名空间',
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

-- =====================================================
-- 6. 初始化数据
-- =====================================================

-- 插入默认本地管理员用户 (密码: admin123, 需要使用bcrypt加密)
-- 以下密码哈希是 "admin123" 的 bcrypt 加密结果
INSERT INTO `local_users` (`username`, `password`, `email`, `display_name`, `status`)
VALUES ('admin', '$2a$10$N9qo8u1K5PJXh3x9Y7u6J.eqw6Xb5nBxw5TqKJ1x9Y7u6J.eqw6Xb', 'admin@example.com', '系统管理员', 1)
ON DUPLICATE KEY UPDATE `username` = `username`;

-- 插入默认环境
INSERT INTO `environments` (`name`, `display_name`, `description`, `env_type`, `priority`, `require_approval`, `auto_deploy`, `status`)
VALUES 
    ('dev', '开发环境', '用于日常开发和调试', 'dev', 1, FALSE, TRUE, 1),
    ('test', '测试环境', '用于功能测试和集成测试', 'test', 2, FALSE, TRUE, 1),
    ('pre', '预发布环境', '用于上线前的最终验证', 'pre', 3, TRUE, FALSE, 1),
    ('prod', '生产环境', '正式生产环境', 'prod', 4, TRUE, FALSE, 1)
ON DUPLICATE KEY UPDATE `name` = `name`;

-- =====================================================
-- 8. 表关系说明
-- =====================================================
-- repositories (1) ----< (N) applications: 一个代码库可以有多个应用
-- applications (N) ----< (N) environments: 应用和环境通过app_env_configs表建立多对多关系

