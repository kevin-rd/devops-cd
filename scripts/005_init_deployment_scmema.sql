-- DevOps CD 工具 - Base Service 数据库表结构
-- 版本: v2.0
-- 创建日期: 2025-10-15
-- 数据库: MySQL 8.0+


-- =====================================================
-- 1. 部署任务表 (deployments)
-- =====================================================
CREATE TABLE `deployments` (
  `id`              bigint      NOT NULL AUTO_INCREMENT,
  `batch_id`        bigint      NOT NULL,
  `release_id`      bigint      NOT NULL,
  `app_id`          bigint      NOT NULL,
  `env`             varchar(20) NOT NULL COMMENT 'pre 或 prod',
  `cluster`         varchar(63) NOT NULL COMMENT '集群名称',
  `namespace`       varchar(63) NOT NULL COMMENT 'K8s 命名空间',
  `deployment_name` varchar(63) NOT NULL COMMENT '部署名称',
  `values`          json COMMENT '合并后的helm values',
  `task_id`         varchar(100)         DEFAULT NULL COMMENT 'K8s 部署任务ID',
  `status`          varchar(20) NOT NULL DEFAULT 'pending' COMMENT 'pending/running/success/failed',
  `retry_count`     int                  DEFAULT '0',
  `max_retry_count` int                  DEFAULT '3',
  `error_message`   text,
  `started_at`      timestamp   NULL     DEFAULT NULL,
  `finished_at`     timestamp   NULL     DEFAULT NULL,
  `created_at`      timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`      timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE = InnoDB
  AUTO_INCREMENT = 1000000
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_general_ci COMMENT ='部署任务表';


-- =====================================================
-- 2. Project环境配置表 (project_env_configs)
-- 用途: 存储Project级部署配置
-- 设计:
--   - 粒度: project+env
-- =====================================================
CREATE TABLE `project_env_configs` (
  `id`                       BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `project_id`               BIGINT UNSIGNED NOT NULL,
  `env`                      VARCHAR(32)     NOT NULL COMMENT '环境，如 pre/prod',
  `allow_clusters`           JSON            NOT NULL COMMENT '允许的集群列表',
  `default_clusters`         JSON            NOT NULL COMMENT '默认集群列表',
  `namespace`                VARCHAR(63)     NOT NULL DEFAULT '' COMMENT 'kubernetes命名空间',
  `deployment_name_template` VARCHAR(255)    NOT NULL DEFAULT '' COMMENT '部署名称模板',
  `chart_repo_url`           VARCHAR(255)    NOT NULL DEFAULT '' COMMENT 'Chart仓库URL',
  `values_repo_url`          VARCHAR(255)             DEFAULT NULL COMMENT 'Values仓库URL, 若为空则不配置',
  `values_path_template`     VARCHAR(255)             DEFAULT NULL COMMENT 'Values仓库路径, 可以使用go-template',
  `created_at`               DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`               DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_project_env` (`project_id`, `env`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;


-- =====================================================
-- 3. 应用环境配置表
-- 用途: 记录应用级的部署信息
-- 设计:
--   - 粒度: app+env+cluster
--   - 有 env='pre' 记录 -> 应用需要部署到 pre
--   - 无 env='pre' 记录 -> 应用跳过 pre,直接到 prod
-- =====================================================
CREATE TABLE IF NOT EXISTS `app_env_configs` (
  `id`                       BIGINT      NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `app_id`                   BIGINT      NOT NULL COMMENT '应用ID,关联 applications.id',
  `env`                      VARCHAR(20) NOT NULL COMMENT '环境名称: pre/prod/dev/test/uat 等',
  `cluster`                  VARCHAR(50) NOT NULL DEFAULT 'default' COMMENT '集群名称',

  -- 部署配置
  `deployment_name_override` VARCHAR(63)          DEFAULT NULL COMMENT '部署名称覆盖,为空则使用默认模板',
  `replicas`                 INT                  DEFAULT 1 COMMENT '副本数量',
  `config_data`              JSON                 DEFAULT NULL COMMENT '环境专属配置(JSON格式,用于存储扩展配置)',

  -- 系统字段
  `status`                   TINYINT     NOT NULL DEFAULT 1 COMMENT '状态(1:启用 0:禁用,用于临时禁用配置)',
  `created_at`               TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at`               TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at`               TIMESTAMP   NULL     DEFAULT NULL COMMENT '软删除时间',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_app_env_cluster` (`app_id`, `env`, `cluster`),
  INDEX `idx_app_env` (`app_id`, `env`),
  INDEX `idx_status` (`status`),
  INDEX `idx_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_app_env_configs_cluster` FOREIGN KEY (`cluster`) REFERENCES `clusters` (`name`) ON DELETE RESTRICT ON UPDATE CASCADE
  #CONSTRAINT `fk_app_env_configs_app` FOREIGN KEY (`app_id`) REFERENCES `applications` (`id`) ON DELETE CASCADE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='应用环境配置表';