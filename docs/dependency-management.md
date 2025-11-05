# 应用依赖管理说明

## 概述

系统现支持两类依赖：

- **App Type 默认依赖**：通过 `configs/base.yaml` 中 `core.app_types.<app_type>.dependencies` 配置。相同批次中的应用会按照类型顺序自动阻塞，默认不可在批次中取消。
- **应用级依赖**：
  - **默认依赖**：在 `applications.default_depends_on` 字段维护，适用于长期依赖关系，由后台接口 `/api/v1/application/{id}/dependencies` 管理。
  - **临时依赖**：针对单个批次在 `release_apps.temp_depends_on` 字段维护，可在批次详情页通过复选框交互配置，也可通过新接口 `/api/v1/release_app/{id}/dependencies` 更新。

依赖解析统一由 `internal/core/dependency` 模块处理，部署状态机会在进入运行前校验依赖是否满足，若存在阻塞会将 Deployment 状态标记为 `waiting_dependencies` 并记录原因。

## 后端

- `internal/service/batch_service.go#UpdateReleaseDependencies` 新增临时依赖校验与更新逻辑：
  - 防止跨批次、循环依赖、自依赖。
  - 依赖图构建时仅考虑当前批次内的应用，默认依赖自动合并并去重。
- `internal/api/handler/release_app_handler.go` 暴露 `/api/v1/release_app/{id}/dependencies` PUT 接口，接收 `{ batch_id, operator, temp_depends_on }`。
- `dto.ReleaseAppResponse` 中追加 `default_depends_on`、`temp_depends_on` 字段，前端可直接消费。

## 前端

- `Batch` 详情页新增“依赖”列，区分展示默认依赖（紫色标签）与临时依赖（蓝色标签），并提供“设置依赖”按钮。
- 弹窗采用复选框形式：默认依赖自动勾选且只读，用户可额外选择临时依赖；保存后调用上述接口并刷新批次数据。

## 注意事项

- 默认依赖中若包含未加入当前批次的应用，会在弹窗中提示，部署时按已发布记录判断是否就绪。
- 当前未为依赖调整编写新的自动化测试；运行 `go test ./...` 时，请注意 `internal/core/batch` 下已有的格式化错误需先处理。

