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

