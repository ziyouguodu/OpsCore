# WORKLOG.md

## 2026-06-22 14:52 CST - 任务与事件列表分页和表格横线修正

- 修复全局表格操作列样式：`td.row-actions` 不再使用 `display:flex`，避免操作列破坏表格单元格导致横线错位。
- 操作列按钮改为在标准表格单元格内以 inline-flex 排列，资产、中间件、值班、任务、事件和用户列表统一受益。
- 任务跟踪页面新增分页状态、分页计算和分页控件，列表数据改为 `pagedTasks` 渲染，并补充空状态。
- 事件管理页面新增分页状态、分页计算和分页控件，列表数据改为 `pagedIncidents` 渲染，并补充空状态。
- 任务/事件分页条调整为表格容器外、列表列内，结构参照资产台账页面，避免分页条落入表格边框或成为详情布局的独立列。
- `AGENTS.md` 新增表格页面检查规则：主业务管理列表需提供分页，操作列不能改变 `td` 的 table-cell 行为，交付前需核查表头、行高、横线、操作列、空状态和分页条。

## 2026-06-22 09:08 CST - 页面标题描述层级收口

- 恢复值班管理页面的顶层 `topbar`，使其与资产台账、中间件、任务、事件、权限和 AI 配置等页面保持一致。
- 将值班管理顶层描述改为“面向事件响应连续性，统一管理当前值班、排班日历、交接日志与升级策略。”
- 移除各业务页面内容区 `section-head` 中重复的页面级描述，仅保留操作按钮、状态标签和业务内容。
- 移除值班管理内部重复的“值班管理”头部与各子 Tab/卡片头部说明，保留子区块标题、状态、数据和操作。
- `AGENTS.md` 新增 UI 约定：页面级标题和描述只保留在顶层 `topbar`，内容区不再重复解释。

## 2026-06-18 16:03 CST - 团队配置删除与页面核查约定

- 值班管理的团队配置弹窗新增“删除”操作，空团队可直接移除。
- 已有关联人员的团队会禁用删除，并提示先调整人员团队后再删除，避免人员、当前值班或排班模板出现无归属数据。
- 团队人数保持只读自动统计，来源是值班人员列表中对应团队的人数，不提供手工录入。
- 团队配置弹窗补充响应式布局，窄屏下团队名称、人数和删除按钮改为单列展示。
- `AGENTS.md` 新增前端页面或交互变更后的功能点核查要求：检查入口、按钮、弹窗、保存/取消、状态刷新、权限态和残留冲突。

## 2026-06-18 15:42 CST - 值班管理交互收口

- 值班管理页移除外层重复标题，只保留页面内部的集成标题与描述，减少同屏标题冲突。
- 值班概览移除纯展示用的“模拟告警”按钮，避免演示能力干扰真实业务操作。
- 值班列表去掉人员详情按钮和右侧详情抽屉，列表行不再默认弹出详情；操作列仅保留“安排值班”，并修正操作列对齐与链接样式。
- “安排值班”确认后会同步更新值班人员状态为“值班中”、下次值班日期、本月值班次数，并刷新团队筛选计数。
- 新增“团队配置”统一弹窗，支持新增值班团队和修改团队名称；保存后同步更新值班人员、当前值班和排班模板中的团队归属。
- 同步更新 `AGENTS.md`，补充值班管理交互、团队配置和模拟告警移除的后续开发约束。

## 2026-06-15 - 项目上下文整理

本工作记录根据当前 OpsCore 仓库文件、历史  Codex 任务包以及工作区内已有交付物整理生成。

## 当前仓库状态

- OpsCore 当前是一个完整的一期全栈运维平台骨架，技术栈包含 Vue 3、Vite、Go REST API、PostgreSQL 和 Docker Compose。
- 根仓库当前使用的产品名称是 OpsCore。
- 根目录 `README.md` 是当前启动方式、账号、Smoke 验收和功能范围的主要依据。
- 已生成根目录 `AGENTS.md`，用于保存后续 Codex 会话需要读取的工程和产品上下文。

## 已观察到的一期实现范围

- 首页仪表盘：资产数量、今日值班、活跃任务、活跃事件、资产类型统计和事件等级统计。
- CMDB 资产台账：支持物理机、虚拟机等基础设施记录。
- 中间件与数据库台账：覆盖 MySQL、Redis、Kafka、PostgreSQL、达梦、Nginx、ElasticSearch、Nacos、RocketMQ、MinIO 等实例。
- 值班管理：支持 daily 和 weekly 两类规则。
- 任务跟踪：包含状态校验和状态流转 API。
- 事件管理：包含 P1/P2/P3/P4 等级和状态校验。
- 用户与角色管理：包含 `super_admin` 和 `ops_engineer`。
- 资产和中间件凭据：使用加密方式存储敏感信息。
- 前端全局 AI Copilot 入口：当前围绕资产查询、活跃事件、今日值班、待处理任务和处置建议展开。

## 重要安全与权限决策

- 默认账号仍为 `admin` / `ChangeMe123!`，仅用于本地初始启动。
- 首次登录必须强制完成密码初始化，之后才能访问业务 API。
- 新密码至少 8 位，前端和后端都会校验。
- 生产环境必须替换默认 JWT 密钥、数据库口令和凭据加密密钥。
- `OPSCORE_CREDENTIAL_ENCRYPTION_KEY` 至少 32 字节，且生产环境必须固定保存；变更后旧密文将无法解密。
- 资产和中间件凭据保存接口不得在响应中暴露明文密钥。
- 凭据查看需要高权限和密码验证。
- 运维工程师可以维护资产和中间件、处理任务、跟进事件，但不得意外获得用户管理或高权限凭据能力。

## 当前本地命令

启动 Docker Compose 栈：

```bash
cd deploy
cp .env.example .env
docker compose up --build
```

运行后端测试：

```bash
cd backend
go test ./...
```

运行前端开发服务：

```bash
cd frontend
npm install
npm run dev
```

构建前端：

```bash
cd frontend
npm run build
```

Docker Compose 启动后运行 API Smoke 流程：

```bash
scripts/smoke-api.sh
```

使用已初始化的管理员密码运行 Smoke 流程：

```bash
ADMIN_USERNAME=admin ADMIN_PASSWORD='your-current-password' scripts/smoke-api.sh
```

通过后端容器重置本地管理员密码：

```bash
scripts/reset-admin-password.sh 'TempAdmin123!'
```

## Smoke 流程覆盖点

`scripts/smoke-api.sh` 当前验证：

- 管理员登录。
- 首次密码修改门禁。
- 初始化后访问仪表盘。
- 创建运维工程师用户。
- 运维工程师执行用户管理动作时被 RBAC 拒绝。
- 运维工程师创建资产。
- 运维工程师保存凭据时被拒绝。
- 管理员保存和查看凭据。
- 保存凭据响应不暴露明文密钥。
- 任务默认类型和非法状态拒绝。
- 任务状态更新。
- 事件默认等级和非法等级拒绝。
- 事件状态更新。
- 值班规则校验。
- 每日值班创建。

## 已有交付物

- `deliverables/opscore_prototype_v15.html`
- `deliverables/asset_management_framework_review.html`
- `deliverables/cmdb_asset_ledger_review.html`
- `deliverables/middleware_database_review.html`
- `deliverables/collaboration_incident_response_review.html`
- `deliverables/oncall_management_review.html`
- `deliverables/task_tracking_review.html`
- `deliverables/incident_management_review.html`
- `deliverables/identity_permission_review.html`



## 后续会话工程备注

- `frontend/src/App.vue` 当前较大，包含路由状态、认证状态、示例数据、菜单、表单、凭据流程、列表和视图逻辑。下一步适合做组件拆分。
- `frontend/src/api.js` 集中管理 API 基础地址、Token 存储、请求封装和登录逻辑。
- 后端路由在 `backend/internal/api/server.go` 中通过 Go `http.ServeMux` 的 method/path pattern 注册。
- 后端已有 API 行为、变更规则、凭据行为、认证、加密和领域状态逻辑测试。
- 服务端校验必须和前端表单约束保持一致。
- 在权限、确认和审计日志完备前，不要加入真实自动化执行。

## 建议下一步

1. 在不改变行为的前提下，从 `App.vue` 拆分前端组件。
2. 如果 UI 继续迭代，补充小型前端 lint / test 设置。
3. 为当前接口和权限要求补充 API 文档。
4. 为凭据查看、用户管理、事件变更和未来自动化动作增加审计日志。
5. 将 AI Copilot 扩展为基于真实资产、事件、任务、值班数据和预案的权限感知工作流。

## 2026-06-17 - 暗色控制台与值班管理重设计

- 登录页主标题调整为“智能运维中枢指挥平台”，登录表单和首次初始化密码表单不再预填默认账号密码。
- 主界面延续暗色专业风，默认折叠左侧菜单栏，保留图标入口并可通过侧边栏按钮展开。
- 值班管理参考高保真原型重做为“当前值班 + 排班日历 + 交接班 + 排班记录”结构：
  - 当前值班展示主值、备值、值班窗口、覆盖范围，并提供“确认接班”动作。
  - 排班日历展示未来 14 天主备值、节假日和换班标记，管理员可从真实排班记录进入编辑。
  - 交接班区汇总活跃事件、待跟进任务和换班记录，并提供“确认交接”动作。
  - 排班记录表继续保留 daily / weekly 原始规则，供审计和后续调整。
- 交互设计检查项继续保留：新增/编辑表单不默认展示，离开页面自动取消，页面模块应可读、可点击、状态明确。
- 本次已执行 `cd frontend && npm run build`，前端构建通过。

## 2026-06-17 - 登录页增强、值班工作区拆分与 AI Copilot 配置

- 登录背景页在保持简洁的前提下增加轻量图表与 AI 节点元素，用于表达健康态势、事件影响、闭环处置和 AI 辅助能力。
- 值班管理仍归属于“协同与事件响应”，但页面内拆为值班概览、排班日历、交接班、接班管理四个工作区，避免单页混杂全部逻辑。
- 新增“系统配置 / AI Copilot 配置”页面：
  - 支持本地模型、OpenAI GPT、Anthropic Claude、Google Gemini、OpenAI 兼容接口。
  - 支持配置模型服务地址、模型名称、本地模型地址、Temperature、Max Tokens。
  - 支持资产、事件、任务、值班上下文授权和问答审计开关。
  - 明确生产 API Key 后续应由后端密钥托管和调用代理处理，不在前端落真实密钥。
- 本次已执行 `cd frontend && npm run build`，前端构建通过。

## 2026-06-17 - 登录页中枢视觉与菜单冲突清理

- 登录背景页移除割裂的独立柱形图和 AI 节点图，改为统一的“OpsCore 智能运维中枢”态势面板，围绕资产态势、值班接续、任务闭环、事件响应和 AI 决策辅助表达产品定位。
- 登录卡片从浅色背景改为暗色玻璃面板，与整体深色控制台风格保持一致。
- 左侧菜单移除“消息与通知中心”一级菜单，避免与“系统配置 / 通知渠道”重复。

## 2026-06-17 - 登录页原型文案聚焦平台定位

- `deliverables/login_command_center_review.html` 中移除“首页看健康，异常看影响...”操作口号和重复胶囊标签。
- 原型文案改为平台定位：连接资产、可观测、事件、变更、知识与自动化能力，构建面向业务连续性的统一运维控制平面。
- 保留底部“健康 / 影响 / 根因 / 处置 / 复盘”作为平台方法论主线，由中枢图承载平台能力关系。

## 2026-06-17 - 登录页原型首屏文案收敛

- `deliverables/login_command_center_review.html` 的主标题下方改为单段平台定位文案，减少双段说明造成的视觉拥挤。
- 删除次级强调段落样式，让登录页首屏由标题、单段定位文案和中枢能力图共同表达平台定位。
- `AGENTS.md` 补充登录页首屏文案约定：优先保留一段清晰的平台定位说明。

## 2026-06-17 - 登录页原型落地到正式前端

- `frontend/src/App.vue` 的登录页改为已确认原型结构：品牌、主标题、单段平台定位文案、统一中枢能力图和暗色登录卡片。
- 移除正式登录页中的胶囊式能力标签，避免首屏说明重复；中枢图节点改为统一运维数据底座、业务连续性保障、自动化处置闭环、治理与审计、AI 决策中枢。
- 登录卡片补齐原型中的“凭据加密 / RBAC / 审计预留”安全能力提示，保持正式实现与确认原型一致。
- `frontend/src/styles.css` 对齐原型的深色网格背景、登录页布局、节点视觉、登录卡片和响应式位置。
- 修复登录页移动宽度下的横向溢出问题，移动端会缩小标题和说明文字，并将中枢图节点改为流式两列布局，允许页面纵向滚动。
- `AGENTS.md` 补充：登录页正式实现应与已确认的 `deliverables/login_command_center_review.html` 保持一致。

## 2026-06-18 - Docker 镜像重建验证约定

- 当前前端、后端和数据库均按 Docker 栈启动运行。
- 后续每次完成代码修改和本地测试后，需要重新执行 `cd deploy && docker compose up --build -d`，确保前端镜像、后端镜像和运行中的服务使用最新代码。
- 验证完成后再进行页面访问、API 检查或 Smoke 流程，避免本地源码已变更但容器仍运行旧镜像。

## 2026-06-18 - 重新构建并运行最新前端镜像

- 已在 `deploy/` 执行 `docker compose up --build -d`，重新构建 `deploy-frontend:latest`，同时后端镜像也完成构建并重启。
- 当前容器状态：`deploy-frontend-1`、`deploy-backend-1`、`deploy-postgres-1` 均为运行状态，PostgreSQL 为 healthy。
- 验证结果：`http://localhost:5173/` 返回 200，`http://localhost:8080/api/health` 返回 `{"status":"ok"}`。

## 2026-06-18 - README 开源化重写

- 根据当前仓库真实结构、Docker Compose、前后端技术栈、Smoke 脚本、权限和凭据安全边界重写 `README.md`。
- README 新增项目状态、产品定位、技术栈、仓库结构、快速开始、环境变量、本地开发、容器验证、Smoke 验收、权限与安全、API 概览、数据枚举约束、贡献说明、路线图和许可证状态。
- 明确当前仓库尚未提供 `LICENSE` 文件，正式开源前需要补充许可证、贡献指南、安全策略和行为准则。

## 2026-06-18 - 值班管理模块按高保真原型重构

- 基于 `/Users/mac/Desktop/work/开发规划/duty-management-deliverables/` 下的 `duty-management.html`、PRD、数据模型和 Prisma Schema 重新梳理值班管理模块。
- 正式前端仍沿用当前 Vue 3 + Vite + 原生 CSS 架构，没有引入 React、Tailwind 或 Prisma；Prisma Schema 作为业务模型参考，不改变当前 Go + PostgreSQL 后端实现。
- `frontend/src/App.vue` 将值班管理重构为内嵌“值班中心”：
  - 概览：当前值班人员、值班指标、快速接班和统计报表；模拟告警入口已在后续版本移除。
  - 排班日历：按月日历展示主值班/备份值班，支持点击日期分配。
  - 排班配置：值班模板、轮换周期、启用/停用、编辑和删除。
  - 值班列表：人员筛选、团队配置和快速安排值班；后续版本已去掉行点击详情和详情抽屉。
  - 交接班日志：交接记录列表和提交交接弹窗。
  - 升级策略：P1 告警升级路径展示和策略编辑入口。
  - 统计报表：值班均衡度、连续值班、替班次数、满意度和图表化概览。
- `frontend/src/styles.css` 新增值班中心深色高信息密度样式，包括日历网格、模板卡片、表格操作列、团队配置、弹窗和 Toast。
- `AGENTS.md` 补充值班管理正式页面应严格参考高保真 HTML 原型的约定。
- 本次已执行 `cd frontend && npm run build`，前端构建通过；随后执行 `cd deploy && docker compose up --build -d`，前端、后端和 PostgreSQL 容器均处于运行状态。
- 验证结果：`http://localhost:5173/` 返回 200，`http://localhost:8080/api/health` 返回 `{"status":"ok"}`。

## 2026-06-18 - 值班管理二次融合设计调整

- 根据页面截图反馈，值班管理不再照搬参考原型的独立应用顶栏，移除模块内部的 `AIOps 运维平台 / 生产环境 / 搜索 / 通知 / 用户` 顶栏。
- 将值班中心左侧内嵌菜单改为横向 Tab，并把团队筛选与搜索收敛为页面工具条，更符合 OpsCore 后台整体布局。
- 将“统计报表”从独立 Tab 合并到“概览”页面，减少二级功能切换复杂度。
- 值班列表“添加人员”改为关联系统用户的正式弹窗，避免浏览器原生 prompt 和孤立姓名录入。
- 排班配置的编辑能力改为完整弹窗，支持维护模板名称、团队、轮换周期、值班时段和值班成员。
- 升级策略编辑弹窗补齐升级级别配置，可维护通知对象、升级时间和通知渠道。
- `AGENTS.md` 已同步新的值班管理设计规则：参考原型业务逻辑与文案，但视觉和交互必须融合 OpsCore 整体后台。
- 本次已执行 `cd frontend && npm run build`，前端构建通过；随后执行 `cd deploy && docker compose up --build -d`，前端、后端和 PostgreSQL 容器均处于运行状态，PostgreSQL 为 healthy。
- 复核 `AGENTS.md` 登录与密码约定后确认：
  - 登录页未展示默认账号密码，符合“登录页保持简洁，不展示默认账号密码”的要求。
  - 当前记录的后台登录密码为 `OpsCore2026`，仅保留在项目协作文档中，没有出现在登录表单默认值或页面提示中。
  - 前端支持首次登录初始化密码校验，包含当前密码、新密码、确认密码和至少 8 位校验。
  - 后端 `mustChangePassword` 门禁会阻止未初始化密码用户访问业务 API，仅允许 `/api/auth/me` 和 `/api/auth/password`。
  - 资产与实例凭据查看使用统一二次校验密码；未配置统一密码时回退当前登录密码，符合现有权限与用户体验边界。
- 最终端口 `curl` 复检因本次外部执行额度限制未能再次执行；已通过 Docker Compose 状态确认容器运行。

## 2026-06-22 - AI Copilot 配置增加连接测试

- 后端新增 `POST /api/copilot/test-connection`，仅超级管理员可调用，用于验证填写的模型接口地址、模型名称和 API Key 是否可真实访问。
- 连接测试支持 OpenAI GPT、OpenAI 兼容接口、Anthropic Claude、Google Gemini 和本地 Ollama 风格模型服务；返回连接是否可用、HTTP 状态码、响应延迟和失败原因。
- 连接测试只使用本次请求中的 API Key 发起验证，不持久化密钥，也不会在接口响应中回显密钥。
- 前端 AI Copilot 配置页新增“测试连接”按钮和测试结果状态条，测试中会禁用按钮，成功/失败会给出明确反馈。
- `AGENTS.md` 已补充 AI Copilot 连接测试的权限、密钥不落地和不泄露规则。
- 验证结果：
  - `GOCACHE=/Users/mac/Desktop/work/OpsCore/.cache/go-build go test ./...` 通过。
  - `npm run build` 通过。
  - `cd deploy && docker compose up --build -d` 已重新构建并启动前端、后端和 PostgreSQL。
  - `http://localhost:5173/` 返回 200，`http://localhost:8080/api/health` 返回 `{"status":"ok"}`。
  - `ADMIN_PASSWORD='OpsCore2026' scripts/smoke-api.sh` 通过；默认初始化密码已失效，符合管理员密码已初始化后的本地状态。
  - 容器内新接口已验证：缺少 hosted provider API Key 时返回 400。
