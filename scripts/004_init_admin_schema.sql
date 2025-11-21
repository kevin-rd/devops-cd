-- DevOps CD 工具 - 仓库源配置表
-- 版本: v4.0
-- 创建日期: 2025-11-20
-- 数据库: MySQL 8.0+

CREATE TABLE IF NOT EXISTS `repo_sync_sources` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
    `platform` VARCHAR(20) NOT NULL COMMENT '平台类型 gitea/gitlab/github',
    `base_url` VARCHAR(255) NOT NULL COMMENT 'Git 平台基础 URL',
    `namespace` VARCHAR(255) NOT NULL COMMENT '命名空间/组织/用户',
    `auth_token_enc` TEXT NOT NULL COMMENT '加密后的访问令牌',
    `enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    `default_project_id` BIGINT UNSIGNED NULL COMMENT '默认项目ID（扫描时自动设置）',
    `default_team_id` BIGINT UNSIGNED NULL COMMENT '默认团队ID（扫描时自动设置）',
    `last_synced_at` DATETIME NULL COMMENT '最近同步时间',
    `last_status` VARCHAR(20) NULL COMMENT '最近同步状态 success/failed',
    `last_message` TEXT NULL COMMENT '最近同步结果信息',
    `ext` JSON NULL COMMENT '扩展参数',
    `created_by` VARCHAR(50) NULL COMMENT '创建人',
    `updated_by` VARCHAR(50) NULL COMMENT '更新人',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` DATETIME NULL COMMENT '删除时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_repo_source_base_namespace` (`base_url`, `namespace`),
    KEY `idx_repo_source_platform` (`platform`),
    KEY `idx_repo_source_enabled` (`enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='仓库同步源配置';


