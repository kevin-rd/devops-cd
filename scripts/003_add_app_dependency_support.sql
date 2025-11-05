-- DevOps CD 工具 - 应用依赖扩展脚本
-- 版本: v3.0
-- 创建日期: 2025-11-05
-- 数据库: MySQL 8.0+

-- =====================================================
-- 1. applications 表新增默认依赖字段
-- =====================================================
ALTER TABLE `applications`
    ADD COLUMN `default_depends_on` JSON NULL COMMENT '默认依赖的应用ID列表(JSON)';

-- =====================================================
-- 2. release_apps 表新增临时依赖字段
-- =====================================================
ALTER TABLE `release_apps`
    ADD COLUMN `temp_depends_on` JSON NULL COMMENT '批次临时依赖的应用ID列表(JSON)';

