-- DevOps CD 工具 - Base Service 数据库表结构
-- 版本: v2.0
-- 创建日期: 2025-10-15
-- 数据库: MySQL 8.0+

-- =====================================================
-- 1. 代码库表 (repositories)
-- =====================================================
CREATE TABLE IF NOT EXISTS `repositories` (
  `id` BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT 'ID',
  `project` VARCHAR(63) NOT NULL COMMENT 'user/org 如(my_org)', -- todo: rename to namespace
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
  `name` VARCHAR(63) NOT NULL COMMENT '应用名称',   -- 期望 RFC1035
  `project_id` VARCHAR(63) NOT NULL COMMENT '项目名称(继承自repository.project)',    -- todo:
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



