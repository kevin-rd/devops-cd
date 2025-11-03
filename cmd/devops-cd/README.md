# Base Service

DevOps CD 工具的基础服务,提供代码库、应用、环境管理和认证功能。

## 功能特性

- ✅ LDAP + 本地用户双认证
- ✅ JWT Token 认证
- ✅ 代码库管理 CRUD
- ✅ 应用管理 CRUD
- ✅ 环境管理 CRUD
- ✅ 团队管理 CRUD
- ✅ 应用环境配置管理
- ✅ 灵活的配置文件加载(支持命令行参数和环境变量)

## 技术栈

- Go 1.21+
- Gin Web Framework
- GORM ORM
- MySQL 8.0+
- JWT认证
- LDAP集成

## 项目结构

```
devops-cd/
├── cmd/
│   └── base/
│       ├── main.go          # Base服务入口
│       └── README.md
├── internal/
│   ├── api/
│   │   ├── handler/         # HTTP处理器
│   │   ├── middleware/      # 中间件
│   │   └── router/          # 路由配置
│   ├── service/             # 业务逻辑层
│   ├── repository/          # 数据访问层
│   ├── model/               # 数据模型
│   ├── dto/                 # 数据传输对象
│   └── pkg/                 # 内部工具包
│       ├── config/          # 配置管理
│       ├── database/        # 数据库
│       ├── logger/          # 日志
│       ├── jwt/             # JWT工具
│       └── crypto/          # 加密工具
├── pkg/                     # 公共工具包
│   ├── constants/
│   ├── errors/
│   └── utils/
├── configs/
│   └── base.yaml            # Base服务配置文件
├── migrations/              # 数据库迁移脚本
├── go.mod
└── README.md
```

## 快速开始

### 前置要求

- Go 1.21+
- MySQL 8.0+
- (可选) LDAP服务器

### 1. 初始化数据库

```bash
# 创建数据库
mysql -u root -p -e "CREATE DATABASE devops_cd CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 执行迁移脚本
mysql -u root -p devops_cd < migrations/001_init_base_schema.sql
```

### 2. 配置

修改 `configs/base.yaml` 中的配置:
- 数据库连接信息
- JWT Secret (生产环境必须修改)
- AES Key (生产环境必须修改)
- LDAP配置 (如果使用LDAP认证)

### 3. 安装依赖

```bash
go mod download
```

### 4. 运行服务

#### 方式1: 使用默认配置文件 (configs/base.yaml)

```bash
cd cmd/base
go run main.go
```

#### 方式2: 通过命令行参数指定配置文件

```bash
cd cmd/base
go run main.go -config=../../configs/base.yaml

# 或者使用相对路径
go run main.go -config=/path/to/your/config.yaml
```

#### 方式3: 通过环境变量指定配置文件

```bash
export CONFIG_FILE=configs/base.yaml
cd cmd/base
go run main.go
```

#### 方式4: 构建后运行

```bash
# 构建
go build -o build/base ./cmd/base

# 运行(使用默认配置)
./build/base

# 运行(指定配置)
./build/base -config=configs/base.yaml

# 运行(使用环境变量)
CONFIG_FILE=configs/base.yaml ./build/base
```

### 5. 查看版本信息

```bash
cd cmd/base
go run main.go -version
```

服务将在 http://localhost:8080 启动

## API 文档

### 认证 API

#### 登录
```
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123",
  "auth_type": "local"  // ldap 或 local
}
```

响应:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 7200,
    "user": {
      "username": "admin",
      "email": "admin@example.com",
      "display_name": "系统管理员",
      "auth_type": "local"
    }
  }
}
```

#### 刷新Token
```
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

#### 获取当前用户信息
```
GET /api/v1/auth/me
Authorization: Bearer {access_token}
```

## Docker 部署

### 构建镜像

```bash
make docker-build
```

### 运行容器

```bash
docker run -d \
  --name base-service \
  -p 8080:8080 \
  -e DATABASE_HOST=mysql \
  -e DATABASE_PASSWORD=password \
  devops-cd-base:latest
```

## 开发

### 热重载

安装 air:
```bash
go install github.com/cosmtrek/air@latest
```

运行:
```bash
make dev
```

### 代码格式化

```bash
make fmt
```

### 代码检查

```bash
make lint
```

## 环境变量

可以通过环境变量覆盖配置文件:

- `DATABASE_HOST`: 数据库主机
- `DATABASE_PORT`: 数据库端口
- `DATABASE_NAME`: 数据库名称
- `DATABASE_USER`: 数据库用户名
- `DATABASE_PASSWORD`: 数据库密码
- `LDAP_HOST`: LDAP主机
- `LDAP_PORT`: LDAP端口

## 注意事项

1. **生产环境配置**: 修改 `config.yaml` 中的 JWT Secret 和 AES Key
2. **LDAP配置**: 根据实际LDAP服务器配置修改相关参数
3. **日志**: 生产环境建议使用文件日志输出
4. **数据库**: 确保数据库已正确初始化表结构

## TODO

- [ ] Repository CRUD
- [ ] Application CRUD  
- [ ] Environment CRUD
- [ ] Team CRUD
- [ ] API 文档(Swagger)
- [ ] 单元测试
- [ ] 集成测试

## 许可证

待定

