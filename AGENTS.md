# AGENTS.md

## 项目身份

OpsCore 是一个智能运维平台项目，产品定位为智能运维中枢指挥平台，围绕业务连续性、事件闭环、SLA/SLO、AI 辅助分析和受控自动化构建统一运维工作台，符合平台工程、SRE、aiops 理念，目前一期功能覆盖资产管理、中间件与数据库管理、值班管理、任务跟踪、事件响应、权限管理，以及初步的 AI Copilot 入口。

## 当前技术栈

- 前端：Vue 3、Vite、原生 CSS。
- 前端 E2E：Playwright / `@playwright/test`。
- 后端：Go 1.24 REST API。
- 数据库：PostgreSQL 16。
- 后端数据库驱动：`github.com/jackc/pgx/v5`。
- 本地编排：`deploy/` 目录下的 Docker Compose。
- 前端容器：Nginx 承载 Vite 构建产物，并将 `/api` 反向代理到后端容器。

## 仓库结构

- `README.md`：当前启动方式、默认账号、Smoke 验收和业务说明的主要依据。
- `frontend/`：Vue 前端应用。
  - `src/App.vue`：当前单文件应用壳、业务视图、路由状态、表单和列表逻辑。
  - `src/api.js`：API 客户端、Token 存储和登录辅助方法。
  - `src/styles.css`：全局 UI 样式。
  - `package.json`：Vite 启动和构建脚本。
- `backend/`：Go REST API。
  - `cmd/server/main.go`：服务启动入口。
  - `internal/api/`：HTTP 路由、请求处理、校验和 API 测试。
  - `internal/auth/`：密码、JWT、RBAC 和认证测试。
  - `internal/config/`：环境变量配置。
  - `internal/crypto/`：敏感凭据加密。
  - `internal/domain/`：领域状态规则。
  - `internal/models/`：后端共享模型。
  - `internal/store/`：PostgreSQL 表结构和持久化逻辑。
- `deploy/`：Docker Compose 和环境变量模板。
- `scripts/`：本地运维脚本。
  - `smoke-api.sh`：通过后端容器执行端到端 API Smoke 流程。
  - `reset-admin-password.sh`：通过后端容器本地子命令重置管理员密码。
- `deliverables/`：评审文档、原型 HTML 

## 产品范围

一期当前覆盖：

- 首页仪表盘：资产、值班、任务、事件、权限待配置等指标。
- CMDB 资产台账：聚焦服务器和基础设施资产。
- 中间件与数据库实例管理：覆盖 MySQL、Redis、Kafka、PostgreSQL、达梦、Nginx、ElasticSearch、Nacos、RocketMQ、MinIO 等常见服务。
- 值班管理：支持 `daily` / `weekly` 两类规则。
- 任务跟踪：包含任务状态校验和状态流转。
- 事件管理：支持 P1/P2/P3/P4 等级和事件状态流转。
- 权限管理：包含 `super_admin` 和 `ops_engineer` 两类角色。
- 资产和中间件敏感凭据：使用 AES-GCM 加密存储。
- 全局 AI Copilot 入口：当前主要承担查询、摘要和建议类交互。
- AI Copilot 配置：一期前端提供模型厂商、端点、模型、上下文授权、审计边界、连接测试和后端密钥托管界面；真实模型调用代理后续再接入。

产品原则：首页看健康，异常看影响，告警看根因，处置看流程，复盘看改进，AI 贯穿查询、分析、建议和自动化。

## 平台工程与 AIOps 方向

- 当前一期规划整体符合平台工程和 AIOps 的初始形态：以统一入口承载资产、实例、值班、任务、事件和权限，形成面向运维团队的控制台。
- 平台工程侧应继续强化自助服务、标准化资产/服务目录、统一权限护栏、自动化流程编排、API 集成层和开发/运维协同体验。
- AIOps 侧应继续强化可观测数据接入、告警降噪、拓扑关联、影响面分析、根因辅助、预案推荐、复盘沉淀和受控自动化。
- AI Copilot 当前定位为查询、汇总和建议入口；在审计、权限和人工确认完备前，不应暗示其已自动完成生产处置。
- AI Copilot 连接测试必须由后端代理发起真实请求确认接口地址、模型和 API Key 可用；测试过程不得持久化 API Key，不得在响应、日志或页面提示中泄露密钥。
- AI Copilot 连接测试的失败信息也必须做密钥清洗，尤其是 Google Gemini 等把 key 放在 URL query 中的 provider，不能把底层 HTTP 错误原样回显给前端。
- AI Copilot 连接测试需要限制服务端可访问地址：本地模型 provider 可使用 `localhost`、`127.0.0.1`、私网地址或 `host.docker.internal`； hosted provider 默认不得使用 loopback、私网地址、链路本地地址或云元数据服务地址，避免把连接测试变成 SSRF 探测入口。
- AI Copilot Endpoint 安全策略必须覆盖配置保存、HTTP 重定向和 DNS 解析结果。Hosted provider 的重定向目标及域名解析 IP 仍需拒绝 loopback、私网、链路本地和元数据地址；实际连接应使用已校验的解析结果，避免 DNS 重绑定绕过。
- AI Copilot 配置保存必须通过后端完成，API Key 使用现有 AES-GCM 密钥能力加密后托管；GET/PUT 响应只能返回 `hasApiKey` 状态，不得返回明文 API Key。切换模型厂商或 endpoint 且未重新输入 Key 时，应清理旧托管 Key，避免跨厂商错配。

## 开发规则

- 优先沿用现有技术栈和仓库模式。除非任务明确要求，不要引入新的框架、路由库、状态管理库、UI 组件库或 CSS 体系。
- 修改范围应尽量贴近当前任务，不做无关重构。
- 后端必须执行关键业务规则校验，即使前端已经做过表单校验。
- 普通 API 响应不得暴露明文凭据。凭据查看必须保留权限校验和密码验证。
- 保留首次登录修改密码门禁。`mustChangePassword` 用户不得访问业务 API。
- 遵守 RBAC 边界：
  - `super_admin` 可管理用户、值班变更、凭据验证和高权限操作。
  - `ops_engineer` 可维护资产和中间件、处理任务、跟进事件，但不得意外获得用户管理或高权限凭据能力。
- 高风险自动化能力必须保持建议态或显式人工确认。没有权限模型、审计记录和确认流程前，不要加入真实生产命令执行。
- AI 输出必须表达为分析、建议或推荐。除非系统已通过受审计流程真实执行处置，否则不要暗示 AI 已经完成生产操作。
- 灰度或未来菜单入口应保持禁用，或明确展示为占位，避免误导用户认为模块已经可用。

## 后端约定

- 修改 API 行为、RBAC、校验、加密或状态流转时，应在对应的 `backend/internal/...` 包内新增或更新测试。
- HTTP 路由行为应与 `backend/internal/api/server.go` 保持一致。
- 枚举值必须在服务端校验。当前重要示例：
  - 资产类型、环境、状态、并网状态。
  - 中间件类型、环境、状态。
  - 任务状态：`待处理`、`处理中`、`待确认`、`已完成`、`已关闭`。
  - 事件等级：`P1`、`P2`、`P3`、`P4`。
  - 事件状态：`新建`、`处理中`、`已恢复`、`已关闭`。
  - 值班规则类型：`daily` 或 `weekly`。
- 保持安全默认值：
  - 空任务类型默认保存为 `任务`。
  - 空事件等级默认保存为 `P3`。
  - 空事件状态默认保存为 `新建`。
- 保持凭据加密稳定。`OPSCORE_CREDENTIAL_ENCRYPTION_KEY` 至少 32 字节，生产环境必须固定保存。

## 前端约定

- UI 文案默认使用中文，除非任务明确要求其他语言。
- 应用应呈现为真实可操作的运维控制台：信息密度较高、清晰、稳定、面向重复操作。
- 登录页必须保持简洁，不展示默认账号密码；登录背景主标题使用“智能运维中枢指挥平台”。
- 登录页视觉应围绕“智能运维中枢指挥平台”统一表达，不使用割裂的孤立图表；登录卡片使用暗色体系，与控制台风格保持一致。
- 登录页顶部文案应表达整个平台定位，避免把“一期功能清单”或重复标签堆在主视觉旁；中枢图负责承载平台能力关系。
- 登录页首屏文案优先保留一段清晰的平台定位说明，避免主标题下出现多段解释导致视觉重心分散。
- 登录页正式实现应与已确认的 `deliverables/login_command_center_review.html` 保持一致，包含单段平台定位文案、统一中枢能力图和暗色登录卡片。
- 避免营销式落地页、超大 Hero、装饰性图表或只适合展示的大屏效果。
- 除真实状态、错误、空状态或确认提示外，不要在界面里堆叠教学说明文字。
- 每个业务页面只保留顶层 `topbar` 的标题、面包屑和描述；内容区 `section-head`、Tab 子页和卡片头部不再重复页面级说明。内容区应优先展示操作、筛选、状态、列表和数据本身。
- 权限状态和禁用状态必须清晰。
- 列表、详情、编辑流程应保持高效：
  - 页面默认展示概览、筛选、列表和详情。
  - 新增/编辑表单只在用户触发操作后展开。
  - 行点击、详情面板、编辑、删除、导入导出和凭据操作应体现当前权限与业务语义。
- 主业务管理列表应保持统一表格体验：资产、中间件、任务、事件等可增长列表必须提供分页；表格操作列不能把 `td` 改成 `display:flex`，避免边框横线错位，按钮应在单元格内部用 inline-flex 或内联排列。
- 值班管理页面应以“概览 + 排班日历 + 交接班 + 当前值班接班”为主，不再把单条表格录入作为默认主界面。
- 值班管理虽然归属于“协同与事件响应”，但页面内必须拆清工作区：值班概览、排班日历、交接班、接班管理分别承载不同任务，避免把全部功能堆在一个视图里。
- 值班管理正式页面应参考 `/Users/mac/Desktop/work/开发规划/duty-management-deliverables/duty-management.html` 的业务交互逻辑和中文文案，但不能照搬其独立应用顶栏或内嵌侧栏；应融合到 OpsCore 后台整体设计中，用横向 Tab、页面工具条和统一弹窗承载概览、排班日历、排班配置、值班列表、交接班日志和升级策略。统计报表优先整合到概览中。
- 值班列表新增人员应关联系统用户，避免用浏览器原生 prompt 录入孤立姓名；排班配置和升级策略编辑必须使用完整业务弹窗，至少能维护团队、规则、人员、时间窗口、升级对象、升级延迟和通知渠道。
- 值班列表以表格信息为主，不额外弹出人员详情抽屉；操作列只保留必要操作并保持对齐。执行“安排值班”并确认后，必须同步更新人员状态、下次值班日期和团队筛选计数。
- 值班团队不能作为硬编码展示项长期固定；前端至少需要支持新增团队、修改团队名称和删除空团队，并同步影响值班人员、当前值班和排班模板中的团队归属。团队人数应由值班人员列表自动统计，不提供手工编辑。
- 删除值班团队必须有保护逻辑：已有人员归属的团队不能直接删除，应先迁移或调整人员团队，避免产生孤儿人员、排班模板或当前值班记录。
- 值班管理不应保留纯展示性质的“模拟告警”按钮，避免把测试演示操作误认为真实生产能力。
- AI Copilot 配置页属于“系统配置”，支持本地模型、OpenAI GPT、Anthropic Claude、Google Gemini 和 OpenAI 兼容接口；生产密钥不得落前端持久化，保存后输入框应清空并仅展示托管状态。
- AI Copilot 配置页的“测试连接”属于系统级密钥验证操作，只允许超级管理员使用；普通角色不得获得测试或提交真实生产密钥的能力。
- Docker Compose 栈中后端容器访问宿主机本地模型时，默认使用 `http://host.docker.internal:11434`；`deploy/docker-compose.yml` 需要为 backend 保留 `extra_hosts: ["host.docker.internal:host-gateway"]`，以兼容 Linux/远程 Docker 环境。如果后端在宿主机本地运行，可再改回 `http://localhost:11434`。
- 左侧菜单避免与系统配置重复，例如“消息与通知中心”不再作为独立一级菜单，通知能力统一归入“系统配置 / 通知渠道”。
- 生产和默认本地栈不应依赖前端演示数据。`VITE_ENABLE_DEMO_DATA` 默认必须为 `false`；只有做静态原型演示或无后端演示时才可显式设为 `true`。
- 如果 `App.vue` 继续增长，应优先拆分为聚焦组件，而不是继续把不相关逻辑追加到同一文件。

## 设计风格

- 暗色专业风、科技感
- 高信息密度 + 清晰视觉层级（卡片、颜色区分、chip）
- 零学习成本：所有操作都有明确按钮和反馈（Toast + 实时刷新）
- 响应式：PC 优先，也可在手机上基本浏览

## 环境变量

使用 `deploy/.env.example` 和 `frontend/.env.example` 作为模板。

重要变量：

- `POSTGRES_DB`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `POSTGRES_PORT`
- `BACKEND_PORT`
- `FRONTEND_PORT`
- `OPSCORE_DATABASE_URL`
- `OPSCORE_JWT_SECRET`
- `OPSCORE_CREDENTIAL_ENCRYPTION_KEY`
- `OPSCORE_INITIAL_ADMIN_PASSWORD`
- `OPSCORE_CORS_ORIGIN`
- `VITE_API_BASE`
- `VITE_ENABLE_DEMO_DATA`

生产部署必须替换默认密钥。不要提交真实 `.env` 文件或生产凭据。

## 运行手册

启动完整本地栈：

```bash
cd deploy
cp .env.example .env
docker compose up --build
```

访问地址：

- 前端：`http://localhost:5173`
- 后端健康检查：`http://localhost:8080/api/health`
- PostgreSQL：`localhost:5432`

后端本地开发：

```bash
cd backend
go mod tidy
go test ./...
go run ./cmd/server
```

前端本地开发：

```bash
cd frontend
npm install
npm run dev
```

前端构建：

```bash
cd frontend
npm run build
```

前端 E2E 检查：

```bash
cd frontend
npm run test:e2e
```

首次运行或浏览器缓存缺失时：

```bash
cd frontend
npx playwright install chromium
```

Docker Compose 启动后运行 Smoke 测试：

```bash
scripts/smoke-api.sh
```

如果管理员密码已经初始化，不再是默认密码：

```bash
ADMIN_USERNAME=admin ADMIN_PASSWORD='your-current-password' scripts/smoke-api.sh
```

通过后端容器重置本地管理员密码：

```bash
scripts/reset-admin-password.sh 'TempAdmin123!'
```

## 验证清单

完成代码变更前，运行最小相关检查：

- 仅后端变更：`cd backend && go test ./...`
- 为避免 Go 构建缓存污染 Git 状态，建议使用 `GOCACHE=/Users/mac/Desktop/work/OpsCore/.cache/go-build go test ./...`；`backend/.gocache/` 已从 Git 跟踪中移除并保持忽略。
- 仅前端变更：`cd frontend && npm run build`
- 前端页面或交互变更：除构建外，还必须核查对应页面功能点是否闭环，包括入口是否可见、按钮是否可触发、弹窗是否能取消/保存、状态是否刷新、禁用/权限态是否清晰、列表/详情/筛选/分页是否无明显残留或冲突。
- 前端页面或交互变更优先补充或执行 Playwright 检查：`cd frontend && npm run test:e2e`。当前 E2E 至少包含登录页 smoke 和一期页面逐页点击巡检，覆盖首页 KPI 跳转、侧边栏菜单、表单打开/取消、详情隐藏、分页、值班 Tab/弹窗、权限 Tab 和 AI Copilot 配置入口。如果浏览器依赖缺失，先运行 `npx playwright install chromium`。
- 涉及表格页面时，必须额外核查表头、行高、横线、操作列、空状态和分页条是否与资产台账等已确认页面保持一致。
- 全栈、认证、RBAC 或业务流程变更：
  - `cd backend && go test ./...`
  - `cd frontend && npm run build`
  - 启动 Docker Compose 后运行 `scripts/smoke-api.sh`
- 后续每次完成代码修改和本地测试后，都需要重新构建打包镜像并运行当前栈：
  - `cd deploy && docker compose up --build -d`
  - 前端、后端和数据库均应保持启动状态，再进行页面访问或 API 验证。

如果 Docker 不可用，应说明 Smoke 流程未运行，并列出已完成的后端/前端检查。

## 文档与交付物

- `README.md` 应始终与真实启动方式、验证方式、凭据规则和权限行为保持一致。
- 面向客户或评审的交付物放在 `deliverables/`。

## 建议下一步

- 将过大的 `frontend/src/App.vue` 拆分为布局、导航、资源列表、表单、凭据、任务、事件、值班和权限组件。
- 如果前端继续高频迭代，补充基础 lint 和最小测试能力。
- 后端路由稳定后，补充 API 文档或 OpenAPI 风格的接口说明。
- 在启用任何高风险操作或自动化动作前，先补充审计日志。
- 将 AI Copilot 从静态助手行为扩展为基于真实资产、事件、任务、值班和预案的权限感知查询与建议流程。
