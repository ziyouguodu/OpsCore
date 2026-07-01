# OpsCore Confirm Dialog Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace all browser-native dangerous delete confirmations with one OpsCore-styled application confirmation dialog.

**Architecture:** Add a focused Vue component for the modal UI, then keep one global confirmation controller in `frontend/src/App.vue` so existing delete functions can request confirmation without duplicating modal state. Extend the current Playwright UI audit to prove the new dialog opens, cancels, supports Escape, does not trigger `window.confirm`, and can confirm-delete a test asset.

**Tech Stack:** Vue 3 `<script setup>`, Vite, native CSS, Playwright, existing OpsCore REST API client.

---

## File Structure

- Create: `frontend/src/components/ConfirmDialog.vue`
  - Owns only modal presentation, keyboard/backdrop cancellation, focus on confirm button, and emits.
- Modify: `frontend/src/App.vue`
  - Imports `ConfirmDialog`.
  - Adds a single global confirm state and helper functions.
  - Replaces 7 `window.confirm` usages in delete handlers.
  - Renders `<ConfirmDialog />` once near the root template.
- Modify: `frontend/src/styles.css`
  - Adds compact dark confirm dialog styles matching OpsCore.
- Modify: `frontend/e2e/ui-audit.spec.js`
  - Adds helpers and coverage for unified delete confirmation.
- Modify: `AGENTS.md`
  - Records that dangerous deletes must use the unified dialog, not native browser confirm.
- Modify: `WORKLOG.md`
  - Records implementation, verification, and Docker rebuild.

## Task 1: Add Playwright Coverage For Unified Delete Confirmation

**Files:**
- Modify: `frontend/e2e/ui-audit.spec.js`

- [ ] **Step 1: Add helpers near existing helper functions**

Add these helpers after `verifyPager(page)`:

```js
async function expectNoNativeDialog(page) {
  let nativeDialogSeen = false
  page.on('dialog', async dialog => {
    nativeDialogSeen = true
    await dialog.dismiss().catch(() => {})
  })
  return () => nativeDialogSeen
}

async function openConfirmFromFirstDangerAction(page, actionName, expectedTitle) {
  await page.getByRole('button', { name: actionName }).first().click()
  const dialog = page.getByTestId('confirm-dialog')
  await expect(dialog).toBeVisible()
  await expect(dialog.getByRole('heading', { name: expectedTitle })).toBeVisible()
  return dialog
}
```

- [ ] **Step 2: Add cancel, Escape, and confirm-delete assertions in the asset section**

In the asset page block, immediately after `await verifyListDetail(page, '隐藏资产详情')`, add:

```js
    const nativeDialogSeen = await expectNoNativeDialog(page)
    let assetRowsBefore = await page.locator('tbody tr.clickable-row').count()
    await openConfirmFromFirstDangerAction(page, '删除', '删除资产')
    await page.getByTestId('confirm-cancel').click()
    await expect(page.getByTestId('confirm-dialog')).toBeHidden()
    expect(nativeDialogSeen()).toBe(false)

    await openConfirmFromFirstDangerAction(page, '删除', '删除资产')
    await page.keyboard.press('Escape')
    await expect(page.getByTestId('confirm-dialog')).toBeHidden()
    expect(nativeDialogSeen()).toBe(false)

    await openConfirmFromFirstDangerAction(page, '删除', '删除资产')
    await page.getByTestId('confirm-accept').click()
    await expect(page.getByTestId('confirm-dialog')).toBeHidden()
    await expect.poll(async () => page.locator('tbody tr.clickable-row').count()).toBeLessThan(assetRowsBefore)
```

- [ ] **Step 3: Run E2E and verify it fails before implementation**

Run:

```bash
cd frontend
ADMIN_PASSWORD='OpsCore2026' npm run test:e2e
```

Expected: FAIL because `data-testid="confirm-dialog"` does not exist and native `window.confirm` still handles delete.

## Task 2: Create The Reusable Confirm Dialog Component

**Files:**
- Create: `frontend/src/components/ConfirmDialog.vue`
- Modify: `frontend/src/styles.css`

- [ ] **Step 1: Create `frontend/src/components/ConfirmDialog.vue`**

Use this full component:

```vue
<script setup>
import { nextTick, ref, watch, onBeforeUnmount } from 'vue'

const props = defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, default: '确认操作' },
  message: { type: String, default: '' },
  target: { type: String, default: '' },
  confirmLabel: { type: String, default: '确认删除' },
  busy: { type: Boolean, default: false }
})

const emit = defineEmits(['confirm', 'cancel'])
const confirmButton = ref(null)

function cancel() {
  if (props.busy) return
  emit('cancel')
}

function confirm() {
  if (props.busy) return
  emit('confirm')
}

function onKeydown(event) {
  if (event.key === 'Escape' && props.open && !props.busy) {
    event.preventDefault()
    cancel()
  }
}

watch(
  () => props.open,
  async isOpen => {
    if (isOpen) {
      window.addEventListener('keydown', onKeydown)
      await nextTick()
      confirmButton.value?.focus()
    } else {
      window.removeEventListener('keydown', onKeydown)
    }
  }
)

onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="confirm-backdrop" @click.self="cancel">
      <section
        class="confirm-dialog"
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="confirm-title"
        aria-describedby="confirm-description"
        data-testid="confirm-dialog"
      >
        <header class="confirm-head">
          <span class="confirm-icon" aria-hidden="true">!</span>
          <div>
            <p class="confirm-kicker">危险操作</p>
            <h2 id="confirm-title">{{ title }}</h2>
          </div>
          <button
            class="icon-button confirm-close"
            type="button"
            aria-label="关闭确认弹窗"
            :disabled="busy"
            @click="cancel"
          >
            ×
          </button>
        </header>

        <div class="confirm-body">
          <p id="confirm-description">{{ message }}</p>
          <div v-if="target" class="confirm-target">
            <span>操作对象</span>
            <strong>{{ target }}</strong>
          </div>
        </div>

        <footer class="confirm-actions">
          <button type="button" class="ghost-button" :disabled="busy" data-testid="confirm-cancel" @click="cancel">取消</button>
          <button
            ref="confirmButton"
            type="button"
            class="danger-button"
            :disabled="busy"
            data-testid="confirm-accept"
            @click="confirm"
          >
            {{ busy ? '处理中...' : confirmLabel }}
          </button>
        </footer>
      </section>
    </div>
  </Teleport>
</template>
```

- [ ] **Step 2: Add styles to `frontend/src/styles.css`**

Append this block near existing modal styles:

```css
.confirm-backdrop {
  position: fixed;
  inset: 0;
  z-index: 80;
  display: grid;
  place-items: center;
  padding: 24px;
  background: rgba(2, 8, 23, 0.72);
  backdrop-filter: blur(8px);
}

.confirm-dialog {
  width: min(420px, 100%);
  border: 1px solid rgba(248, 113, 113, 0.42);
  border-radius: 12px;
  background: #071221;
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.46), 0 0 0 1px rgba(148, 163, 184, 0.08) inset;
  color: #e5edf8;
  overflow: hidden;
}

.confirm-head {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 14px;
  align-items: start;
  padding: 20px 20px 14px;
}

.confirm-icon {
  display: inline-grid;
  width: 34px;
  height: 34px;
  place-items: center;
  border-radius: 10px;
  border: 1px solid rgba(248, 113, 113, 0.54);
  background: rgba(127, 29, 29, 0.28);
  color: #fecaca;
  font-weight: 800;
}

.confirm-kicker {
  margin: 0 0 3px;
  color: #fca5a5;
  font-size: 12px;
  font-weight: 700;
}

.confirm-head h2 {
  margin: 0;
  font-size: 20px;
  line-height: 1.25;
}

.confirm-close {
  width: 34px;
  height: 34px;
}

.confirm-body {
  padding: 0 20px 18px;
}

.confirm-body p {
  margin: 0;
  color: #a8b6cc;
  line-height: 1.7;
}

.confirm-target {
  margin-top: 14px;
  padding: 12px;
  border-radius: 10px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  background: rgba(15, 23, 42, 0.72);
}

.confirm-target span {
  display: block;
  margin-bottom: 4px;
  color: #7dd3fc;
  font-size: 12px;
}

.confirm-target strong {
  display: block;
  color: #f8fafc;
  font-size: 14px;
  line-height: 1.5;
  word-break: break-word;
}

.confirm-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 16px 20px 20px;
  border-top: 1px solid rgba(148, 163, 184, 0.14);
}

.danger-button {
  border: 1px solid rgba(248, 113, 113, 0.52);
  border-radius: 10px;
  padding: 10px 16px;
  background: linear-gradient(135deg, #dc2626, #b91c1c);
  color: #fff;
  font-weight: 800;
  cursor: pointer;
}

.danger-button:disabled {
  cursor: not-allowed;
  opacity: 0.7;
}

@media (max-width: 640px) {
  .confirm-actions {
    flex-direction: column-reverse;
  }

  .confirm-actions button {
    width: 100%;
  }
}
```

- [ ] **Step 3: Run frontend build to catch component syntax errors**

Run:

```bash
cd frontend
npm run build
```

Expected: PASS after `App.vue` imports/renders the component in Task 3. If run before Task 3, the new component can exist unused and the build should still pass.

## Task 3: Wire The Dialog Into `App.vue` And Replace Native Confirms

**Files:**
- Modify: `frontend/src/App.vue`

- [ ] **Step 1: Import the component**

At the top of `frontend/src/App.vue`, change imports to include:

```js
import ConfirmDialog from './components/ConfirmDialog.vue'
```

- [ ] **Step 2: Add global confirmation state after the existing `error` ref**

Add:

```js
const confirmState = reactive({
  open: false,
  title: '',
  message: '',
  target: '',
  confirmLabel: '确认删除',
  busy: false,
  action: null,
  resolver: null,
  trigger: null
})

function resetConfirmState() {
  const trigger = confirmState.trigger
  confirmState.open = false
  confirmState.title = ''
  confirmState.message = ''
  confirmState.target = ''
  confirmState.confirmLabel = '确认删除'
  confirmState.busy = false
  confirmState.action = null
  confirmState.resolver = null
  confirmState.trigger = null
  trigger?.focus?.()
}

function cancelConfirm() {
  if (confirmState.busy) return
  confirmState.resolver?.(false)
  resetConfirmState()
}

function requestConfirm(options, action) {
  if (confirmState.open) cancelConfirm()
  confirmState.title = options.title
  confirmState.message = options.message
  confirmState.target = options.target || ''
  confirmState.confirmLabel = options.confirmLabel || '确认删除'
  confirmState.action = action
  confirmState.trigger = document.activeElement
  confirmState.open = true
  return new Promise(resolve => {
    confirmState.resolver = resolve
  })
}

async function acceptConfirm() {
  if (confirmState.busy) return
  confirmState.busy = true
  try {
    await confirmState.action?.()
    confirmState.resolver?.(true)
  } catch (err) {
    error.value = err.message || '操作失败'
    confirmState.resolver?.(false)
  } finally {
    resetConfirmState()
  }
}
```

- [ ] **Step 3: Replace each delete function guard**

Change `deleteAsset(asset)` to:

```js
async function deleteAsset(asset) {
  if (!asset) return
  await requestConfirm({
    title: '删除资产',
    message: '删除后无法恢复，关联凭据也将一并清理。',
    target: `${asset.assetNo || asset.id} · ${asset.business || asset.hostname || asset.owner || '未命名资产'}`,
    confirmLabel: '确认删除资产'
  }, async () => {
    loading.value = true
    try {
      await api.deleteAsset(asset.id)
      state.assets = state.assets.filter(item => item.id !== asset.id)
      if (selectedAsset.value?.id === asset.id) selectedAsset.value = null
      closeAssetForm()
      await loadDashboard()
    } catch (err) {
      error.value = `删除资产失败：${err.message}`
    } finally {
      loading.value = false
    }
  })
}
```

Change `deleteMiddleware(item)` to:

```js
async function deleteMiddleware(item) {
  if (!item) return
  await requestConfirm({
    title: '删除实例',
    message: '删除后无法恢复，关联凭据也将一并清理。',
    target: `${item.name || item.id} · ${item.kind || '中间件与数据库实例'}`,
    confirmLabel: '确认删除实例'
  }, async () => {
    loading.value = true
    try {
      await api.deleteMiddleware(item.id)
      state.middleware = state.middleware.filter(entry => entry.id !== item.id)
      if (selectedMiddleware.value?.id === item.id) selectedMiddleware.value = null
      closeMiddlewareForm()
      await loadDashboard()
    } catch (err) {
      error.value = `删除实例失败：${err.message}`
    } finally {
      loading.value = false
    }
  })
}
```

Change `deleteTask(task)` to:

```js
async function deleteTask(task) {
  if (!task) return
  await requestConfirm({
    title: '删除任务',
    message: '删除后无法恢复，任务处理记录将从列表中移除。',
    target: `${task.title || task.id} · ${task.status || '未知状态'}`,
    confirmLabel: '确认删除任务'
  }, async () => {
    loading.value = true
    try {
      await api.deleteTask(task.id)
      state.tasks = state.tasks.filter(item => item.id !== task.id)
      if (selectedTask.value?.id === task.id) selectedTask.value = null
      closeTaskForm()
      await loadDashboard()
    } catch (err) {
      error.value = `删除任务失败：${err.message}`
    } finally {
      loading.value = false
    }
  })
}
```

Change `deleteIncident(incident)` to:

```js
async function deleteIncident(incident) {
  if (!incident) return
  await requestConfirm({
    title: '删除事件',
    message: '删除后无法恢复，事件跟进和闭环信息将从列表中移除。',
    target: `${incident.title || incident.id} · ${incident.level || 'P3'}`,
    confirmLabel: '确认删除事件'
  }, async () => {
    loading.value = true
    try {
      await api.deleteIncident(incident.id)
      state.incidents = state.incidents.filter(item => item.id !== incident.id)
      if (selectedIncident.value?.id === incident.id) selectedIncident.value = null
      closeIncidentForm()
      await loadDashboard()
    } catch (err) {
      error.value = `删除事件失败：${err.message}`
    } finally {
      loading.value = false
    }
  })
}
```

Change `deleteOncall(item)` to:

```js
async function deleteOncall(item) {
  if (!canManageOncall.value) {
    error.value = '当前角色只能查看值班安排，不能删除值班记录'
    return
  }
  if (!item) return
  await requestConfirm({
    title: '删除值班记录',
    message: '删除后无法恢复，该值班安排将从当前列表中移除。',
    target: `${item.date || item.week || item.id} · ${item.primary || '未指定主值'}`,
    confirmLabel: '确认删除值班'
  }, async () => {
    loading.value = true
    try {
      await api.deleteOncall(item.id)
      state.oncalls = state.oncalls.filter(entry => entry.id !== item.id)
      closeOncallForm()
      await loadDashboard()
    } catch (err) {
      error.value = `删除值班失败：${err.message}`
    } finally {
      loading.value = false
    }
  })
}
```

Change `deleteDutySchedule(schedule)` to:

```js
function deleteDutySchedule(schedule) {
  if (!schedule) return
  requestConfirm({
    title: '删除排班模板',
    message: '删除后该排班规则不再用于后续值班生成。',
    target: `${schedule.team || '未命名团队'} · ${schedule.cycle || '未设置规则'}`,
    confirmLabel: '确认删除模板'
  }, async () => {
    dutySchedules.value = dutySchedules.value.filter(item => item.id !== schedule.id)
    pushDutyToast('排班模板已删除', 'warning')
  })
}
```

Change `deleteUser(user)` to:

```js
async function deleteUser(user) {
  if (!canManageUsers.value) {
    error.value = '当前角色无权删除用户'
    return
  }
  if (user?.id === currentUserID()) {
    error.value = '不能删除当前登录用户'
    return
  }
  if (!user) return
  await requestConfirm({
    title: '删除用户',
    message: '删除后该账号将无法登录，相关角色关系也会被移除。',
    target: `${user.username} · ${user.displayName || '未设置姓名'}`,
    confirmLabel: '确认删除用户'
  }, async () => {
    loading.value = true
    try {
      await api.deleteUser(user.id)
      state.users = state.users.filter(item => item.id !== user.id)
      closeUserForm()
    } catch (err) {
      error.value = `删除用户失败：${err.message}`
    } finally {
      loading.value = false
    }
  })
}
```

- [ ] **Step 4: Render the dialog once in the template**

Place this near other global overlays, before the closing root element:

```vue
    <ConfirmDialog
      :open="confirmState.open"
      :title="confirmState.title"
      :message="confirmState.message"
      :target="confirmState.target"
      :confirm-label="confirmState.confirmLabel"
      :busy="confirmState.busy"
      @cancel="cancelConfirm"
      @confirm="acceptConfirm"
    />
```

- [ ] **Step 5: Verify no native confirms remain**

Run:

```bash
rg -n "window\\.confirm" frontend/src/App.vue
```

Expected: no output.

## Task 4: Documentation And Verification

**Files:**
- Modify: `AGENTS.md`
- Modify: `WORKLOG.md`

- [ ] **Step 1: Update `AGENTS.md` frontend convention**

Add this bullet under the frontend conventions:

```markdown
- 危险删除操作必须使用统一应用内确认弹窗，不得使用浏览器原生 `window.confirm`；确认弹窗需要支持取消、关闭、遮罩、Escape、处理中禁用和焦点返回。
```

- [ ] **Step 2: Update `WORKLOG.md`**

Add a dated entry:

```markdown
## 2026-06-26

- 统一危险操作确认弹窗：新增 `ConfirmDialog.vue`，替换资产、实例、任务、事件、值班记录、排班模板和用户删除中的原生确认框。
- 交互验证：补充 Playwright 覆盖应用内确认弹窗的取消、Escape、确认删除和无原生弹窗行为。
- 验证：`npm run build`、`ADMIN_PASSWORD='OpsCore2026' npm run test:e2e`、`GOCACHE=/Users/mac/Desktop/work/OpsCore/.cache/go-build go test ./...`、`ADMIN_PASSWORD='OpsCore2026' scripts/smoke-api.sh`、`cd deploy && docker compose up --build -d`。
```

- [ ] **Step 3: Run frontend checks**

Run:

```bash
cd frontend
npm run build
ADMIN_PASSWORD='OpsCore2026' npm run test:e2e
```

Expected: both PASS.

- [ ] **Step 4: Run backend and smoke checks**

Run:

```bash
GOCACHE=/Users/mac/Desktop/work/OpsCore/.cache/go-build go test ./...
ADMIN_PASSWORD='OpsCore2026' scripts/smoke-api.sh
```

Expected: backend tests PASS, smoke script PASS.

- [ ] **Step 5: Rebuild and keep Docker Compose stack running**

Run:

```bash
cd deploy
docker compose up --build -d
```

Expected: frontend, backend, and PostgreSQL containers are healthy/running.

## Self-Review

- Spec coverage: The plan covers component creation, visual styling, 7 delete call sites, busy/cancel behavior, E2E coverage, docs, and Docker rebuild.
- Placeholder scan: No `TBD`, `TODO`, `implement later`, or unspecified tests remain.
- Type consistency: The plan uses `confirmState`, `requestConfirm`, `acceptConfirm`, `cancelConfirm`, and the component props/events consistently across tasks.
