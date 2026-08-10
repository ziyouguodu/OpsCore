# AI Copilot Model Profiles Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace OpsCore's single Copilot configuration with secure multi-model profiles and a list-first management workspace with one explicitly active profile.

**Architecture:** Add a versioned PostgreSQL migration and focused store methods for profile CRUD, encrypted keys, and transactional activation. Expose super-admin REST endpoints while retaining the legacy active-config adapter used by chat. Replace the always-visible single form with a profile table and conditional Local/Hosted editor.

**Tech Stack:** Go 1.24 `net/http`, PostgreSQL 16 with `pgx/v5`, AES-GCM credential box, Vue 3, Vite, Node test runner, Playwright, Docker Compose.

## Global Constraints

- Exactly zero or one profile may be active; the first created profile becomes active.
- API keys are encrypted at rest and never returned by API responses or audit details.
- Only `super_admin` may list, create, edit, test, activate, or delete profiles.
- Local profiles show and validate only `localEndpoint` and `localModel`; hosted profiles show and validate only `endpoint`, `model`, and `apiKey`.
- Current active profiles cannot be deleted.
- Temperature is `0..2`, defaults to `0.2`; Max Tokens is `1..4096`, defaults to `2048`.
- Preserve the existing endpoint SSRF, redirect, DNS pinning, error sanitization, audit, and rate-limit boundaries.
- Every code change is followed by relevant tests; final delivery rebuilds the Docker Compose stack and updates `AGENTS.md` and `WORKLOG.md`.

---

### Task 1: Model and migration

**Files:**
- Modify: `backend/internal/models/models.go`
- Modify: `backend/internal/store/schema.go`
- Modify: `backend/internal/store/migrations.go`
- Test: `backend/internal/store/migrations_test.go`

**Interfaces:**
- Produces: `models.CopilotModelConfig` with `ID`, `Name`, provider fields, masked key state, generation parameters, context flags, active state, and timestamps.
- Produces: migration `009_copilot_model_configs` and table `copilot_model_configs`.

- [ ] Write a failing migration test that asserts the new migration version and SQL constraints/indexes exist.
- [ ] Run `GOCACHE=/Users/mac/Desktop/work/OpsCore/.cache/go-build go test ./internal/store -run CopilotModel -count=1` and confirm it fails because migration `009` is absent.
- [ ] Add `CopilotModelConfig` and migration SQL with a case-insensitive unique name index and a unique partial active index.
- [ ] Add SQL migration logic that copies `system_settings.copilot_config` into “默认模型配置” without exposing or re-encrypting `apiKeyEncrypted`.
- [ ] Re-run the focused store tests and confirm PASS.

### Task 2: Store profile lifecycle

**Files:**
- Create: `backend/internal/store/copilot_profiles.go`
- Modify: `backend/internal/store/copilot.go`
- Modify: `backend/internal/store/store.go`
- Test: `backend/internal/store/copilot_profiles_test.go`

**Interfaces:**
- Produces: `ListCopilotModelConfigs(context.Context) ([]models.CopilotModelConfig, error)`.
- Produces: `CreateCopilotModelConfig(context.Context, models.CopilotModelConfig) (models.CopilotModelConfig, error)`.
- Produces: `UpdateCopilotModelConfig(context.Context, int64, models.CopilotModelConfig) (models.CopilotModelConfig, error)`.
- Produces: `DeleteCopilotModelConfig(context.Context, int64) error`.
- Produces: `ActivateCopilotModelConfig(context.Context, int64) (models.CopilotModelConfig, error)`.
- Produces: `GetCopilotModelConfig(context.Context, int64)`, `GetActiveCopilotModelConfig(context.Context)`, and encrypted-key access for the selected profile.

- [ ] Write failing store tests for first-profile activation, masked list output, key retention, provider/endpoint key clearing, active deletion protection, and activation switching.
- [ ] Run focused tests and confirm failures identify missing store methods.
- [ ] Implement profile row scanning and public masking helpers in `copilot_profiles.go`.
- [ ] Implement CRUD with AES-GCM key handling and `pgx` transactions; lock activation rows before switching.
- [ ] Make legacy `GetCopilotConfig` and `GetCopilotAPIKey` read the active profile while keeping a safe default when no profile exists.
- [ ] Run focused store tests and all `./internal/store` tests.

### Task 3: REST API and chat integration

**Files:**
- Modify: `backend/internal/api/persistence.go`
- Modify: `backend/internal/api/server.go`
- Create: `backend/internal/api/copilot_profiles.go`
- Modify: `backend/internal/api/copilot.go`
- Modify: `backend/internal/api/copilot_chat.go`
- Modify: `backend/internal/api/audit.go`
- Modify: `backend/internal/api/mutation_test.go`
- Test: `backend/internal/api/copilot_test.go`

**Interfaces:**
- Produces REST endpoints under `/api/copilot/configs` plus `/{id}/activate` and `/{id}/test-connection`.
- Consumes the store methods from Task 2.

- [ ] Add failing API tests for RBAC, masked profile CRUD, current deletion rejection, activation, saved-key connection test, and chat active-profile usage.
- [ ] Run `go test ./internal/api -run Copilot -count=1` and confirm expected failures.
- [ ] Extend the persistence interface and test store with profile methods.
- [ ] Register routes and implement strict ID parsing, validation, normalized defaults, and safe conflict/not-found responses.
- [ ] Reuse `testCopilotConnection` for unsaved and saved profiles; fetch the stored key only on the server.
- [ ] Add distinct audit descriptors for create, update, delete, activate, and test actions without key material.
- [ ] Run focused API tests, then all backend tests and `go vet ./...`.

### Task 4: Frontend state and list-first workspace

**Files:**
- Modify: `frontend/src/App.vue`
- Modify: `frontend/src/components/CopilotSettingsView.vue`
- Modify: `frontend/src/styles.css`
- Modify: `frontend/tests/component-boundaries.test.js`
- Test: `frontend/tests/copilot-settings.test.js`

**Interfaces:**
- Consumes: profile list and CRUD endpoints from Task 3.
- Produces: editor state modes `closed`, `create`, and `edit`, with provider-conditional form fields.

- [ ] Add failing source/component tests proving non-Local markup is conditional, Local markup excludes hosted fields, and the saved-profile table/actions exist.
- [ ] Run `npm run test:unit` and confirm the new tests fail for missing behavior.
- [ ] Add profile list/loading state and API orchestration to `App.vue`; preserve unsaved form cancellation when leaving the page.
- [ ] Refactor `CopilotSettingsView.vue` to show the table by default and reveal the editor only after Create/Edit.
- [ ] Render Local fields only when provider is `local`; otherwise render Endpoint, model, and API Key only.
- [ ] Add actions for test, save, activate, and delete; use the shared confirm dialog and Toast.
- [ ] Add compact dark-theme table/editor/mobile styles without nested cards or page-level overflow.
- [ ] Run unit tests, lint, and production build.

### Task 5: Browser workflows and documentation

**Files:**
- Modify: `frontend/e2e/ui-audit.spec.js`
- Modify: `frontend/e2e/visual-regression.spec.js`
- Modify: `AGENTS.md`
- Modify: `WORKLOG.md`
- Modify: `README.md` only if operator-visible API or configuration instructions require it.

**Interfaces:**
- Verifies all earlier tasks through the running Docker Compose stack.

- [ ] Add Playwright coverage for opening the profile editor, Local/Hosted field switching, cancel, current-state visibility, and desktop/mobile overflow.
- [ ] Update documentation with multi-profile behavior, unique active configuration, defaults, encryption, and verification evidence.
- [ ] Run `npm run test:unit`, `npm run lint`, `npm run build`, backend `go test ./...`, and `go vet ./...`.
- [ ] Run `docker compose up --build -d` from `deploy/` and wait for all three services to become healthy.
- [ ] Run `scripts/smoke-api.sh` and `npm run test:e2e` against the rebuilt stack.
- [ ] Inspect desktop and 390px visual screenshots for field leakage, overlap, horizontal overflow, table alignment, and action clarity.
- [ ] Review the final diff for accidental key exposure, unrelated changes, and stale single-config documentation.
