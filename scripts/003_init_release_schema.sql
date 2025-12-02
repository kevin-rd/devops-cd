-- DevOps CD 工具 - Core Service 数据库表结构
-- 版本: v2.0
-- 创建日期: 2025-10-15
-- 数据库: MySQL 8.0+

-- =====================================================
-- 1. 构建记录表 (builds)
-- =====================================================
CREATE TABLE IF NOT EXISTS `builds` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `repo_id` BIGINT NOT NULL COMMENT '代码库ID',
  `app_id` BIGINT NOT NULL COMMENT '应用ID',
  `build_number` INT NOT NULL COMMENT 'CI构建编号（仓库维度）',
  `build_status` VARCHAR(20) NOT NULL COMMENT '构建状态(success/failure/error/killed)',
  `build_event` VARCHAR(20) NOT NULL COMMENT '触发事件(push/tag/pull_request/promote/rollback)',
  `build_link` VARCHAR(255) DEFAULT NULL COMMENT '构建链接',

  `commit_sha` VARCHAR(64) NOT NULL COMMENT 'Git commit SHA',
  `commit_ref` VARCHAR(255) DEFAULT NULL COMMENT 'Git ref (refs/tags/xxx or refs/heads/xxx)',
  `commit_branch` VARCHAR(100) DEFAULT NULL COMMENT '分支名',
  `commit_message` TEXT DEFAULT NULL COMMENT '提交信息',
  `commit_link` VARCHAR(255) DEFAULT NULL COMMENT '提交链接',
  `commit_author` VARCHAR(100) DEFAULT NULL COMMENT '提交者',

  `build_created` BIGINT NOT NULL COMMENT '创建时间戳(秒)',
  `build_started` BIGINT NOT NULL COMMENT '开始时间戳(秒)',
  `build_finished` BIGINT NOT NULL COMMENT '完成时间戳(秒)',
  `build_duration` INT DEFAULT NULL COMMENT '构建耗时(秒)',

  `image_tag` VARCHAR(100) NOT NULL COMMENT '镜像标签',
  `image_url` VARCHAR(500) DEFAULT NULL COMMENT '完整镜像地址',
  `app_build_success` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '该应用构建是否成功',

  `environment` VARCHAR(50) DEFAULT NULL COMMENT '目标环境(production/staging/testing等)',

  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `idx_repo_build` (`repo_id`, `build_number`),
  CONSTRAINT `fk_builds_repo` FOREIGN KEY (`repo_id`) REFERENCES `repositories`(`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_builds_app` FOREIGN KEY (`app_id`) REFERENCES `applications`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=1000000 DEFAULT CHARSET=utf8mb4 COMMENT='应用构建记录表';

-- =====================================================
-- 2. 发布批次表 (batches)
-- =====================================================
CREATE TABLE IF NOT EXISTS `release_batches` (
  `id` BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',

  -- 基本信息
  `batch_number` VARCHAR(200) NOT NULL UNIQUE COMMENT '批次编号/标题(用户填写,如:2025 1010 zkme项目日常更新)',
  `project_id` BIGINT NOT NULL COMMENT '关联的项目ID',
  `initiator` VARCHAR(50) DEFAULT NULL COMMENT '发起人',
  `release_notes` TEXT DEFAULT NULL COMMENT '批次发布说明',

  -- 审批信息（独立于部署流程）
  `approval_status` VARCHAR(20) NOT NULL DEFAULT 'pending' COMMENT '审批状态(pending/approved/rejected/skipped)',
  `approved_by` VARCHAR(50) DEFAULT NULL COMMENT '审批人',
  `approved_at` TIMESTAMP NULL DEFAULT NULL COMMENT '审批时间',
  `reject_reason` TEXT DEFAULT NULL COMMENT '拒绝原因',

  -- 部署流程状态
  -- 枚举: DRAFT/SEALED/PRE_DEPLOYING/PRE_DEPLOYED/PROD_DEPLOYING/PROD_DEPLOYED/COMPLETED/CANCELLED
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '部署流程状态(0:草稿 10:已封板 21:预发布中 22:预发布完成 31:生产部署中 32:生产部署完成 40:已完成 90:已取消)',

  -- 时间戳追踪
  `tagged_at` TIMESTAMP NULL DEFAULT NULL COMMENT '封板时间',
  `pre_deploy_started_at` TIMESTAMP NULL DEFAULT NULL COMMENT '预发布开始时间',
  `pre_deploy_finished_at` TIMESTAMP NULL DEFAULT NULL COMMENT '预发布完成时间',
  `prod_deploy_started_at` TIMESTAMP NULL DEFAULT NULL COMMENT '生产部署开始时间',
  `prod_deploy_finished_at` TIMESTAMP NULL DEFAULT NULL COMMENT '生产部署完成时间',

  -- 验收和取消
  `final_accepted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '最终验收时间',
  `final_accepted_by` VARCHAR(50) DEFAULT NULL COMMENT '验收人',
  `cancelled_at` TIMESTAMP NULL DEFAULT NULL COMMENT '取消时间',
  `cancelled_by` VARCHAR(50) DEFAULT NULL COMMENT '取消人',
  `cancel_reason` TEXT DEFAULT NULL COMMENT '取消原因',

  -- 系统字段
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

  UNIQUE INDEX `uk_batch_number` (`batch_number`, `project_id`),
  INDEX `idx_status` (`status`),
  INDEX `idx_approval_status` (`approval_status`),
  INDEX `idx_initiator` (`initiator`),
  INDEX `idx_created_at` (`created_at`),
  INDEX `idx_project_id` (`project_id`)
#   FOREIGN KEY (project_id) REFERENCES `projects`(`id`) ON DELETE RESTRICT
) ENGINE=InnoDB AUTO_INCREMENT=100 DEFAULT CHARSET=utf8mb4 COMMENT='发布批次表';

-- =====================================================
-- 3. 批次应用关联表 (release_apps) - 简约版
-- =====================================================
CREATE TABLE IF NOT EXISTS `release_apps` (
    `id` BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
    `batch_id` BIGINT NOT NULL COMMENT '批次ID',
    `app_id` BIGINT NOT NULL COMMENT '应用ID',

    -- 构建关联（可空：允许无构建应用加入批次，封板时校验）
    `build_id` BIGINT DEFAULT NULL COMMENT '关联的构建ID（封板时固定，不可再变更）',

    -- 版本信息
    `previous_deployed_tag` VARCHAR(100) DEFAULT NULL COMMENT '部署前的版本（封板时从 applications.deployed_tag 获取）',
    `target_tag` VARCHAR(100) DEFAULT NULL COMMENT '目标部署版本（封板时从 build.image_tag 获取并固定，部署期间代表期望版本，部署完成后代表已部署版本）',
    `latest_build_id` BIGINT DEFAULT NULL COMMENT '最新检测到的构建ID（新tag到达时更新）',

    -- 业务字段
    `release_notes` TEXT DEFAULT NULL COMMENT '应用级发布说明（可选）',
    `is_locked` BOOLEAN DEFAULT FALSE COMMENT '是否已锁定（封板后为true，不可再修改）',
    `skip_pre_env` TINYINT(1) NOT NULL DEFAULT 0,
    `status` TINYINT NOT NULL DEFAULT 0 COMMENT '应用级发布状态(0:待发布)',
    `failed_reason` TEXT DEFAULT NULL COMMENT '应用级发布失败原因（可选）',

    -- 系统字段
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    -- 索引
    UNIQUE KEY `uk_batch_app` (`batch_id`, `app_id`) COMMENT '同一批次不能重复添加同一应用',

    CONSTRAINT `fk_release_apps_build` FOREIGN KEY (`build_id`) REFERENCES `builds`(`id`) ON DELETE RESTRICT
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4 COMMENT='批次应用关联表';


-- =====================================================
-- 5. 表关系说明
-- =====================================================
-- applications (1) ----< (N) builds: 一个应用有多个构建记录
-- release_batches (1) ----< (N) release_apps: 一个发布批次有多个应用


