# 数据库表设计


## 设计原则

- 核心业务实体一律软删除，保证历史可追溯、误删可恢复、审计完整性
- 所有表使用`id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY`
- 统一使用 `created_at、updated_at、deleted_at` 时间戳
- 字符集统一 `utf8mb4_unicode_ci`

## 核心表设计

- project（项目/工作空间）
- team （团队/组织）
- user （用户）
- team_members （团队成员关系）

- repo （代码仓库）
- app （应用）
- builds （构建记录）
- release_batch （发布批次）
- release_app （批次中的应用）
- deployment （部署记录, 生成表）


## 软删除 vs 硬删除
表名,删除方式,说明
- project / team / user / repo / app 软删除,核心实体
- release_batch 只能取消, 不能删除
- release_app 创建阶段硬删除, 封板后使用软删除(todo)
- deployment 软删除,部署审计记录
- builds 软删除 + 定时归档
- team_members 硬删除