-- DevOps CD 工具 - Base Service 数据库表结构
-- 版本: v2.0
-- 创建日期: 2025-10-15
-- 数据库: MySQL 8.0+


-- =====================================================
-- 1. 部署任务表 (deployments)
-- =====================================================
CREATE TABLE `deployments` (
  `id`              bigint       NOT NULL AUTO_INCREMENT,
  `batch_id`        bigint       NOT NULL,
  `release_id`      bigint       NOT NULL,
  `app_id`          bigint       NOT NULL,
  `env`             varchar(20)  NOT NULL COMMENT 'pre 或 prod',
  `cluster`         varchar(63)  NOT NULL COMMENT '集群名称',
  `namespace`       varchar(63)  NOT NULL COMMENT 'K8s 命名空间',
  `deployment_name` varchar(63)  NOT NULL COMMENT '部署名称',
  `values_yaml`     text         NOT NULL COMMENT '部署配置',
  `image_url`       varchar(253)          DEFAULT NULL,
  `image_tag`       varchar(100) NOT NULL,
  `task_id`         varchar(100)          DEFAULT NULL COMMENT 'K8s 部署任务ID',
  `status`          varchar(20)  NOT NULL DEFAULT 'pending' COMMENT 'pending/running/success/failed',
  `retry_count`     int                   DEFAULT '0',
  `max_retry_count` int                   DEFAULT '3',
  `error_message`   text,
  `started_at`      timestamp    NULL     DEFAULT NULL,
  `finished_at`     timestamp    NULL     DEFAULT NULL,
  `created_at`      timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`      timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE = InnoDB
  AUTO_INCREMENT = 1000000
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_general_ci COMMENT ='部署任务表';


-- =====================================================
-- 2. Project环境配置表 (project_env_configs)
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