# WORKLOG.md

## 2026-07-14 - README 一期技术架构图

- 在 README“技术栈”标题下增加 GitHub Mermaid 一期技术架构图，采用 OpsCore 平台边界和纵向数据链路表达。
- 架构图覆盖 Nginx 统一接入、Vue 3 前端体验、Go REST API、认证与安全、核心业务、平台治理、AI Copilot、pgx 数据访问和 PostgreSQL 持久化。
- 云端模型、本地模型和 AES-GCM 环境变量密钥作为平台外部依赖展示；Docker Compose 与 Kubernetes 等部署方式继续由部署章节说明，不混入核心逻辑架构。

## 2026-07-11 - 全栈架构复审、状态流一致性与可访问交互

- 将后端超大持久化接口拆成健康、认证、用户、资产、实例、凭据安全、Copilot、值班、任务、事件和审计等领域接口，并迁移到独立 `persistence.go`。
- 将任务/事件 API 处理器和 PostgreSQL 持久化分别拆到 `api/work_items.go`、`store/work_items.go`；将凭据二次校验、资产凭据、实例凭据和旧明文迁移集中到 `store/credentials.go`。
- 新建任务强制从“待处理”开始，新建事件强制从“新建”开始；新增 API 回归测试和 Smoke 检查，禁止通过创建接口跳过闭环流程。
- 新增 `workflow-status.js`，任务/事件详情只展示当前状态与合法下一状态；内容编辑表单不再承担状态跳转，避免用户提交后才收到冲突错误。
- 新增 `useDashboardViewModel.js`，把首页指标、状态分布和优先事项编排从 `App.vue` 抽离；`App.vue` 从 1778 行降至 1674 行。
- 资产、实例、任务、事件和用户工作区补齐关键输入控件的可访问名称；四类可点击表格行同时支持 Enter 与 Space 键打开详情。
- 敏感凭据查看不再在统一校验密码缺失时回退账号登录密码；权限页明确显示“未配置，禁止查看明文凭据”，Smoke 会先配置统一校验密码再执行受控查看。
- 真实桌面/移动截图巡检后，将首页空数据优先级面板改为按内容高度布局，避免被右侧响应流程拉出大块无效留白。
- E2E 清理不再静默忽略删除失败；临时用户或业务数据若因关系约束未清理，测试会明确失败，避免巡检账号长期污染本地环境。
- 最终验证：
  - `npm run lint`、27 项前端单元/结构测试和 Vite 生产构建通过。
  - `go test ./...`、`go test -race ./...` 与 `go vet ./...` 全部通过。
  - 最新前后端镜像已重新构建，PostgreSQL、后端和前端三容器均为 healthy；健康接口返回数据库与服务均为 `ok`。
  - API Smoke 全流程通过，覆盖 RBAC、统一凭据校验、服务端分页、任务/事件初始状态与并发流转、值班规则和审计记录。
  - Chromium E2E 6/6 通过，覆盖登录、移动导航、滚动恢复、一期页面逐页交互、角色权限及桌面/移动视觉截图。
  - 最终截图保存在 `output/playwright/01-login-desktop.png` 至 `11-navigation-mobile.png`；检查未发现页面级横向溢出、表格横线错位、控件重叠或控制台 error/warning。
  - E2E 完成后数据库中 `duty-audit-*`、`ui-ops-*`、`smoke_ops_*` 临时用户计数为 0。

## 2026-06-29 - 前端组件化、实时指标与全量交互回归

- 将首页、资产、实例、值班、任务、事件、权限、AI Copilot 配置、登录/首次密码初始化、侧栏、顶栏、分页和确认弹窗从 `App.vue` 拆为聚焦组件；`App.vue` 只保留全局状态和 API 编排。
- 新增 `frontend/src/dashboard-metrics.js`，首页资产健康率、任务闭环率、事件闭环率和平均响应时长改为基于真实记录计算；没有有效样本时显示 `--`，不再展示与实际数据矛盾的固定指标。
- 新增 `frontend/src/duty-date.js`，值班日历、未来排班、值班列表和交接记录均相对当前日期生成，移除固定 `2026-06-10` 日期依赖。
- 值班组件改为保留已保存的本地工作区状态，切换页面时仅关闭未保存弹窗；升级策略取消或离页会恢复保存前草稿。
- 六类值班弹窗统一增加 `role="dialog"`、`aria-modal` 和标题关联；团队、人员、排班、交接、分配和升级策略均可被辅助技术稳定识别。
- AI Copilot 厂商卡改为语义化按钮，支持键盘焦点；普通运维工程师的模型厂商、Endpoint、模型、Key、上下文授权、测试和保存控件全部明确禁用。
- 清理旧版值班页、内嵌顶栏和内嵌侧栏遗留 CSS，构建 CSS 从约 58 KB 降至约 50 KB。
- Playwright E2E 扩展为 3 个用例，新增权限/AI 独立工作区、分页容量切换、值班 Tab 状态保持和 `ops_engineer` UI RBAC 验证。
- 真实浏览器巡检发现并修复首页第二行健康指标文字挤压；修复后主标题、数值和支持数据使用稳定三行层级。
- 验证结果：
  - `cd frontend && npm run test:unit` 通过，11 个单元/结构测试全部通过。
  - `cd frontend && npm run build` 通过，33 个模块完成生产构建。
  - `ADMIN_PASSWORD='OpsCore2026' npm run test:e2e` 在本轮中间版本通过，3 个 Chromium 用例全部通过；认证拆分和实时指标最终改动仍需额度恢复后再补跑一次最终 E2E。
  - `GOCACHE=/Users/mac/Desktop/work/OpsCore/.cache/go-build go test ./...` 与 `go vet ./...` 通过。
  - `cd deploy && docker compose up --build -d` 已按最新代码重新构建，前端、后端和 PostgreSQL 均已启动。
  - 当前 `scripts/smoke-api.sh` 在沙箱内因 Docker socket 权限被拒绝；最终 Smoke 需在允许 Docker socket 的执行窗口补跑。
  - 应用内浏览器已完成登录、首页 KPI 跳转、值班日期、值班弹窗语义、值班 Tab 状态保持、权限工作区和控制台日志检查；最终桌面/移动截图因外部浏览器执行额度限制需在 19:22 后补齐。

## 2026-06-26 - 统一危险操作确认弹窗

- 新增 `frontend/src/components/ConfirmDialog.vue`，统一资产、实例、任务、事件、值班记录、排班模板和用户删除的危险操作确认体验。
- 新增 `frontend/src/composables/useConfirmDialog.js`，将确认弹窗状态、请求、取消、确认执行和焦点返回逻辑从 `App.vue` 抽离，降低主文件继续膨胀的风险。
- 新增 `frontend/src/navigation.js` 和 `frontend/src/ui/icons.js`，集中维护菜单、路由页面元信息和线性图标路径。
- 新增 `frontend/src/components/SidebarNav.vue`，将左侧分级菜单从 `App.vue` 抽离为独立组件，并保留菜单展开、灰度占位和二级菜单点击行为。
- 新增 `frontend/src/components/Topbar.vue`，将顶层标题、面包屑、加载/错误提示和退出入口从 `App.vue` 抽离为独立组件。
- 替换前端 7 处浏览器原生 `window.confirm`，确认弹窗支持取消、关闭、遮罩、Escape、处理中禁用和焦点返回。
- Playwright UI 巡检新增确认弹窗覆盖：通过 API 创建临时巡检资产，仅删除该测试资产，验证取消、Escape、确认删除和无浏览器原生确认框。
- 修正本地源码 E2E 环境：Vite dev server 增加 `/api` 代理，前端 API 默认值改为 `/api`，与 Docker Nginx 反代行为保持一致。
- `AGENTS.md` 已同步危险删除确认弹窗规则，`README.md` 已同步本地 Vite API 代理说明。
- 验证结果：
  - 红灯验证：新增 E2E 在实现前按预期失败，原因是 `confirm-dialog` 不存在。
  - `cd frontend && npm run build` 通过。
  - `PLAYWRIGHT_BASE_URL='http://127.0.0.1:5174' ADMIN_PASSWORD='OpsCore2026' npm run test:e2e` 通过，登录页与一期逐页点击巡检共 2 个用例通过。
  - `cd deploy && docker compose up --build -d` 已重新构建前端与后端镜像，前端、后端和 PostgreSQL 均处于运行状态。
  - `docker compose ps` 确认 `deploy-frontend-1`、`deploy-backend-1`、`deploy-postgres-1` 正常运行。
  - `GOCACHE=/Users/mac/Desktop/work/OpsCore/.cache/go-build go test ./...` 通过。
  - `ADMIN_PASSWORD='OpsCore2026' scripts/smoke-api.sh` 通过。
  - 重建后默认端口 `ADMIN_PASSWORD='OpsCore2026' npm run test:e2e` 通过。
  - 抽离确认弹窗控制器、导航配置、图标路径、侧边栏组件和顶层标题栏组件后，`cd frontend && npm run build` 与 `ADMIN_PASSWORD='OpsCore2026' npm run test:e2e` 均通过。
  - 最终复检：`cd deploy && docker compose up --build -d` 后，`ADMIN_PASSWORD='OpsCore2026' npm run test:e2e`、`GOCACHE=/Users/mac/Desktop/work/OpsCore/.cache/go-build go test ./...`、`ADMIN_PASSWORD='OpsCore2026' scripts/smoke-api.sh`、`docker compose ps`、`git diff --check` 均通过。

## 2026-06-26 - 前端结构继续拆分

- 新增 `frontend/src/components/SvgIcon.vue`，统一线性图标渲染，侧边栏、首页 KPI、角色卡和 Copilot 操作按钮不再重复内联 SVG 路径模板。
- 新增 `frontend/src/components/CopilotWidget.vue`，将 AI Copilot 悬浮入口、聊天窗口、隐藏/放大按钮和输入区从 `App.vue` 抽离；问答上下文和回答生成逻辑仍保留在 `App.vue`。
- 当前验证：
  - `cd frontend && npm run build` 通过。
  - `git diff --check` 通过。
  - 原生 `window.confirm` / `window.prompt` / `window.alert` 残留扫描通过。
- 当前限制：本轮 Playwright E2E 外部执行被额度限制拒绝，提示需等到 19:35 后恢复；为避免扩大未验证交互改动，后续页面级拆分暂停，额度恢复后优先补跑 `ADMIN_PASSWORD='OpsCore2026' npm run test:e2e`。

## 2026-06-25 - AI Copilot Endpoint SSRF 防护加固

- AI Copilot 配置保存与连接测试统一执行 Endpoint 地址策略校验，hosted provider 不能保存 loopback、私网、链路本地或云元数据服务地址。
- 连接测试使用专用 HTTP Client：
  - 禁用环境代理，避免通过代理绕过目标地址校验。
  - 每次 HTTP 重定向重新校验目标 URL，并限制最多 5 次重定向。
  - 请求拨号前解析域名并校验全部 IP；实际连接已校验的 IP，避免 DNS 重绑定窗口。
  - 本地模型 provider 继续允许私网地址和 `host.docker.internal`，但仍拒绝云元数据服务和其他危险特殊地址。
- 新增配置保存、重定向、hosted DNS 私网结果、本地私网模型和元数据 DNS 结果的回归测试。
- 修正 `README.md` 中“后续增加后端密钥托管”的过时描述，明确密钥托管已完成。
- 删除仍被 Git 跟踪的 `backend/cmd/.DS_Store` 和 `backend/internal/.DS_Store`；仓库已通过 `.gitignore` 持续忽略该类文件。
- 验证结果：
  - `GOCACHE=/Users/mac/Desktop/work/OpsCore/.cache/go-build go test ./...` 通过。
  - `cd frontend && npm run build` 通过。
  - `ADMIN_PASSWORD='OpsCore2026' npm run test:e2e` 通过，登录页与一期逐页点击巡检共 2 个用例通过。
  - `ADMIN_PASSWORD='OpsCore2026' scripts/smoke-api.sh` 通过。
  - `cd deploy && docker compose up --build -d` 已重新构建前端与后端镜像，前端、后端和 PostgreSQL 均处于运行状态。
  - `http://localhost:5173/` 返回 200，`http://localhost:8080/api/health` 返回 `{"status":"ok"}`。
  - 运行态验证确认：hosted provider 保存私网 Endpoint 返回 400，响应未泄露测试 API Key。
  - `git diff --check` 通过。

## 2026-06-23 - 技术架构审查后修复项

- 清理 Git 中误跟踪的 Go 构建缓存：`backend/.gocache/` 已从索引移除，后续继续由 `.gitignore` 忽略；本地测试建议使用 `.cache/go-build` 作为 `GOCACHE`。
- 前端示例数据改为显式开关控制：新增 `VITE_ENABLE_DEMO_DATA`，默认 `false`；只有静态演示或无后端演示时才显示 sample 数据，避免真实生产空数据被样例数据掩盖。
- Docker 构建链路同步 `VITE_ENABLE_DEMO_DATA` build arg，Compose 模板和前端模板默认均为关闭。
- AI Copilot 连接测试增加服务端 Endpoint 访问限制：hosted provider 默认拒绝 loopback、私网、链路本地和云元数据地址；本地模型 provider 仍允许 `localhost`、私网地址和 `host.docker.internal`。
- AI Copilot 连接失败信息继续做密钥清洗，包含 query escape 后的 API Key 形态，避免 Gemini 等 provider 的 Key 从 URL 或错误详情泄露。
- `AGENTS.md` 和 `README.md` 已同步演示数据开关、Go 缓存路径和 Copilot Endpoint 安全约束。
- 验证结果：
  - `git diff --check` 通过。
  - `GOCACHE=/Users/mac/Desktop/work/OpsCore/.cache/go-build go test ./...` 通过；沙箱内因 `httptest` 监听端口受限失败，外部执行通过。
  - `cd frontend && npm run build` 通过。
  - `cd frontend && npm run test:e2e` 通过；沙箱内因 Vite 监听 `5173` 受限失败，外部执行通过。
  - `cd deploy && docker compose up --build -d` 已重新构建并启动前端、后端和 PostgreSQL。
  - `http://localhost:5173/` 返回 200，`http://localhost:8080/api/health` 返回 `{"status":"ok"}`。
  - `ADMIN_PASSWORD='OpsCore2026' scripts/smoke-api.sh` 通过。

## 2026-06-23 - 逐页 UI 巡检用例补充

- 新增 `frontend/e2e/ui-audit.spec.js`，用于覆盖一期页面的真实点击路径：
  - 登录页进入控制台。
  - 首页四个 KPI 卡片跳转到资产、值班、任务和事件页面。
  - 左侧菜单进入资产台账、中间件与数据库、值班管理、任务跟踪、事件管理、权限管理和 AI Copilot 配置。
  - 资产、中间件、任务、事件页面验证新增/取消、行点击详情、详情隐藏、分页条和高级搜索。
  - 值班管理验证概览、排班日历、排班配置、值班列表、交接班日志、升级策略，以及分配值班、添加人员、团队配置、提交交接和编辑策略弹窗的打开/取消。
  - 权限管理验证新增用户弹窗和菜单/资源权限 Tab。
  - AI Copilot 配置验证本地模型配置入口、测试连接反馈和全局悬浮 Copilot 打开/隐藏。
- 巡检用例会在必要时通过 API 补充临时资产、中间件、值班、任务和事件数据，并在结束后清理，避免空列表导致巡检空转。
- 当前验证结果：
  - `node --check frontend/e2e/ui-audit.spec.js` 通过。
  - `cd frontend && npm run build` 通过。
  - `git diff --check` 通过。
  - 真正的 `ADMIN_PASSWORD='OpsCore2026' npx playwright test e2e/ui-audit.spec.js --project=chromium` 由于 Codex 外部执行额度限制被拒绝，尚未完成真实浏览器逐页点击执行；额度恢复后需要第一时间补跑。

## 2026-06-24 - 逐页 UI 巡检执行与交互修复

- 成功执行 `frontend/e2e/ui-audit.spec.js` 的真实浏览器逐页点击巡检，覆盖登录、首页 KPI 跳转、资产台账、中间件与数据库、值班管理、任务跟踪、事件管理、权限管理、AI Copilot 配置和全局 Copilot 悬浮入口。
- 巡检中发现并修复权限管理顶层标题问题：点击“用户与角色”时顶层标题不再显示子页名，统一保留为“权限管理”，子页名交由 Tab 表达。
- 巡检中发现并修复编辑表单失焦关闭的运行时错误：`closeEditorOnFocusOut` 现在会同步缓存面板 DOM，避免异步阶段 `event.currentTarget` 变为 `null` 导致 `contains` 报错。
- 修正 E2E 巡检用例中两处不稳定选择器：详情面板按用户可见的“隐藏详情”按钮断言，值班弹窗按实际 `.duty-modal` 结构定位。
- 验证结果：
  - `ADMIN_PASSWORD='OpsCore2026' npx playwright test e2e/ui-audit.spec.js --project=chromium` 通过。
  - `cd frontend && npm run test:e2e` 通过，登录 smoke 和逐页点击巡检共 2 个用例均通过。
  - `GOCACHE=/Users/mac/Desktop/work/OpsCore/.cache/go-build go test ./...` 通过。
  - `ADMIN_PASSWORD='OpsCore2026' scripts/smoke-api.sh` 通过。
  - `cd deploy && docker compose up --build -d` 已重新构建并启动前端、后端和 PostgreSQL。
  - `http://localhost:5173/` 返回 200，`http://localhost:8080/api/health` 返回 `{"status":"ok"}`。

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

## 2026-06-23 - AI Copilot 连接测试安全与本地模型体验优化

- 补充 Google Gemini 连接失败场景的回归测试，确认 query string 中的 API Key 不会出现在接口响应或前端提示中。
- 后端连接失败错误统一经过密钥清洗后再返回；本地模型使用 `localhost` / `127.0.0.1` 且连接失败时，会提示 Docker 栈应使用 `http://host.docker.internal:11434`。
- 前端本地模型默认地址从 `http://localhost:11434` 调整为 `http://host.docker.internal:11434`，更符合当前 Docker Compose 运行方式。
- `AGENTS.md` 已同步连接测试错误清洗和 Docker 本地模型地址约定。
- 验证结果：
  - `GOCACHE=/Users/mac/Desktop/work/OpsCore/.cache/go-build go test ./internal/api -run Copilot` 通过。
  - `GOCACHE=/Users/mac/Desktop/work/OpsCore/.cache/go-build go test ./...` 通过。
  - `npm run build` 通过。
  - `cd deploy && docker compose up --build -d` 已重新构建并启动前端、后端和 PostgreSQL。
  - `http://localhost:5173/` 返回 200，`http://localhost:8080/api/health` 返回 `{"status":"ok"}`。
  - `ADMIN_PASSWORD='OpsCore2026' scripts/smoke-api.sh` 通过。
  - 运行态验证确认：Gemini 失败路径不泄露 API Key，本地模型 `localhost` 失败路径会提示 `host.docker.internal`。

## 2026-06-23 - Docker Compose 宿主机模型访问兼容

- `deploy/docker-compose.yml` 为 backend 服务新增 `extra_hosts: ["host.docker.internal:host-gateway"]`。
- 该配置让后端容器在 Linux/远程 Docker 环境下也能解析 `host.docker.internal`，与 AI Copilot 本地模型默认地址保持一致。
- `AGENTS.md` 已同步该部署约定，避免后续误删导致本地模型连接测试在非 Docker Desktop 环境失效。
- 验证结果：
  - `docker compose config` 通过，backend 已解析出 `extra_hosts: host.docker.internal=host-gateway`。
  - `cd deploy && docker compose up --build -d` 已重新构建并启动前端、后端和 PostgreSQL。
  - `docker compose exec -T backend getent hosts host.docker.internal` 可解析到宿主机网关地址。
  - `http://localhost:5173/` 返回 200，`http://localhost:8080/api/health` 返回 `{"status":"ok"}`。
  - `ADMIN_PASSWORD='OpsCore2026' scripts/smoke-api.sh` 通过。

## 2026-06-23 - AI Copilot 配置后端加密托管

- 新增 `GET /api/copilot/config` 和 `PUT /api/copilot/config`，仅超级管理员可读写 AI Copilot 配置。
- Copilot 配置保存复用 `system_settings`，API Key 通过现有 AES-GCM `credentialBox` 加密托管，不新增表结构。
- 配置读取和保存响应只返回 `hasApiKey`，不会返回明文 API Key；前端保存成功后清空 API Key 输入框，仅显示托管状态。
- 测试连接接口支持在请求未携带 API Key 时复用后端托管 Key；切换模型厂商或 Endpoint 且未重新输入 Key 时，后端会清理旧托管 Key，避免跨厂商错配。
- 前端 AI Copilot 配置页启动时会加载后端配置，保存按钮改为真正调用后端持久化接口。
- 验证结果：
  - `GOCACHE=/Users/mac/Desktop/work/OpsCore/.cache/go-build go test ./internal/api -run Copilot` 通过。
  - `GOCACHE=/Users/mac/Desktop/work/OpsCore/.cache/go-build go test ./...` 通过。
  - `npm run build` 通过。
  - `cd deploy && docker compose up --build -d` 已重新构建并启动前端、后端和 PostgreSQL。
  - `http://localhost:5173/` 返回 200，`http://localhost:8080/api/health` 返回 `{"status":"ok"}`。
  - `ADMIN_PASSWORD='OpsCore2026' scripts/smoke-api.sh` 通过。
  - 运行态验证确认：保存/读取 Copilot 配置均只返回 `hasApiKey=true`，不会返回明文 API Key。

## 2026-06-23 - 前端安装 Playwright E2E

- `frontend/` 安装 `@playwright/test`，并通过 `npx playwright install chromium` 安装 Chromium、Headless Shell 和 FFmpeg 浏览器依赖。
- `frontend/package.json` 新增脚本：
  - `npm run test:e2e`
  - `npm run test:e2e:headed`
  - `npm run test:e2e:ui`
- 新增 `frontend/playwright.config.js`，默认复用 `http://localhost:5173`，若无现有服务则启动 Vite dev server。
- 新增 `frontend/e2e/login.spec.js`，覆盖登录页标题、账号、密码和进入控制台按钮的最小 smoke 检查。
- `.gitignore` 增加 `frontend/test-results/` 和 `frontend/playwright-report/`，避免 Playwright 运行产物进入版本变更。
- 安装后执行 `npm audit fix`，当前 npm audit 已从 1 个 Vite 高危项修复为 0 vulnerabilities。
- 验证结果：
  - `npm audit` 返回 0 vulnerabilities。
  - `npm run build` 通过，Vite 已升级到 6.4.3。
  - `npm run test:e2e` 通过，Chromium 下 1 个登录页 smoke 用例通过。
  - `cd deploy && docker compose up --build -d` 已重新构建并启动前端、后端和 PostgreSQL。
  - `http://localhost:5173/` 返回 200，`http://localhost:8080/api/health` 返回 `{"status":"ok"}`。
  - `ADMIN_PASSWORD='OpsCore2026' scripts/smoke-api.sh` 通过。

## 2026-06-27 - 前端结构、分页与视觉巡检优化

- 将首页 KPI、运营指标、事件/任务优先级和事件响应流程提取为 `DashboardView.vue`，保持原有跳转和优先级项交互。
- 新增统一 `PaginationBar.vue`，替换资产、中间件、任务和事件四处重复分页；补齐上一页/下一页可访问名称、边界禁用和窗口化页码。
- Playwright 先验证旧分页缺少可访问名称和边界禁用，再验证新组件通过；完整一期页面点击巡检继续通过。
- 清理旧版值班单条/批量录入、旧接班/交接确认、旧概览计算等不可达逻辑，并移除未使用的事件状态统计，`App.vue` 从 3536 行降至 3183 行。
- 真实浏览器视觉巡检发现 1280px 桌面宽度下登录主标题孤字换行；新增标题高度回归断言并将桌面固定字号调整为 56px，复截图确认标题单行完整显示。
- 将关闭状态的 AI Copilot 入口从页面右下角悬浮按钮迁到顶栏图标按钮，避免遮挡首页事件响应流程；聊天窗的放大、还原、隐藏和发送交互保持不变。
- 更新 `AGENTS.md`，补充聚焦组件目录、统一分页、Copilot 顶栏入口和登录页视觉回归约定。
- 视觉巡检产物保存于 `output/playwright/`，覆盖登录页、首页、资产台账和值班概览。
- 最终验证：`npm run build`、`GOCACHE=/Users/mac/Desktop/work/OpsCore/.cache/go-build go test ./...`、`ADMIN_PASSWORD='OpsCore2026' npm run test:e2e`（2/2）、`ADMIN_PASSWORD='OpsCore2026' scripts/smoke-api.sh` 和 `docker compose up --build -d` 均通过。
- `docker compose ps` 确认 frontend、backend 均为 Up，PostgreSQL 为 Up (healthy)。最终宿主机 `curl` 复检因外部执行额度限制未执行；Playwright 已从 5173 访问前端，API Smoke 已完成真实登录、资产、凭据、任务、事件和值班接口验证。

## 2026-06-29 - 一期业务页组件化与全栈回归

- 按页面边界新增 `TaskView.vue`、`IncidentView.vue`、`AssetView.vue`、`MiddlewareView.vue` 和 `DutyManagementView.vue`，将任务、事件、资产、实例和值班工作区从 `App.vue` 中分离。
- `App.vue` 保留认证、全局数据、API 请求、RBAC 判断和统一确认流程；页组件通过明确 props/emits 连接，文件行数从 3183 降至 2083。
- 值班页专属的 Tab、日历、团队、排班、人员、交接和升级策略状态收敛到 `DutyManagementView.vue`；写操作统一受 `canWriteOncall` 控制，删除排班继续复用全局应用内确认弹窗。
- Playwright 为五个业务工作区新增可访问 `region` 断言；每个断言均先在旧实现上失败，再在组件拆分后通过。
- 逐组件执行 `npm run build`、`docker compose up --build -d` 和一期页面点击巡检，确认筛选、表单打开/取消、详情隐藏、删除确认、分页、值班 Tab/弹窗和 Copilot 入口无回归。
- 全量验证通过：`go test ./...`、`go vet ./...`、`npm run build`、`ADMIN_PASSWORD=OpsCore2026 npm run test:e2e`（2/2）、`ADMIN_PASSWORD=OpsCore2026 scripts/smoke-api.sh` 和 Docker Compose 重建。
- 真实浏览器截图已复核登录、首页、资产、实例和值班页，未发现重叠、溢出或表格横线错位。继续截取任务、事件、权限和 Copilot 页时遇到外部 Playwright 执行额度限制，该项未写为已完成。
- 静态审查发现值班页的“今天”和“未来 3 天”固定在 `2026-06-10`；新增 `src/duty-date.js` 和 Node 原生单元测试，将排班生成、未来值班、当天标记、接班日期和交接时间改为基于实际当前日期计算。
- `npm run test:unit` 先在辅助函数缺失时 2/2 失败，实现后 2/2 通过；随后 `npm run build` 通过并完成 Docker Compose 镜像重建。
- 继续拆分 `AuthView.vue`、`PermissionsView.vue` 和 `CopilotSettingsView.vue`，应用壳不再直接承载登录页、权限页和 Copilot 配置页模板；新增组件边界测试防止页面逻辑回流。
- 首页健康指标改为根据真实资产、实例、任务和事件数据计算；无有效数据时展示 `--`，不再使用固定的健康率、响应时间和闭环率演示数字。
- 清理已废弃的旧值班布局 CSS，前端构建 CSS 体积由约 58 KB 降至约 50 KB；值班日期改为按当前日期生成，避免固定年份数据。
- 真实浏览器 1280×720 巡检确认首页指标文案不再重叠，8 个一期路由直达并刷新后均保留当前页面，且每页只有一个一级标题、无默认弹窗、无页面级横向溢出和控制台错误。
- 390×844 视觉巡检发现折叠侧栏的高优先级网格规则把主内容压缩到 82px；先增加失败的 Playwright 回归，再修复小屏单列网格，修复后主内容宽 390px、KPI 宽约 342px。
- 小屏隐藏桌面侧栏后新增顶栏移动导航按钮、遮罩抽屉和选择页面后自动关闭逻辑；Playwright 已覆盖抽屉打开、切换到任务跟踪和自动关闭。
- 修复跨模块导航保留旧滚动位置的问题；真实浏览器从值班页 `667.5px` 位置切换任务页后已回到 `0px`，对应 Playwright 回归已加入并等待完整 E2E 续跑。
- 后端权限审查发现授权使用 Token 中旧角色的问题；新增角色降级回归测试并改为每次请求使用数据库当前角色，目标测试已由 200 错误放行转为预期 403。
- README 明确值班中心当前持久化边界：后端已支持基础 `daily` / `weekly` 值班记录，团队、模板、日历分配、交接和升级策略仍需后续领域表与 API。
- 部署审查将前端镜像依赖安装改为 `npm ci`，并为 backend/frontend 增加健康检查；frontend 改为等待 backend 健康后启动，Compose 状态可直接反映三层服务可用性。
- 后端全包回归发现旧测试夹具仍把数据库用户固定为 `super_admin`，与“按数据库当前角色授权”的新规则冲突；统一补齐 `ops_engineer` 用户夹具后，资产允许写入和用户/凭据/值班禁止写入场景均按真实当前角色验证。
- 最终验证已补齐：前端单元测试 12/12、生产构建、Chromium E2E 5/5、后端 `go test -count=1 ./...`、`go vet ./...`、最终镜像重建后的 API Smoke 和 `git diff --check` 均通过。
- Playwright CLI 复核 1280×720 登录页、首页、值班、权限、Copilot 配置页以及 390×844 移动首页/导航抽屉；页面无框架错误、控制台无 error/warn、无页面级横向溢出，移动导航切换任务页后自动关闭。Browser 插件在布局截图阶段连续超时，按既有授权回退仓库 Playwright Chromium 完成视觉证据采集。
- 最终执行 `docker compose up --build -d`，frontend、backend、PostgreSQL 均为 `healthy`；静态扫描未发现原生 `alert/confirm/prompt`、调试输出、固定生产日期、表格单元格 flex 或默认开启演示数据。

## 2026-06-30 - 全量代码审查修复与值班持久化

- 用户与角色：存储层使用事务级 advisory lock 保护超级管理员角色，禁止降级或删除系统最后一个 `super_admin`；接口返回 409，权限页同步禁用唯一管理员的角色下调。
- 任务与事件：状态更新和完整编辑均在 PostgreSQL `FOR UPDATE` 行锁事务内再次校验状态流转，消除并发请求基于旧状态产生非法回退的竞态。
- 值班中心：新增 `/api/duty-center` 读写接口和 `duty_center_state` 持久化表，覆盖团队、系统用户成员、排班模板、日历分配、当前值班、交接日志和升级策略；revision 乐观锁阻止多用户静默覆盖。
- 值班前端移除固定人员、告警和响应指标，所有统计改为真实持久化数据计算；补齐成员编辑/移除、团队引用保护、升级层级增删和明确空状态。
- 会话安全：API 错误保留 HTTP status，已认证请求返回 401 时统一清理会话；Bearer Token 从 `localStorage` 改为标签页级 `sessionStorage`。
- HTTP 加固：JSON 请求体限制为 1 MiB，拒绝未知字段和尾随 JSON；Go HTTP Server 增加读写、Header、Idle 超时和 SIGTERM 优雅退出；Nginx 增加 CSP、禁止嵌入、MIME 嗅探和权限策略响应头。
- 数据库：引入 `schema_migrations` 版本记录，现有结构作为 `001_initial`，值班中心作为 `002_duty_center`，迁移只执行一次并在事务中提交。
- 资产台账：取消已从产品设计移除的“并网状态”字段，迁移 `003_remove_connected_status` 同步删除旧数据库列。
- 结构整理：后端校验规则拆至 `validation.go`，前端演示数据与权限/Copilot 静态配置拆至独立模块。
- 行为验证：扩展 API Smoke，覆盖最后管理员保护、未知 JSON 字段、值班读写权限以及任务/事件关闭后不可回退；Playwright 使用真实 API 临时建立值班数据，验证 UI 保存后刷新仍存在并在测试后恢复原状态。
- 全量 E2E 复验发现值班中心 revision 大于 0 后无法继续保存；存储层改为事务内按 revision 条件更新，仅在 revision 为 0 时初始化插入，修复后刷新持久化与并发冲突检查均通过。
- 桌面视觉复验发现无值班人员时“值班均衡度”错误显示 100%；新增 `calculateDutyBalance` 纯逻辑及回归测试，无有效样本时统一显示 `--`。
- 最终验证结果：前端单元测试 16/16、生产构建、Chromium E2E 5/5、后端 `go test -count=1 ./...`、`go vet ./...`、扩展 API Smoke、迁移版本与字段检查、Nginx 安全响应头、`git diff --check` 和 Docker Compose 三容器健康检查均通过。
- Playwright CLI 已完成 1280×720 值班页和 390×844 值班页/首页截图复验；移动端 `scrollWidth` 与 `clientWidth` 均为 390，控制台 error/warn 均为 0，未发现页面级横向溢出或组件重叠。

## 2026-07-10 - 生产配置门禁、认证限流与操作审计

- 新增 `OPSCORE_ENV` 和生产配置校验；生产模式拒绝占位数据库密码、JWT、AES-GCM 凭据密钥和管理员初始密码。
- 新增 `deploy/docker-compose.production.yml`，远程/生产启动必须显式提供关键密钥；本地 Compose 继续保留开发默认值。
- 登录和资产/实例凭据二次验证增加失败次数窗口与临时锁定，达到阈值返回 `429` 和 `Retry-After`，成功验证后清除失败状态。
- 新增迁移 `004_audit_events`、审计存储层和 `GET /api/audit-events`；关键写操作、登录和凭据查看记录操作人、资源、结果、IP 与时间，不记录请求正文和敏感值。
- 权限管理菜单启用“操作审计”页面，支持账号/动作/资源/IP 搜索与成功/失败筛选，仅超级管理员可查看。
- 500、PostgreSQL、加密和 SQLSTATE 等底层错误改为服务端记录详细信息、客户端返回安全错误文案。
- 验证结果：后端 `go test ./...`、`go vet ./...`、前端单元测试 16/16、生产构建、Docker Compose 重建、迁移版本检查、API Smoke 和 Chromium E2E 5/5 均通过；Smoke 产生的登录、凭据、用户、值班、任务和事件审计记录已在 PostgreSQL 验证。

## 2026-07-10 - 服务端分页、关系化值班与真实 Copilot 代理

- 资产、实例、任务和事件列表改为服务端分页、筛选、排序和聚合计数；页码超过有效范围时自动回落最后一页，迁移 `005`/`008` 增加组合索引和 `pg_trgm` 模糊搜索索引。
- 迁移 `006_typed_dates_user_refs` 将任务、事件和值班日期改为 PostgreSQL `timestamptz`/`date`，负责人关联 `users` 外键；前端 `datetime-local` 统一转换为带时区 RFC3339，避免东八区时间偏移。
- 负责人有关联用户 ID 时由后端读取当前显示名，用户改名会事务内同步任务、事件、值班和关系化值班名单；已进入值班名单的用户删除返回 409。
- 迁移 `007_relational_duty_center` 新增团队、成员、模板、日历分配、当前值班、交接和升级策略关系表；JSON 只保留兼容快照，关系表与 revision 在同一事务提交并在启动时回填旧快照。
- 新增 `POST /api/copilot/chat`：真实代理本地 Ollama、OpenAI 兼容、Anthropic 和 Gemini，按当前角色与配置开关注入非敏感上下文，复用 DNS/重定向/SSRF 防护，并限制单用户/IP 每分钟 20 次。
- Copilot 问答审计改为强制安全边界；前端聊天加入加载、失败、自动聚焦、Escape 隐藏和可访问状态播报。
- 新增全局语义 Toast、`useCopilotChat`、`useDeferredLoader`、`useToast` 和无依赖 `npm run lint`；静态检查禁止原生弹窗、持久化认证数据、调试日志和未经审计的 `v-html`。
- 生产 Compose 覆盖取消 PostgreSQL 与后端宿主机端口映射，为后端启用只读文件系统和 `no-new-privileges`，并增加生产重启策略。
- 健康检查增加 PostgreSQL readiness；生产内部反向代理启用可信客户端 IP 头，本地直连默认不信任，避免伪造 IP 绕过限流或污染审计。
- Dashboard 聚合从多次串行计数收敛为单次汇总查询加两次分组查询；今日值班优先读取关系化当前值班，基础周排班只统计当前 ISO 周。

## 2026-07-11 - AI Copilot 多模型配置与列表优先工作区

- 按已确认的方案 A 将单条 Copilot 配置升级为多模型配置档案；新增迁移 `009_copilot_model_configs`、名称唯一索引和唯一当前启用部分索引。
- 迁移会把既有 `system_settings.copilot_config` 直接转换为“默认模型配置”，保留原有 AES-GCM 密文，不接触或重新暴露 API Key 明文；真实 PostgreSQL 已验证兼容配置、模型、密钥状态和启用状态均保留。
- 新增模型配置列表、创建、编辑、删除、连接测试和设为当前 API；首条配置自动启用，当前配置禁止删除，Hosted 模型启用前必须存在托管 Key。
- 兼容 `GET/PUT /api/copilot/config` 的读取路径已映射到当前启用配置；`POST /api/copilot/chat` 继续只读取服务端当前启用项和当前用户授权上下文。
- 前端改为列表优先：默认展示配置名称、厂商、模型、Endpoint、密钥状态和启用状态，新增/编辑后才展开表单；Local 与 Hosted 配置字段互斥显示。
- 参数文案改为“回答随机性（Temperature）”和“最大输出长度（Max Tokens）”，默认值统一为 `0.2` 和 `2048`。
- Playwright 移动端回归发现从桌面编辑态切换到 390px 后 Copilot 网格被表格最小内容宽度撑到 1092px；将工作区轨道约束为 `minmax(0, 1fr)` 并允许子项收缩后，页面级 `scrollWidth` 恢复为 390px。
- 验证通过：前端单元测试 30/30、lint、Vite 生产构建、Go 全包测试、`go vet ./...`、API Smoke、Chromium E2E 6/6、Copilot 桌面编辑态和移动端视觉回归、Docker Compose 三容器健康检查。
- 最终仅追加表格操作按钮暗色可读性样式后，单元测试、lint、生产构建和 Docker 镜像重建再次通过；外部执行额度恢复后已对当前最终镜像补跑完整 Chromium E2E，6/6 通过，并再次通过 Go 全包测试、`go vet ./...`、API Smoke、迁移唯一启用约束和三容器健康检查。
