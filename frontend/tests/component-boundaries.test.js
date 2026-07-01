import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const appSource = await readFile(new URL('../src/App.vue', import.meta.url), 'utf8')
const dutySource = await readFile(new URL('../src/components/DutyManagementView.vue', import.meta.url), 'utf8')
const stylesSource = await readFile(new URL('../src/styles.css', import.meta.url), 'utf8')
const dashboardSource = await readFile(new URL('../src/components/DashboardView.vue', import.meta.url), 'utf8')

test('keeps permission and Copilot settings templates in focused components', () => {
  assert.match(appSource, /import PermissionsView from/)
  assert.match(appSource, /<PermissionsView/)
  assert.doesNotMatch(appSource, /<section v-if="activeView === 'permissions'"/)

  assert.match(appSource, /import CopilotSettingsView from/)
  assert.match(appSource, /<CopilotSettingsView/)
  assert.doesNotMatch(appSource, /<section v-if="activeView === 'copilot-settings'"/)
})

test('keeps login and first-password flows in a focused authentication component', () => {
  assert.match(appSource, /import AuthView from/)
  assert.match(appSource, /<AuthView/)
  assert.doesNotMatch(appSource, /<section v-if="!auth\.token" class="auth-screen"/)
  assert.doesNotMatch(appSource, /<section v-else-if="needsInitialPassword"/)
})

test('closes transient duty dialogs on navigation while using the persisted duty API', () => {
  assert.match(appSource, /<DutyManagementView[\s\S]*v-show="activeView === 'oncall'"/)
  assert.match(appSource, /:active="activeView === 'oncall'"/)
  assert.match(dutySource, /active: \{ type: Boolean/)
  assert.match(dutySource, /watch\(\(\) => props\.active/)
  assert.match(dutySource, /closeDutyTransientState/)
  assert.match(dutySource, /api\('\/duty-center'/)
  assert.match(dutySource, /method: 'PUT'/)
  assert.doesNotMatch(dutySource, /张伟|李娜|4\.2min|今日已处理 14/)
})

test('removes retired on-call layouts from the global stylesheet', () => {
  for (const selector of ['.oncall-overview', '.calendar-planner', '.oncall-console', '.duty-topbar', '.duty-shell', '.duty-sidebar', '.duty-alert-card']) {
    assert.doesNotMatch(stylesSource, new RegExp(selector.replace('.', '\\.')))
  }
})

test('exposes every duty modal as an accessible dialog', () => {
  const dialogs = dutySource.match(/role="dialog"/g) || []
  const modalFlags = dutySource.match(/aria-modal="true"/g) || []
  assert.equal(dialogs.length, 6)
  assert.equal(modalFlags.length, 6)
})

test('keeps fixed calendar years out of the duty production interface', () => {
  assert.doesNotMatch(dutySource, /2026-\d{2}-\d{2}/)
})

test('keeps dashboard metric titles, values and supporting data on separate rows', () => {
  assert.equal((dashboardSource.match(/class="metric-copy"/g) || []).length, 4)
  assert.match(stylesSource, /\.metric-copy \{[^}]*display: grid/)
  assert.match(stylesSource, /\.metric-copy strong \{[^}]*display: block/)
})
