# DevOps CD Platform - Frontend

基于 React + TypeScript + Ant Design 的持续部署平台前端。

## 技术栈

- **框架**: React 18 + TypeScript
- **构建工具**: Vite
- **UI 库**: Ant Design 5.x
- **路由**: React Router v6
- **状态管理**: Zustand
- **数据请求**: TanStack Query (React Query)
- **国际化**: i18next
- **HTTP 客户端**: Axios

## 快速开始

### 安装依赖

```bash
npm install
# 或
pnpm install
```

### 开发运行

```bash
npm run dev
```

访问 http://localhost:5173

### 构建

```bash
npm run build
```

构建产物在 `dist/` 目录。

### 预览构建产物

```bash
npm run preview
```

## 项目结构

```
src/
├── assets/              # 静态资源
├── components/          # 通用组件
├── pages/               # 页面组件
├── services/            # API 服务
├── stores/              # 状态管理
├── types/               # TypeScript 类型
├── utils/               # 工具函数
├── locales/             # 国际化语言文件
├── routes/              # 路由配置
├── App.tsx              # 根组件
└── main.tsx             # 入口文件
```

## 环境变量

- `VITE_API_BASE_URL`: 后端 API 地址

## 开发规范

- 使用 TypeScript 严格模式
- 遵循 ESLint 规则
- 组件使用函数式组件 + Hooks
- API 请求使用 React Query
- 全局状态使用 Zustand

## 设计风格

- 扁平化设计
- 卡片式布局
- 圆角元素
- Mac 风格

## 前后端接口
- Repo页面
  - Repo视图
    - Main: /api/v1/repositories?page=1&page_size=20&with_applications=true 
  - App视图
    - Main: /api/v1/applications?page=1&page_size=20
  - Options: /api/v1/projects & /api/v1/teams
- Batch页面
  - Batch列表: /api/v1/batches?page=1&page_size=20&start_time=2025-10-26T16%3A00%3A00.000Z&end_time=2025-11-26T15%3A59%3A59.999Z
  - Batch详情: /api/v1/batch?id=31&app_page=1&app_page_size=20
    - 单个Release详情: /api/v1/release_app?id=84
  - AppSelection组件: /api/v1/application_builds?page=1&page_size=20&project_id=1
- Project管理页
  - Main: /api/v1/projects?page=1&page_size=10&keyword=&with_teams=true
- Repo源管理页
  - Main: /api/v1/repo-sources?page=1&page_size=10
- 通用:
  - 应用类型: /api/v1/application/types