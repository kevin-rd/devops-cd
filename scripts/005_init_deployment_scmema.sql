-- DevOps CD 工具 - Base Service 数据库表结构
-- 版本: v2.0
-- 创建日期: 2025-10-15
-- 数据库: MySQL 8.0+

-- =====================================================
-- 1. Project环境配置表 (project_env_configs)
-- =====================================================
CREATE TABLE `project_env_configs` (
  `id`                       BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `project_id`               BIGINT UNSIGNED NOT NULL,
  `env`                      VARCHAR(32)     NOT NULL COMMENT '环境，如 pre/prod',
  `allow_clusters`           JSON            NOT NULL COMMENT '允许的集群列表',
  `default_clusters`         JSON            NOT NULL COMMENT '默认集群列表',
  `namespace`                VARCHAR(63)     NOT NULL DEFAULT '' COMMENT 'kubernetes命名空间',
  `deployment_name_template` VARCHAR(255)    NOT NULL DEFAULT '' COMMENT '部署名称模板',
  `created_at`               DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`               DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_project_env` (`project_id`, `env`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;