import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const appSource = await readFile(new URL('../src/App.vue', import.meta.url), 'utf8')
const dutySource = await readFile(new URL('../src/components/DutyManagementView.vue', import.meta.url), 'utf8')
const stylesSource = await readFile(new URL('../src/styles.css', import.meta.url), 'utf8')
const dashboardSource = await readFile(new URL('../src/components/DashboardView.vue', import.meta.url), 'utf8')
const copilotSource = await readFile(new URL('../src/components/CopilotWidget.vue', import.meta.url), 'utf8')
const copilotComposableSource = await readFile(new URL('../src/composables/useCopilotChat.js', import.meta.url), 'utf8')
const toastSource = await readFile(new URL('../src/components/ToastStack.vue', import.meta.url), 'utf8')
const routeComposableSource = await readFile(new URL('../src/composables/useWorkspaceRoute.js', import.meta.url), 'utf8')
const navigationSource = await readFile(new URL('../src/navigation.js', import.meta.url), 'utf8')
const topbarSource = await readFile(new URL('../src/components/Topbar.vue', import.meta.url), 'utf8')
const pageHeaderSource = await readFile(new URL('../src/components/WorkspacePageHeader.vue', import.meta.url), 'utf8')
const workspaceComponents = [
  ['AssetView', 'cmdb'], ['MiddlewareView', 'middleware'], ['DutyManagementView', 'oncall'],
  ['TaskView', 'tasks'], ['IncidentView', 'incidents'], ['PermissionsView', 'permissions'],
  ['AuditView', 'audit'], ['CopilotSettingsView', 'copilot-settings']
]
const workspaceSources = await Promise.all(workspaceComponents.map(async ([component, view]) => ({
  component, view, source: await readFile(new URL(`../src/components/${component}.vue`, import.meta.url), 'utf8')
})))
let dashboardComposableSource = ''
try {
  dashboardComposableSource = await readFile(new URL('../src/composables/useDashboardViewModel.js', import.meta.url), 'utf8')
} catch {}

test('keeps each business page header directly inside its workspace and out of the topbar', () => {
  assert.doesNotMatch(topbarSource, /<h1\b|workspace-page-header|class="breadcrumb"/)
  assert.match(pageHeaderSource, /<header class="workspace-page-header">/)
  assert.equal((pageHeaderSource.match(/<h1\b/g) || []).length, 1)
  assert.equal((pageHeaderSource.match(/<p\b/g) || []).length, 1)
  assert.equal((pageHeaderSource.match(/class="breadcrumb"/g) || []).length, 1)
  for (const { component, view, source } of workspaceSources) {
    assert.equal((source.match(/<WorkspacePageHeader\b/g) || []).length, 1, component)
    assert.match(source, new RegExp(`<section[^>]*class="(?:panel|duty-center)[^"]*"[^>]*>\\s*<WorkspacePageHeader[^>]*view="${view}"`), component)
    assert.doesNotMatch(source, /<h1\b/, component)
  }
  assert.match(dutySource, /<WorkspacePageHeader v-if="active" view="oncall"/)
  assert.equal((dashboardSource.match(/<h1\b/g) || []).length, 1)
  assert.match(dashboardSource, /<h1>\{\{ greeting\(now\) \}\}，\{\{ username \}\}/)
  assert.doesNotMatch(dashboardSource, /WorkspacePageHeader|class="brand-line"|<footer\b|class="footer"/)
})

test('keeps permission and Copilot settings templates in focused components', () => {
  assert.match(appSource, /import PermissionsView from/)
  assert.match(appSource, /<PermissionsView/)
  assert.doesNotMatch(appSource, /<section v-if="activeView === 'permissions'"/)

  assert.match(appSource, /import CopilotSettingsView from/)
  assert.match(appSource, /<CopilotSettingsView/)
  assert.doesNotMatch(appSource, /<section v-if="activeView === 'copilot-settings'"/)
})

test('does not duplicate permission sidebar navigation inside the workspace', () => {
  const permissions = workspaceSources.find(item => item.component === 'PermissionsView').source
  assert.doesNotMatch(permissions, /segmented-tabs|navigate-tab/)
  assert.doesNotMatch(appSource, /@navigate-tab=/)
  assert.match(permissions, /permissionTab === 'users'/)
  for (const tab of ['users', 'resources']) assert.match(navigationSource, new RegExp(`permissionTab: '${tab}'`))
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
  assert.equal(dialogs.length, 7)
  assert.equal(modalFlags.length, 7)
  assert.match(dutySource, /<DutyDialog/)
  assert.match(dutySource, /role="tablist"/)
  assert.match(dutySource, /role="tabpanel"/)
})

test('keeps fixed calendar years out of the duty production interface', () => {
  assert.doesNotMatch(dutySource, /2026-\d{2}-\d{2}/)
})

test('keeps exactly six simple dashboard KPIs without secondary metric cards', () => {
  assert.match(dashboardSource, /aria-label="核心指标"/)
  assert.equal((dashboardSource.match(/class="kpi glass/g) || []).length, 6)
  assert.doesNotMatch(dashboardSource, /metric-copy|secondary-metrics|asset-bars|incident-levels/)
})

test('keeps model-backed Copilot orchestration outside the application shell', () => {
  assert.match(appSource, /useCopilotChat/)
  assert.match(copilotComposableSource, /api\('\/copilot\/chat'/)
  assert.doesNotMatch(appSource, /function answerCopilot/)
  assert.match(copilotSource, /role="dialog"/)
  assert.match(copilotSource, /aria-live="polite"/)
})

test('uses semantic toast feedback for completed mutations', () => {
  assert.match(appSource, /<ToastStack/)
  assert.match(toastSource, /aria-live="polite"/)
  assert.match(toastSource, /role="status"/)
})

test('keeps hash routing and scroll restoration outside the application shell', () => {
  assert.match(appSource, /useWorkspaceRoute/)
  assert.match(routeComposableSource, /hashchange/)
  assert.match(routeComposableSource, /window\.scrollTo/)
  assert.doesNotMatch(appSource, /function parseRouteHash/)
})

test('keeps dashboard presentation orchestration outside the application shell', () => {
  assert.match(appSource, /useDashboardViewModel/)
  assert.match(dashboardComposableSource, /dashboardPriorityItems/)
  assert.doesNotMatch(appSource, /const dashboardPriorityItems = computed/)
})

test('keeps the duty workspace grid compact and the navigation free of retired entries', () => {
  assert.match(stylesSource, /\.duty-center \{[^}]*align-content: start/)
  assert.doesNotMatch(stylesSource, /\.duty-center \{[^}]*min-height: 720px/)
  assert.match(stylesSource, /\.duty-tabbar \{[^}]*height: max-content/)
  assert.match(dashboardSource, /class="[^"]*\bduty-open\b[^"]*"/)
  assert.match(dashboardSource, /备班 未安排/)
  assert.doesNotMatch(stylesSource, /\.duty-tabbar[^{]*\{[^}]*overflow: auto/)
})

test('keeps the retired task entry out of the sidebar menu while the page stays reachable', () => {
  assert.doesNotMatch(navigationSource, /id: 'tasks'/)
  assert.match(navigationSource, /'tasks'/)
  assert.match(appSource, /activeView === 'tasks'/)
})

test('keeps the production dashboard aligned with the reviewed command-center boundary', () => {
  assert.match(dashboardSource, /业务系统健康趋势/)
  assert.match(dashboardSource, /核心服务可用性/)
  assert.match(dashboardSource, /今日/)
  assert.match(dashboardSource, /近 7 天/)
  assert.match(dashboardSource, /近 30 天/)
  assert.match(dashboardSource, /自定义/)
  assert.match(dashboardSource, /资产状态健康率单独呈现/)
  assert.match(dashboardSource, /告警中心与监控数据待接入/)
  assert.match(appSource, /:workspace="dashboardWorkspace"/)
  assert.match(dashboardSource, /review-dashboard/)
})
