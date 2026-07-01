# OpsCore Full Review Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复代码审查确认的数据真实性、权限锁死、并发状态、会话失效、数据库迁移、HTTP 加固和测试覆盖问题，并完成全栈验收。

**Architecture:** 后端继续沿用 Go REST API 和 PostgreSQL，以版本化迁移替代启动时整段 DDL，并为值班中心增加团队、成员、模板、日历分配、交接和升级策略资源。前端继续使用现有 Vue 组件模式，将值班中心切换为 API 数据源，并通过统一 API 客户端处理失效会话。

**Tech Stack:** Vue 3、Vite、Playwright、Go 1.24、PostgreSQL 16、Docker Compose。

---

### Task 1: 权限与状态一致性

**Files:**
- Modify: `backend/internal/api/server.go`
- Modify: `backend/internal/store/store.go`
- Test: `backend/internal/api/mutation_test.go`

- [ ] 编写最后一个超级管理员不可降级或删除的失败测试。
- [ ] 编写任务和事件状态原子比较更新的失败测试。
- [ ] 在存储层事务中保护最后管理员，并用期望旧状态执行条件更新。
- [ ] 运行 `go test ./backend/internal/api ./backend/internal/store` 验证通过。

### Task 2: 值班中心持久化

**Files:**
- Create: `backend/internal/models/duty.go`
- Create: `backend/internal/store/duty.go`
- Create: `backend/internal/api/duty.go`
- Modify: `backend/internal/api/server.go`
- Modify: `backend/internal/store/schema.go`
- Modify: `frontend/src/components/DutyManagementView.vue`
- Modify: `frontend/src/App.vue`
- Test: `backend/internal/api/duty_test.go`
- Test: `frontend/e2e/ui-audit.spec.js`

- [ ] 编写值班团队、成员、模板、分配、交接和升级策略 CRUD 的失败 API 测试。
- [ ] 建立关系完整且可级联保护的 PostgreSQL 模型和 REST API。
- [ ] 将值班组件固定数组替换为 API 数据和明确空状态。
- [ ] 增加保存、刷新后仍存在以及删除保护的 E2E 测试。

### Task 3: 会话与 HTTP 安全

**Files:**
- Modify: `frontend/src/api.js`
- Modify: `frontend/src/App.vue`
- Modify: `backend/internal/api/server.go`
- Modify: `backend/cmd/server/main.go`
- Test: `frontend/tests/api.test.js`
- Test: `backend/internal/api/server_test.go`

- [ ] 编写 401 统一退出、请求体超限和未知字段拒绝测试。
- [ ] API 客户端发布会话失效事件并由应用清理状态。
- [ ] 限制 JSON 请求体、拒绝未知/尾随内容，配置服务读写与空闲超时及优雅退出。

### Task 4: 版本化数据库迁移

**Files:**
- Create: `backend/internal/store/migrations/001_initial.sql`
- Create: `backend/internal/store/migrations/002_duty_center.sql`
- Create: `backend/internal/store/migrations.go`
- Modify: `backend/internal/store/schema.go`
- Modify: `backend/internal/store/store.go`
- Test: `backend/internal/store/migrations_test.go`

- [ ] 编写迁移排序、只执行一次和失败回滚测试。
- [ ] 使用嵌入 SQL 和 `schema_migrations` 记录替换启动整段 DDL。
- [ ] 保持已有数据库原地升级兼容。

### Task 5: 真实行为测试与结构收敛

**Files:**
- Modify: `frontend/tests/component-boundaries.test.js`
- Modify: `frontend/e2e/ui-audit.spec.js`
- Modify: `backend/internal/api/*_test.go`
- Modify: `AGENTS.md`
- Modify: `WORKLOG.md`
- Modify: `README.md`

- [ ] 用行为测试替代值班“源码字符串即通过”的关键断言。
- [ ] 覆盖最后管理员、状态并发、会话失效和值班刷新持久化。
- [ ] 更新开发规则、真实持久化边界和运行文档。
- [ ] 运行前后端测试、构建、E2E、API Smoke、静态检查和视觉巡检。
- [ ] 执行 `docker compose up --build -d` 并确认三项健康检查。
