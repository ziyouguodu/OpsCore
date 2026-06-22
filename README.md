# OpsCore

OpsCore 是一个面向业务连续性的智能运维中枢指挥平台。项目目标是把资产、可观测、事件、任务、值班、权限、知识与 AI 辅助决策能力整合到一个统一控制台中，帮助运维团队围绕“健康、影响、根因、处置、复盘”形成闭环。

当前仓库是一期开源前工程骨架，已经包含可运行的 Vue 前端、Go REST API、PostgreSQL 数据库和 Docker Compose 本地部署配置。项目仍处于早期开发阶段，适合产品原型验证、内部试点和二次开发，不应直接作为生产系统使用。

## 项目状态

- 当前阶段：一期功能开发与原型落地。
- 运行方式：本地或远程 Docker Compose。
- 默认数据库：PostgreSQL 16。
- 默认认证方式：账号密码登录。
- 默认角色：`super_admin`、`ops_engineer`。
- 许可证状态：已提供 Apache License 2.0，详见 `LICENSE` 文件。

## 产品定位

OpsCore 不是单一 CMDB、工单系统或监控页面，而是一个以事件闭环和业务连续性为主线的智能运维控制台。

核心原则：

- 首页看健康。
- 异常看影响。
- 告警看根因。
- 处置看流程。
- 复盘看改进。
- AI 贯穿查询、分析、建议和自动化。

AI Copilot 当前定位为查询、汇总和建议入口。在审计、权限和人工确认流程完善前，不应暗示 AI 已经自动完成生产处置。

## 一期功能

已纳入一期的主要模块：

- 首页仪表盘：资产、值班、任务、事件和权限配置状态。
- 资产管理：
  - 资产台账（CMDB）：服务器与基础资产信息、部署信息、网络区域、负责人、状态和受控登录信息。
  - 中间件与数据库：MySQL、Redis、Kafka、PostgreSQL、达梦、Nginx、ElasticSearch、Nacos、RocketMQ、MinIO 等实例管理。
- 协同与事件响应：
  - 值班管理：当前值班、排班日历、交接班、接班管理和排班记录。
  - 任务跟踪：任务创建、状态流转和处理跟进。
  - 事件管理：P1-P4 事件等级、事件状态流转和事件跟进。
- 身份与权限：
  - 超级管理员。
  - 运维工程师。
  - 菜单与资源权限边界。
  - 凭据查看二次校验密码配置。
- AI Copilot：
  - 全局悬浮入口。
  - AI Copilot 配置页，支持本地模型、OpenAI GPT、Anthropic Claude、Google Gemini 和 OpenAI 兼容接口的前端配置形态。

灰度或后续模块仅作为菜单占位，不表示已经具备完整业务能力。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 前端 | Vue 3、Vite、原生 CSS |
| 后端 | Go 1.24、标准库 `net/http` REST API |
| 数据库 | PostgreSQL 16 |
| 数据库驱动 | `github.com/jackc/pgx/v5` |
| 凭据加密 | AES-GCM，密钥来自 `OPSCORE_CREDENTIAL_ENCRYPTION_KEY` |
| 本地编排 | Docker Compose |
| 前端容器 | Nginx 承载 Vite 构建产物，并将 `/api` 反向代理到后端 |

## 仓库结构

```text
.
├── backend/                  # Go REST API
│   ├── cmd/server/           # 服务启动入口
│   └── internal/             # API、认证、配置、加密、领域规则、存储
├── frontend/                 # Vue 3 + Vite 前端
│   ├── src/App.vue           # 当前主要应用视图和交互逻辑
│   ├── src/api.js            # API 客户端和 Token 辅助方法
│   └── src/styles.css        # 全局样式
├── deploy/                   # Docker Compose 和环境变量模板
├── scripts/                  # Smoke、管理员密码重置等脚本
├── deliverables/             # 原型和评审交付物
├── AGENTS.md                 # Codex/协作规则
├── WORKLOG.md                # 项目工作记录
└── logo.png                  # 当前系统 Logo
```

## 快速开始

### 前置条件

- Docker 和 Docker Compose。
- Node.js 与 npm。仅前端本地开发需要。
- Go 1.24。仅后端本地开发需要。

### 使用 Docker Compose 启动

```bash
cd deploy
cp .env.example .env
docker compose up --build -d
```

默认访问地址：

- 前端控制台：`http://localhost:5173`
- 后端健康检查：`http://localhost:8080/api/health`
- PostgreSQL：`localhost:5432`

查看容器状态：

```bash
cd deploy
docker compose ps
```

停止服务：

```bash
cd deploy
docker compose down
```

如果需要清空本地数据库数据：

```bash
cd deploy
docker compose down -v
```

注意：`docker compose down -v` 会删除 PostgreSQL 数据卷，请谨慎使用。

## 默认账号

系统首次启动会自动创建超级管理员账号：

| 项 | 值 |
| --- | --- |
| 用户名 | `admin` |
| 初始密码 | `ChangeMe123!` |

首次登录后系统会强制修改初始化密码。未完成密码初始化的用户不能访问业务 API。

生产或共享环境请在 `deploy/.env` 中修改：

- `OPSCORE_INITIAL_ADMIN_PASSWORD`
- `OPSCORE_JWT_SECRET`
- `OPSCORE_CREDENTIAL_ENCRYPTION_KEY`
- `POSTGRES_PASSWORD`

## 环境变量

Docker Compose 使用 `deploy/.env`。可从模板复制：

```bash
cp deploy/.env.example deploy/.env
```

关键变量：

| 变量 | 说明 |
| --- | --- |
| `POSTGRES_DB` | PostgreSQL 数据库名 |
| `POSTGRES_USER` | PostgreSQL 用户名 |
| `POSTGRES_PASSWORD` | PostgreSQL 密码 |
| `POSTGRES_PORT` | 本机 PostgreSQL 暴露端口 |
| `BACKEND_PORT` | 本机后端 API 端口 |
| `FRONTEND_PORT` | 本机前端端口 |
| `OPSCORE_JWT_SECRET` | JWT 签名密钥 |
| `OPSCORE_CREDENTIAL_ENCRYPTION_KEY` | 敏感凭据 AES-GCM 加密密钥，至少 32 字节 |
| `OPSCORE_INITIAL_ADMIN_PASSWORD` | 初始管理员密码 |
| `OPSCORE_CORS_ORIGIN` | 后端允许的前端来源 |
| `VITE_API_BASE` | 前端 API 基础路径，Compose 默认 `/api` |

远程部署时，如果前端和后端拆分域名，需要同步调整 `VITE_API_BASE` 和 `OPSCORE_CORS_ORIGIN`。

## 本地开发

### 后端

```bash
cd backend
go mod tidy
go test ./...
go run ./cmd/server
```

后端默认读取以下本地配置：

- `OPSCORE_LISTEN_ADDR`，默认 `:8080`
- `OPSCORE_DATABASE_URL`，默认 `postgres://opscore:opscore@localhost:5432/opscore?sslmode=disable`
- `OPSCORE_JWT_SECRET`
- `OPSCORE_CREDENTIAL_ENCRYPTION_KEY`
- `OPSCORE_INITIAL_ADMIN_PASSWORD`
- `OPSCORE_CORS_ORIGIN`

### 前端

```bash
cd frontend
npm install
npm run dev
```

前端本地开发默认配置在 `frontend/.env.example` 中：

```text
VITE_API_BASE=http://localhost:8080/api
```

如果后端地址不同，可创建 `frontend/.env.local` 并覆盖 `VITE_API_BASE`。

### 前端构建

```bash
cd frontend
npm run build
```

### 每次代码修改后的容器验证

本项目当前以后端、前端、数据库均在 Docker Compose 中运行为默认验证方式。完成代码修改和本地测试后，请重新构建镜像并运行当前栈：

```bash
cd deploy
docker compose up --build -d
```

随后验证：

```bash
curl -I http://localhost:5173/
curl http://localhost:8080/api/health
```

## Smoke 验收

Docker Compose 启动后，可在项目根目录运行：

```bash
scripts/smoke-api.sh
```

Smoke 脚本会在后端容器内验证一期关键 API 流程：

- 管理员登录。
- 首次密码修改门禁。
- Dashboard 访问。
- 创建运维工程师用户。
- 运维工程师权限受限校验。
- 资产创建。
- 敏感凭据写入和受限查看。
- 任务默认值和状态流转。
- 事件等级、默认值和状态流转。
- 值班规则校验。

如果管理员密码已经初始化，不再是默认密码：

```bash
ADMIN_USERNAME=admin ADMIN_PASSWORD='your-current-password' scripts/smoke-api.sh
```

如果忘记本地管理员密码，可在 Docker Compose 启动后执行：

```bash
scripts/reset-admin-password.sh 'TempAdmin123!'
```

该脚本只在后端容器内执行本地子命令，不开放网络重置接口。重置后 `admin` 会重新进入首次登录修改密码状态。

## 权限与安全边界

### 角色

| 角色 | 权限说明 |
| --- | --- |
| `super_admin` | 用户管理、权限配置、值班写入、资产/实例凭据配置和凭据查看等全部管理能力 |
| `ops_engineer` | 维护资产和中间件实例、处理任务、跟进事件；默认不能管理用户、修改值班排班或查看敏感凭据 |

### 敏感凭据

- 资产登录信息存储在 `asset_credentials`。
- 中间件与数据库登录信息存储在 `middleware_credentials`。
- 密码/密钥使用 `OPSCORE_CREDENTIAL_ENCRYPTION_KEY` 加密后落库。
- 普通列表和保存响应不会返回明文凭据。
- 凭据查看需要权限校验和二次密码校验。
- 生产环境必须固定保存 `OPSCORE_CREDENTIAL_ENCRYPTION_KEY`。密钥变更后，旧密文将无法解密。

### 生产安全提醒

正式部署前至少需要完成：

- 更换所有默认密码和默认密钥。
- 补充 HTTPS、访问控制和反向代理安全配置。
- 配置数据库备份和恢复流程。
- 增加审计日志，尤其是凭据查看、用户管理、事件变更和自动化操作。
- 为 AI Copilot 增加后端密钥托管、调用代理、权限感知上下文和人工确认机制。

## API 概览

后端 API 统一以 `/api` 为前缀。

| 模块 | 接口示例 |
| --- | --- |
| 健康检查 | `GET /api/health` |
| 认证 | `POST /api/auth/login`、`GET /api/auth/me`、`POST /api/auth/password` |
| Dashboard | `GET /api/dashboard` |
| 用户与角色 | `GET/POST /api/users`、`PUT/DELETE /api/users/{id}` |
| 凭据校验配置 | `GET/PUT /api/security/credential-verification` |
| 资产台账 | `GET/POST /api/assets`、`PUT/DELETE /api/assets/{id}` |
| 资产凭据 | `GET/PUT /api/assets/{id}/credential`、`POST /api/assets/{id}/credential/reveal` |
| 中间件与数据库 | `GET/POST /api/middleware`、`PUT/DELETE /api/middleware/{id}` |
| 实例凭据 | `GET/PUT /api/middleware/{id}/credential`、`POST /api/middleware/{id}/credential/reveal` |
| 值班管理 | `GET/POST /api/oncall`、`PUT/DELETE /api/oncall/{id}` |
| 任务跟踪 | `GET/POST /api/tasks`、`PUT/DELETE /api/tasks/{id}`、`PATCH /api/tasks/{id}` |
| 事件管理 | `GET/POST /api/incidents`、`PUT/DELETE /api/incidents/{id}`、`PATCH /api/incidents/{id}` |

## 数据与枚举约束

服务端会校验关键枚举和必填字段，不能只依赖前端表单。

- 资产环境：`生产`、`仿真`、`研发`。
- 资产类型：`物理机`、`虚拟机`。
- 中间件类型：`MySQL`、`Redis`、`Kafka`、`PostgreSQL`、`达梦`、`Nginx`、`ElasticSearch`、`Nacos`、`RocketMQ`、`MinIO`。
- 任务状态：`待处理`、`处理中`、`待确认`、`已完成`、`已关闭`。
- 事件等级：`P1`、`P2`、`P3`、`P4`。
- 事件状态：`新建`、`处理中`、`已恢复`、`已关闭`。
- 值班规则：`daily`、`weekly`。

## 开源贡献说明

欢迎基于当前项目进行试用、问题反馈和二次开发。正式开放协作前，建议补充以下文件：

- `LICENSE`：明确开源许可证。
- `CONTRIBUTING.md`：贡献流程、分支命名、提交规范和代码评审要求。
- `CODE_OF_CONDUCT.md`：社区行为准则。
- `SECURITY.md`：安全漏洞报告方式和响应流程。

在这些文件补齐前，贡献方式建议先以 Issue 或内部评审记录为主。

提交变更前建议至少完成：

```bash
cd backend && go test ./...
cd frontend && npm run build
cd deploy && docker compose up --build -d
```

涉及认证、RBAC、凭据、任务/事件状态流转或核心业务 API 的变更，应额外运行：

```bash
scripts/smoke-api.sh
```

## 路线图

短期优先事项：

- 将过大的 `frontend/src/App.vue` 拆分为布局、导航、资源列表、表单、凭据、任务、事件、值班和权限组件。
- 补充前端 lint 和基础测试能力。
- 补充 API 文档或 OpenAPI 风格接口说明。
- 增加审计日志。
- 扩展 AI Copilot 为基于真实资产、事件、任务、值班和预案的权限感知查询与建议流程。

中长期方向：

- 可观测数据接入和告警降噪。
- 拓扑关联、影响面分析和根因辅助。
- 标准化服务目录与自助运维能力。
- 预案推荐、复盘沉淀和受控自动化。
- SLO/SLA 管理、FinOps、云原生与 CI/CD 体系扩展。

## 许可证

当前仓库已提供 Apache License 2.0，详见根目录 `LICENSE` 文件。
