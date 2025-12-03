-- DevOps CD 工具 - Base Service 数据库表结构
-- 版本: v2.0
-- 创建日期: 2025-10-15
-- 数据库: MySQL 8.0+

-- =====================================================
-- 1. 用户表 (users)
-- =====================================================
CREATE TABLE IF NOT EXISTS `users` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
    `username` VARCHAR(63) NOT NULL UNIQUE COMMENT '用户名',
    `password` VARCHAR(255) NOT NULL COMMENT '密码(bcrypt加密)',
    `email` VARCHAR(255) DEFAULT NULL COMMENT '邮箱',
    `display_name` VARCHAR(63) DEFAULT NULL COMMENT '显示名称',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态(1:启用 0:禁用)',
    `last_login_at` TIMESTAMP NULL DEFAULT NULL COMMENT '最后登录时间',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间',
    INDEX `idx_username` (`username`)
) ENGINE=InnoDB AUTO_INCREMENT=1000 DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

-- =====================================================
-- 2. 团队表 (teams)
-- =====================================================
CREATE TABLE IF NOT EXISTS `teams` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
    `name` VARCHAR(63) NOT NULL UNIQUE COMMENT '团队名称',
    `display_name` VARCHAR(100) DEFAULT NULL COMMENT '团队展示名称',
    `description` TEXT DEFAULT NULL COMMENT '团队描述',
    `leader_name` VARCHAR(63) DEFAULT NULL COMMENT '团队负责人名称',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态(1:启用 0:禁用)',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '软删除时间'
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COMMENT='团队表';


-- =====================================================
-- 3. 团队成员表 (team_members)
-- =====================================================
CREATE TABLE `team_members` (
  `id` bigint NOT NULL COMMENT '主键ID',
  `team_id` bigint NOT NULL COMMENT '团队ID',
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `roles` json NOT NULL DEFAULT (_utf8mb4'[]') COMMENT '该用户在当前团队拥有的角色列表',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '最后更新时间',
  KEY `idx_team_id` (`team_id`),
  KEY `idx_user_id` (`user_id`),
  CONSTRAINT `fk_team_members_team_id` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT,
  CONSTRAINT `fk_team_members_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='团队成员关系表（支持一人多团队、多角色）';


-- =====================================================
-- 4. 项目表 (projects)
-- =====================================================
CREATE TABLE IF NOT EXISTS `projects` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '项目名称（英文标识）',
  `description` text COMMENT '项目描述',
  `owner_name` varchar(100) DEFAULT NULL COMMENT '项目负责人',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name_deleted_at` (`name`, `deleted_at`),
  INDEX `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='项目表';

-- =====================================================
-- 5. 初始化数据
-- =====================================================

-- 插入默认本地管理员用户 (密码: admin123, 需要使用bcrypt加密)
-- 以下密码哈希是 "admin123" 的 bcrypt 加密结果
INSERT INTO `users` (`username`, `password`, `email`, `display_name`, `status`)
VALUES ('admin', '$2a$10$N9qo8u1K5PJXh3x9Y7u6J.eqw6Xb5nBxw5TqKJ1x9Y7u6J.eqw6Xb', 'admin@example.com', '系统管理员', 1)
ON DUPLICATE KEY UPDATE `username` = `username`;


-- =====================================================
-- 5. 表关系说明
-- =====================================================
-- repositories (1) ----< (N) applications: 一个代码库可以有多个应用
-- applications (N) ----< (N) environments: 应用和环境通过app_env_configs表建立多对多关系

