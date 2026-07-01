<script setup>
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { api, clearToken, getToken, login as loginApi, setSessionExpiredHandler } from './api'
import { copilotProviders, menuPermissionRows, permissionRows, roleCards } from './app-config'
import AssetView from './components/AssetView.vue'
import AuthView from './components/AuthView.vue'
import CopilotSettingsView from './components/CopilotSettingsView.vue'
import CopilotWidget from './components/CopilotWidget.vue'
import ConfirmDialog from './components/ConfirmDialog.vue'
import DashboardView from './components/DashboardView.vue'
import DutyManagementView from './components/DutyManagementView.vue'
import IncidentView from './components/IncidentView.vue'
import MiddlewareView from './components/MiddlewareView.vue'
import PermissionsView from './components/PermissionsView.vue'
import SidebarNav from './components/SidebarNav.vue'
import TaskView from './components/TaskView.vue'
import Topbar from './components/Topbar.vue'
import { useConfirmDialog } from './composables/useConfirmDialog'
import { summarizeDashboardMetrics } from './dashboard-metrics'
import { controlPrinciple, permissionTabs, routeViews, viewMeta } from './navigation'
import { sampleAssets, sampleIncidents, sampleMiddleware, sampleOncalls, sampleTasks, sampleUsers } from './sample-data'

const demoDataEnabled = import.meta.env.VITE_ENABLE_DEMO_DATA === 'true'

function parseRouteHash() {
  const raw = window.location.hash.replace(/^#/, '')
  const [view, tab] = raw.split('/')
  return {
    view: routeViews.includes(view) ? view : 'dashboard',
    permissionTab: permissionTabs.includes(tab) ? tab : 'users'
  }
}

const initialRoute = parseRouteHash()
const activeView = ref(initialRoute.view)
const permissionTab = ref(initialRoute.permissionTab)
const copilotOpen = ref(false)
const copilotExpanded = ref(false)
const sidebarCollapsed = ref(true)
const mobileNavOpen = ref(false)
const loading = ref(false)
const error = ref('')
const { confirmState, requestConfirm, cancelConfirm, acceptConfirm } = useConfirmDialog((err) => {
  error.value = err.message || '操作失败'
})
const selectedAsset = ref(null)
const selectedMiddleware = ref(null)
const selectedTask = ref(null)
const selectedIncident = ref(null)
const assetFormOpen = ref(false)
const middlewareFormOpen = ref(false)
const taskFormOpen = ref(false)
const incidentFormOpen = ref(false)
const userFormOpen = ref(false)
const credential = reactive({ loginUrl: '', username: '', secret: '', hasSecret: false, notes: '' })
const credentialReveal = reactive({ password: '', revealed: false })
const credentialMessage = ref('')
const assetFormCredential = reactive({ loginUrl: '', username: '', secret: '', notes: '' })
const middlewareCredential = reactive({ loginUrl: '', username: '', secret: '', hasSecret: false, notes: '' })
const middlewareCredentialReveal = reactive({ password: '', revealed: false })
const middlewareCredentialMessage = ref('')
const middlewareFormCredential = reactive({ loginUrl: '', username: '', secret: '', notes: '' })
const credentialVerification = reactive({ hasPassword: false, password: '', confirm: '', message: '' })
const auth = reactive({
  token: getToken(),
  user: null,
  username: '',
  password: ''
})
const passwordInit = reactive({
  currentPassword: '',
  newPassword: '',
  confirmPassword: ''
})
const hasAppAccess = computed(() => Boolean(auth.token && auth.user && !auth.user.mustChangePassword))
const needsInitialPassword = computed(() => Boolean(auth.token && auth.user?.mustChangePassword))
const authPending = computed(() => Boolean(auth.token && !auth.user))
const copilotQuestion = ref('')
const copilotMessages = ref([
  { role: 'ai', text: '我可以查询资产、活跃事件、今日值班和待处理任务，并给出处置建议。' }
])

const emptyDashboard = {
  assetCount: 0,
  todayOnCallCount: 0,
  activeTaskCount: 0,
  activeIncidentCount: 0,
  assetTypeCounts: {},
  incidentLevelCounts: {}
}

const state = reactive({
  dashboard: { ...emptyDashboard },
  assets: [],
  middleware: [],
  oncalls: [],
  tasks: [],
  incidents: [],
  users: []
})


const newAsset = reactive({
  id: null,
  assetNo: '',
  type: '物理机',
  vendor: '',
  cpuArch: 'x86_64',
  sn: '',
  location: '',
  business: '',
  ipv4: '',
  ipv6: '',
  environment: '生产',
  os: '',
  hostname: '',
  networkZone: '',
  cpu: '',
  memory: '',
  disk: '',
  deploymentInfo: '',
  owner: '',
  status: '运行中',
  hostMachine: ''
})

const newMiddleware = reactive({
  id: null,
  name: '',
  kind: 'MySQL',
  version: '',
  environment: '生产',
  networkZone: '',
  endpoint: '',
  business: '',
  owner: '',
  status: '运行中',
  assetId: ''
})

const newTask = reactive({ id: null, title: '', type: '任务', assignee: '', status: '待处理', dueAt: '', description: '' })
const newIncident = reactive({ id: null, title: '', level: 'P3', status: '新建', owner: '', business: '', startedAt: '', recoveredAt: '', summary: '' })
const newUser = reactive({ id: null, username: '', displayName: '', password: '', mustChangePassword: true, role: 'ops_engineer' })
const copilotConfig = reactive({
  provider: 'openai',
  model: 'gpt-4.1',
  endpoint: 'https://api.openai.com/v1',
  localEndpoint: 'http://host.docker.internal:11434',
  localModel: 'qwen2.5:7b',
  apiKey: '',
  hasApiKey: false,
  temperature: '0.2',
  maxTokens: '4096',
  enableAssetContext: true,
  enableIncidentContext: true,
  enableTaskContext: true,
  enableOncallContext: true,
  auditEnabled: true
})
const copilotConnection = reactive({
  testing: false,
  ok: null,
  message: '',
  latencyMs: null,
  statusCode: null
})
const assetFilters = reactive({ keyword: '', type: '', environment: '', business: '', networkZone: '', advanced: false })
const middlewareFilters = reactive({ keyword: '', kind: '', environment: '', business: '', networkZone: '', status: '', advanced: false })
const assetPager = reactive({ page: 1, pageSize: 10 })
const middlewarePager = reactive({ page: 1, pageSize: 10 })
const taskPager = reactive({ page: 1, pageSize: 10 })
const incidentPager = reactive({ page: 1, pageSize: 10 })

const activeTitle = computed(() => viewMeta[activeView.value]?.title || '首页健康总览')
const activeBreadcrumb = computed(() => viewMeta[activeView.value]?.breadcrumb || '工作台')
const activeSubtitle = computed(() => viewMeta[activeView.value]?.subtitle || controlPrinciple)

const canManageCredentials = computed(() => {
  const roles = auth.user?.roles || auth.user?.Roles || []
  return roles.includes('super_admin')
})
const canWriteAssets = computed(() => {
  const roles = auth.user?.roles || auth.user?.Roles || []
  return roles.includes('super_admin') || roles.includes('ops_engineer')
})
const canWriteOncall = computed(() => {
  const roles = auth.user?.roles || auth.user?.Roles || []
  return roles.includes('super_admin')
})
const canManageUsers = computed(() => {
  const roles = auth.user?.roles || auth.user?.Roles || []
  return roles.includes('super_admin')
})
function isSuperAdmin() {
  const roles = auth.user?.roles || auth.user?.Roles || []
  return roles.includes('super_admin')
}

function currentUserID() {
  return auth.user?.id || auth.user?.uid || auth.user?.userID || auth.user?.UserID || 0
}

function canDeleteAsset(asset) {
  if (!asset) return false
  if (isSampleRecord(asset)) return false
  return isSuperAdmin() || (asset.createdBy && asset.createdBy === currentUserID())
}

function isSampleRecord(item) {
  return Boolean(item?.__sample)
}

function countBy(items, field, defaults = []) {
  const counts = Object.fromEntries(defaults.map((item) => [item, 0]))
  for (const item of items) {
    const key = item[field] || '未设置'
    counts[key] = (counts[key] || 0) + 1
  }
  return counts
}

const displayAssets = computed(() => state.assets.length ? state.assets : (demoDataEnabled ? sampleAssets : []))
const displayMiddleware = computed(() => state.middleware.length ? state.middleware : (demoDataEnabled ? sampleMiddleware : []))
const displayOncalls = computed(() => state.oncalls.length ? state.oncalls : (demoDataEnabled ? sampleOncalls : []))
const displayTasks = computed(() => state.tasks.length ? state.tasks : (demoDataEnabled ? sampleTasks : []))
const displayIncidents = computed(() => state.incidents.length ? state.incidents : (demoDataEnabled ? sampleIncidents : []))
const displayUsers = computed(() => state.users.length ? state.users : (demoDataEnabled ? sampleUsers : []))
const dashboardMetrics = computed(() => summarizeDashboardMetrics({
  assets: displayAssets.value,
  middleware: displayMiddleware.value,
  tasks: displayTasks.value,
  incidents: displayIncidents.value
}))
const taskStatusCounts = computed(() => countBy(displayTasks.value, 'status', ['待处理', '处理中', '待确认', '已完成', '已关闭']))
const incidentLevelCounts = computed(() => countBy(displayIncidents.value, 'level', ['P1', 'P2', 'P3', 'P4']))
const activeTasks = computed(() => displayTasks.value.filter((item) => !['已完成', '已关闭'].includes(item.status)))
const activeIncidents = computed(() => displayIncidents.value.filter((item) => item.status !== '已关闭'))
const todayOncall = computed(() => displayOncalls.value[0] || {})
const currentTask = computed(() => selectedTask.value || null)
const currentIncident = computed(() => selectedIncident.value || null)
const dashboardAssetCount = computed(() => {
  if (demoDataEnabled && !state.assets.length && !state.middleware.length && state.dashboard.assetCount === 0) {
    return displayAssets.value.length + displayMiddleware.value.length
  }
  return state.dashboard.assetCount
})
const dashboardOncallCount = computed(() => {
  if (demoDataEnabled && !state.oncalls.length && state.dashboard.todayOnCallCount === 0) {
    return displayOncalls.value.length
  }
  return state.dashboard.todayOnCallCount
})
const dashboardTaskCount = computed(() => {
  if (demoDataEnabled && !state.tasks.length && state.dashboard.activeTaskCount === 0) {
    return activeTasks.value.length
  }
  return state.dashboard.activeTaskCount
})
const dashboardIncidentCount = computed(() => {
  if (demoDataEnabled && !state.incidents.length && state.dashboard.activeIncidentCount === 0) {
    return activeIncidents.value.length
  }
  return state.dashboard.activeIncidentCount
})
const assetKpiBars = computed(() => {
  const databaseKinds = ['MySQL', 'PostgreSQL', '达梦']
  const serverCount = displayAssets.value.length
  const databaseCount = displayMiddleware.value.filter((item) => databaseKinds.includes(item.kind)).length
  const middlewareCount = Math.max(displayMiddleware.value.length - databaseCount, 0)
  const items = [
    { label: '服务器', value: serverCount, tone: 'server' },
    { label: '数据库', value: databaseCount, tone: 'database' },
    { label: '中间件', value: middlewareCount, tone: 'middleware' }
  ]
  const max = Math.max(...items.map((item) => item.value), 1)
  return items.map((item) => ({ ...item, height: `${Math.max(24, Math.round((item.value / max) * 78))}%` }))
})
const taskKpiSegments = computed(() => [
  { label: '待处理', value: taskStatusCounts.value['待处理'] || 0, color: '#f59e0b' },
  { label: '处理中', value: taskStatusCounts.value['处理中'] || 0, color: '#2563eb' },
  { label: '待确认', value: taskStatusCounts.value['待确认'] || 0, color: '#14b8a6' }
])
const taskKpiCards = computed(() => {
  const total = Math.max(taskKpiSegments.value.reduce((sum, item) => sum + item.value, 0), 1)
  return taskKpiSegments.value.map((item) => ({
    ...item,
    width: `${Math.max(10, Math.round((item.value / total) * 100))}%`
  }))
})
const activeIncidentLevelCounts = computed(() => countBy(activeIncidents.value, 'level', ['P1', 'P2', 'P3', 'P4']))
const incidentKpiLevels = computed(() => [
  { label: 'P1', value: activeIncidentLevelCounts.value.P1 || 0, desc: '高危', tone: 'p1' },
  { label: 'P2', value: activeIncidentLevelCounts.value.P2 || 0, desc: '重要', tone: 'p2' },
  { label: 'P3', value: activeIncidentLevelCounts.value.P3 || 0, desc: '一般', tone: 'p3' },
  { label: 'P4', value: activeIncidentLevelCounts.value.P4 || 0, desc: '观察', tone: 'p4' }
])
const dashboardPriorityItems = computed(() => {
  const incidentScore = { P1: 1, P2: 2, P3: 3, P4: 4 }
  const taskScore = { 处理中: 5, 待处理: 6, 待确认: 7, 已完成: 8, 已关闭: 9 }
  const incidents = activeIncidents.value.map((item) => ({
    id: `incident-${item.id}`,
    source: item,
    target: 'incidents',
    type: '事件',
    title: item.title,
    owner: item.owner || '未指定',
    status: item.status,
    badge: item.level,
    meta: item.business || item.startedAt || '影响范围待确认',
    score: incidentScore[item.level] || 4
  }))
  const tasks = activeTasks.value.map((item) => ({
    id: `task-${item.id}`,
    source: item,
    target: 'tasks',
    type: '任务',
    title: item.title,
    owner: item.assignee || '未指定',
    status: item.status,
    badge: item.dueAt || item.status,
    meta: item.description || '待补充说明',
    score: taskScore[item.status] || 9
  }))
  return [...incidents, ...tasks].sort((a, b) => a.score - b.score).slice(0, 6)
})
const assetBusinesses = computed(() => uniqueOptions(displayAssets.value, 'business'))
const assetNetworkZones = computed(() => uniqueOptions(displayAssets.value, 'networkZone'))
const middlewareBusinesses = computed(() => uniqueOptions(displayMiddleware.value, 'business'))
const middlewareNetworkZones = computed(() => uniqueOptions(displayMiddleware.value, 'networkZone'))
const filteredAssets = computed(() => filterRows(displayAssets.value, assetFilters, ['assetNo', 'business', 'ipv4', 'ipv6', 'owner', 'deploymentInfo']))
const filteredMiddleware = computed(() => filterRows(displayMiddleware.value, middlewareFilters, ['name', 'kind', 'endpoint', 'business', 'owner']))
const assetPageCount = computed(() => pageCount(filteredAssets.value.length, assetPager.pageSize))
const middlewarePageCount = computed(() => pageCount(filteredMiddleware.value.length, middlewarePager.pageSize))
const taskPageCount = computed(() => pageCount(displayTasks.value.length, taskPager.pageSize))
const incidentPageCount = computed(() => pageCount(displayIncidents.value.length, incidentPager.pageSize))
const pagedAssets = computed(() => paginate(filteredAssets.value, assetPager))
const pagedMiddleware = computed(() => paginate(filteredMiddleware.value, middlewarePager))
const pagedTasks = computed(() => paginate(displayTasks.value, taskPager))
const pagedIncidents = computed(() => paginate(displayIncidents.value, incidentPager))

const selectedCopilotProvider = computed(() => copilotProviders.find((item) => item.id === copilotConfig.provider) || copilotProviders[0])

function uniqueOptions(items, field) {
  return [...new Set(items.map((item) => item[field]).filter(Boolean))]
}

function includesKeyword(item, fields, keyword) {
  if (!keyword) return true
  const query = keyword.trim().toLowerCase()
  return fields.some((field) => String(item[field] || '').toLowerCase().includes(query))
}

function filterRows(items, filters, keywordFields) {
  return items.filter((item) => {
    return includesKeyword(item, keywordFields, filters.keyword) &&
      (!filters.type || item.type === filters.type) &&
      (!filters.kind || item.kind === filters.kind) &&
      (!filters.environment || item.environment === filters.environment) &&
      (!filters.business || item.business === filters.business) &&
      (!filters.networkZone || item.networkZone === filters.networkZone) &&
      (!filters.status || item.status === filters.status)
  })
}

function pageCount(total, pageSize) {
  return Math.max(1, Math.ceil(total / pageSize))
}

function paginate(items, pager) {
  const current = Math.min(pager.page, pageCount(items.length, pager.pageSize))
  const start = (current - 1) * pager.pageSize
  return items.slice(start, start + pager.pageSize)
}

function resetAssetFilters() {
  Object.assign(assetFilters, { keyword: '', type: '', environment: '', business: '', networkZone: '', advanced: false })
  assetPager.page = 1
}

function resetMiddlewareFilters() {
  Object.assign(middlewareFilters, { keyword: '', kind: '', environment: '', business: '', networkZone: '', status: '', advanced: false })
  middlewarePager.page = 1
}

function assetSpec(asset) {
  return [asset.cpu, asset.memory, asset.disk].filter(Boolean).join(' / ') || '-'
}

function associatedAssetName(item) {
  if (!item.assetId) return '未关联'
  const asset = displayAssets.value.find((entry) => entry.id === item.assetId)
  return asset ? asset.assetNo : `资产 ID ${item.assetId}`
}

function routeHash(view = activeView.value, tab = permissionTab.value) {
  return view === 'permissions' ? `#${view}/${tab}` : `#${view}`
}

function resetPageScroll() {
  nextTick(() => window.scrollTo({ top: 0, left: 0, behavior: 'auto' }))
}

function syncRouteFromHash() {
  const next = parseRouteHash()
  if (activeView.value !== next.view || permissionTab.value !== next.permissionTab) {
    closeOpenEditors()
  }
  activeView.value = next.view
  permissionTab.value = next.permissionTab
  mobileNavOpen.value = false
  resetPageScroll()
}

function goToView(view, tab) {
  const nextPermissionTab = permissionTabs.includes(tab) ? tab : permissionTab.value
  if (activeView.value !== view || (view === 'permissions' && permissionTab.value !== nextPermissionTab)) {
    closeOpenEditors()
  }
  activeView.value = view
  if (view === 'permissions') {
    permissionTab.value = nextPermissionTab
  }
  mobileNavOpen.value = false
  resetPageScroll()
  error.value = ''
}

function toggleSidebarNavigation() {
  if (window.matchMedia('(max-width: 1040px)').matches) {
    mobileNavOpen.value = !mobileNavOpen.value
    return
  }
  sidebarCollapsed.value = !sidebarCollapsed.value
}

function closeOpenEditors() {
  if (assetFormOpen.value) closeAssetForm()
  if (middlewareFormOpen.value) closeMiddlewareForm()
  if (taskFormOpen.value) closeTaskForm()
  if (incidentFormOpen.value) closeIncidentForm()
  if (userFormOpen.value) closeUserForm()
}

function closeEditorOnFocusOut(event, closeFn) {
  const panel = event.currentTarget
  const nextTarget = event.relatedTarget
  if (nextTarget && panel?.contains(nextTarget)) return
  window.requestAnimationFrame(() => {
    const active = document.activeElement
    if (active && panel?.contains(active)) return
    closeFn()
  })
}

function openPriorityItem(item) {
  if (!item) return
  if (item.target === 'incidents') {
    chooseIncident(item.source)
  }
  if (item.target === 'tasks') {
    chooseTask(item.source)
  }
  goToView(item.target)
}

async function loadAll() {
  if (!auth.token) return
  loading.value = true
  error.value = ''
  try {
    const me = await api('/auth/me')
    auth.user = me
    if (me.mustChangePassword) {
      return
    }
    const [dashboard, assets, middleware, oncalls, tasks, incidents] = await Promise.all([
      api('/dashboard'),
      api('/assets'),
      api('/middleware'),
      api('/oncall'),
      api('/tasks'),
      api('/incidents')
    ])
    state.dashboard = { ...emptyDashboard, ...dashboard }
    state.assets = assets
    state.middleware = middleware
    state.oncalls = oncalls
    state.tasks = tasks
    state.incidents = incidents
    if ((me.roles || []).includes('super_admin')) {
      try {
        const [users, verification, copilotSettings] = await Promise.all([
          api('/users'),
          api('/security/credential-verification'),
          api('/copilot/config')
        ])
        state.users = users
        credentialVerification.hasPassword = Boolean(verification.hasPassword)
        applyCopilotConfig(copilotSettings)
      } catch {
        state.users = []
        credentialVerification.hasPassword = false
      }
    }
  } catch (err) {
    if (!auth.user) {
      clearToken()
      auth.token = ''
      error.value = `登录状态已失效，请重新登录：${err.message}`
    } else {
      error.value = `数据加载失败：${err.message}`
    }
  } finally {
    loading.value = false
  }
}

async function submitLogin() {
  loading.value = true
  error.value = ''
  try {
    const payload = await loginApi(auth.username, auth.password)
    auth.token = payload.token
    auth.user = payload.user
    await loadAll()
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

async function changeInitialPassword() {
  error.value = ''
  if (!passwordInit.currentPassword || !passwordInit.newPassword) {
    error.value = '请输入当前密码和新密码'
    return
  }
  if (passwordInit.newPassword !== passwordInit.confirmPassword) {
    error.value = '两次输入的新密码不一致'
    return
  }
  if (passwordInit.newPassword.length < 8) {
    error.value = '新密码至少需要 8 位'
    return
  }
  loading.value = true
  try {
    const user = await api('/auth/password', {
      method: 'POST',
      body: JSON.stringify({
        currentPassword: passwordInit.currentPassword,
        newPassword: passwordInit.newPassword
      })
    })
    auth.user = user
    Object.assign(passwordInit, { currentPassword: '', newPassword: '', confirmPassword: '' })
    await loadAll()
  } catch (err) {
    error.value = `初始化密码失败：${err.message}`
  } finally {
    loading.value = false
  }
}

function logout() {
  cancelConfirm()
  mobileNavOpen.value = false
  clearToken()
  auth.token = ''
  auth.user = null
  auth.username = ''
  auth.password = ''
  Object.assign(passwordInit, { currentPassword: '', newPassword: '', confirmPassword: '' })
}

function handleSessionExpired() {
  logout()
  error.value = '登录状态已失效，请重新登录。'
}

setSessionExpiredHandler(handleSessionExpired)

async function saveAsset() {
  error.value = ''
  const required = [
    ['类型', newAsset.type],
    ['CPU 架构', newAsset.cpuArch],
    ['所属业务', newAsset.business],
    ['IPv4', newAsset.ipv4],
    ['环境', newAsset.environment],
    ['操作系统', newAsset.os],
    ['网络区域', newAsset.networkZone],
    ['CPU 规格', newAsset.cpu],
    ['内存规格', newAsset.memory],
    ['磁盘规格', newAsset.disk],
    ['部署信息', newAsset.deploymentInfo],
    ['负责人', newAsset.owner]
  ]
  const missing = required.filter(([, value]) => !String(value || '').trim()).map(([label]) => label)
  if (missing.length) {
    error.value = `请补齐资产必填项：${missing.join('、')}`
    return
  }
  try {
    const method = newAsset.id ? 'PUT' : 'POST'
    const path = newAsset.id ? `/assets/${newAsset.id}` : '/assets'
    const item = await api(path, { method, body: JSON.stringify(newAsset) })
    const existingIndex = state.assets.findIndex((asset) => asset.id === item.id)
    if (existingIndex >= 0) {
      state.assets.splice(existingIndex, 1, item)
    } else {
      state.assets.unshift(item)
    }
    if (canManageCredentials.value && hasAssetFormCredential()) {
      try {
        const savedCredential = await api(`/assets/${item.id}/credential`, {
          method: 'PUT',
          body: JSON.stringify(assetFormCredential)
        })
        Object.assign(credential, savedCredential, { secret: '' })
        Object.assign(credentialReveal, { password: '', revealed: false })
        credentialMessage.value = '资产与登录信息已保存'
      } catch (err) {
        error.value = `资产已保存，但登录信息保存失败：${err.message}`
      }
    }
    selectedAsset.value = item
    closeAssetForm()
  } catch (err) {
    error.value = `保存资产失败：${err.message}`
  }
}

function openAssetForm() {
  resetAssetForm()
  assetFormOpen.value = true
}

function closeAssetForm() {
  resetAssetForm()
  assetFormOpen.value = false
}

function resetAssetForm() {
  Object.assign(newAsset, {
    id: null,
    assetNo: '',
    type: '物理机',
    vendor: '',
    cpuArch: 'x86_64',
    sn: '',
    location: '',
    business: '',
    ipv4: '',
    ipv6: '',
    environment: '生产',
    os: '',
    hostname: '',
    networkZone: '',
    cpu: '',
    memory: '',
    disk: '',
    deploymentInfo: '',
    owner: '',
    status: '运行中',
    hostMachine: ''
  })
  resetAssetFormCredential()
}

function editAsset(asset) {
  Object.assign(newAsset, { ...asset })
  resetAssetFormCredential()
  assetFormOpen.value = true
}

function hasAssetFormCredential() {
  return ['loginUrl', 'username', 'secret', 'notes'].some((key) => String(assetFormCredential[key] || '').trim())
}

function resetAssetFormCredential() {
  Object.assign(assetFormCredential, { loginUrl: '', username: '', secret: '', notes: '' })
}

async function deleteAsset(asset) {
  if (!asset) return
  await requestConfirm({
    title: '删除资产',
    message: '删除后无法恢复，关联凭据也将一并清理。',
    target: `${asset.assetNo || asset.id} · ${asset.business || asset.hostname || asset.owner || '未命名资产'}`,
    confirmLabel: '确认删除资产'
  }, async () => {
    error.value = ''
    try {
      await api(`/assets/${asset.id}`, { method: 'DELETE' })
      state.assets = state.assets.filter((item) => item.id !== asset.id)
      if (selectedAsset.value?.id === asset.id) {
        selectedAsset.value = null
      }
      if (newAsset.id === asset.id) {
        closeAssetForm()
      }
    } catch (err) {
      error.value = `删除资产失败：${err.message}`
    }
  })
}

function exportJSON(filename, payload) {
  const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}

function exportAsset(asset) {
  if (!asset) return
  exportJSON(`${asset.assetNo || `asset-${asset.id}`}.json`, asset)
}

async function saveMiddleware() {
  error.value = ''
  const required = [
    ['实例名称', newMiddleware.name],
    ['类型', newMiddleware.kind],
    ['环境', newMiddleware.environment],
    ['网络区域', newMiddleware.networkZone],
    ['访问地址 / 端口', newMiddleware.endpoint],
    ['所属业务', newMiddleware.business],
    ['负责人', newMiddleware.owner]
  ]
  const missing = required.filter(([, value]) => !String(value || '').trim()).map(([label]) => label)
  if (missing.length) {
    error.value = `请补齐实例必填项：${missing.join('、')}`
    return
  }
  try {
    const payload = { ...newMiddleware, assetId: newMiddleware.assetId ? Number(newMiddleware.assetId) : null }
    const method = newMiddleware.id ? 'PUT' : 'POST'
    const path = newMiddleware.id ? `/middleware/${newMiddleware.id}` : '/middleware'
    const item = await api(path, { method, body: JSON.stringify(payload) })
    const existingIndex = state.middleware.findIndex((entry) => entry.id === item.id)
    if (existingIndex >= 0) {
      state.middleware.splice(existingIndex, 1, item)
    } else {
      state.middleware.unshift(item)
    }
    if (canManageCredentials.value && hasMiddlewareFormCredential()) {
      try {
        const savedCredential = await api(`/middleware/${item.id}/credential`, {
          method: 'PUT',
          body: JSON.stringify(middlewareFormCredential)
        })
        Object.assign(middlewareCredential, savedCredential, { secret: '' })
        Object.assign(middlewareCredentialReveal, { password: '', revealed: false })
        middlewareCredentialMessage.value = '实例与登录信息已保存'
      } catch (err) {
        error.value = `实例已保存，但登录信息保存失败：${err.message}`
      }
    }
    selectedMiddleware.value = item
    closeMiddlewareForm()
  } catch (err) {
    error.value = `新增实例失败：${err.message}`
  }
}

function openMiddlewareForm() {
  resetMiddlewareForm()
  middlewareFormOpen.value = true
}

function closeMiddlewareForm() {
  resetMiddlewareForm()
  middlewareFormOpen.value = false
}

function resetMiddlewareForm() {
  Object.assign(newMiddleware, {
    id: null,
    name: '',
    kind: 'MySQL',
    version: '',
    environment: '生产',
    networkZone: '',
    endpoint: '',
    business: '',
    owner: '',
    status: '运行中',
    assetId: ''
  })
  resetMiddlewareFormCredential()
}

function editMiddleware(item) {
  Object.assign(newMiddleware, { ...item, assetId: item.assetId || '' })
  resetMiddlewareFormCredential()
  middlewareFormOpen.value = true
}

function hasMiddlewareFormCredential() {
  return ['loginUrl', 'username', 'secret', 'notes'].some((key) => String(middlewareFormCredential[key] || '').trim())
}

function resetMiddlewareFormCredential() {
  Object.assign(middlewareFormCredential, { loginUrl: '', username: '', secret: '', notes: '' })
}

async function deleteMiddleware(item) {
  if (!item) return
  await requestConfirm({
    title: '删除实例',
    message: '删除后无法恢复，关联凭据也将一并清理。',
    target: `${item.name || item.id} · ${item.kind || '中间件与数据库实例'}`,
    confirmLabel: '确认删除实例'
  }, async () => {
    error.value = ''
    try {
      await api(`/middleware/${item.id}`, { method: 'DELETE' })
      state.middleware = state.middleware.filter((entry) => entry.id !== item.id)
      if (selectedMiddleware.value?.id === item.id) {
        selectedMiddleware.value = null
        resetMiddlewareCredentialState()
      }
      if (newMiddleware.id === item.id) {
        closeMiddlewareForm()
      }
    } catch (err) {
      error.value = `删除实例失败：${err.message}`
    }
  })
}

function exportMiddleware(item) {
  if (!item) return
  exportJSON(`${item.name || `middleware-${item.id}`}.json`, item)
}

function chooseAsset(asset) {
  selectedAsset.value = asset
  resetAssetCredentialState()
}

function resetAssetCredentialState() {
  credentialMessage.value = ''
  Object.assign(credential, { loginUrl: '', username: '', secret: '', hasSecret: false, notes: '' })
  Object.assign(credentialReveal, { password: '', revealed: false })
}

function hideAssetDetail() {
  selectedAsset.value = null
  resetAssetCredentialState()
}

function chooseMiddleware(item) {
  selectedMiddleware.value = item
  resetMiddlewareCredentialState()
}

function resetMiddlewareCredentialState() {
  middlewareCredentialMessage.value = ''
  Object.assign(middlewareCredential, { loginUrl: '', username: '', secret: '', hasSecret: false, notes: '' })
  Object.assign(middlewareCredentialReveal, { password: '', revealed: false })
}

function hideMiddlewareDetail() {
  selectedMiddleware.value = null
  resetMiddlewareCredentialState()
}

async function loadCredential() {
  if (!selectedAsset.value) return
  credentialMessage.value = ''
  if (isSampleRecord(selectedAsset.value)) {
    credentialMessage.value = '样例资产不连接后端凭据接口，请新增真实资产后维护登录信息'
    return
  }
  try {
    const item = await api(`/assets/${selectedAsset.value.id}/credential`)
    Object.assign(credential, item, { secret: '' })
    Object.assign(credentialReveal, { password: '', revealed: false })
    credentialMessage.value = item.hasSecret ? '登录信息已加载，密码/密钥已隐藏' : '登录信息已加载，当前未保存密码/密钥'
  } catch (err) {
    credentialMessage.value = `无法加载登录信息：${err.message}`
  }
}

async function revealCredential() {
  if (isSampleRecord(selectedAsset.value)) {
    credentialMessage.value = '样例资产不支持查看密码/密钥'
    return
  }
  if (!selectedAsset.value || !credentialReveal.password) {
    credentialMessage.value = '请输入当前登录密码后再查看密码/密钥'
    return
  }
  credentialMessage.value = ''
  try {
    const item = await api(`/assets/${selectedAsset.value.id}/credential/reveal`, {
      method: 'POST',
      body: JSON.stringify({ password: credentialReveal.password })
    })
    Object.assign(credential, item)
    credentialReveal.revealed = true
    credentialReveal.password = ''
    credentialMessage.value = '已通过二次校验，密码/密钥仍以掩码输入框展示'
  } catch (err) {
    credentialMessage.value = `无法查看密码/密钥：${err.message}`
  }
}

async function saveCredential() {
  if (!selectedAsset.value) return
  if (isSampleRecord(selectedAsset.value)) {
    credentialMessage.value = '样例资产不支持保存登录信息'
    return
  }
  try {
    const item = await api(`/assets/${selectedAsset.value.id}/credential`, {
      method: 'PUT',
      body: JSON.stringify(credential)
    })
    Object.assign(credential, item, { secret: '' })
    Object.assign(credentialReveal, { password: '', revealed: false })
    credentialMessage.value = '登录信息已保存'
  } catch (err) {
    credentialMessage.value = `保存登录信息失败：${err.message}`
  }
}

async function loadMiddlewareCredential() {
  if (!selectedMiddleware.value) return
  middlewareCredentialMessage.value = ''
  if (isSampleRecord(selectedMiddleware.value)) {
    middlewareCredentialMessage.value = '样例实例不连接后端凭据接口，请新增真实实例后维护账号密码'
    return
  }
  try {
    const item = await api(`/middleware/${selectedMiddleware.value.id}/credential`)
    Object.assign(middlewareCredential, item, { secret: '' })
    Object.assign(middlewareCredentialReveal, { password: '', revealed: false })
    middlewareCredentialMessage.value = item.hasSecret ? '实例账号密码已加载，密码/密钥已隐藏' : '实例账号密码已加载，当前未保存密码/密钥'
  } catch (err) {
    middlewareCredentialMessage.value = `无法加载实例账号密码：${err.message}`
  }
}

async function revealMiddlewareCredential() {
  if (isSampleRecord(selectedMiddleware.value)) {
    middlewareCredentialMessage.value = '样例实例不支持查看密码/密钥'
    return
  }
  if (!selectedMiddleware.value || !middlewareCredentialReveal.password) {
    middlewareCredentialMessage.value = '请输入当前登录密码后再查看密码/密钥'
    return
  }
  middlewareCredentialMessage.value = ''
  try {
    const item = await api(`/middleware/${selectedMiddleware.value.id}/credential/reveal`, {
      method: 'POST',
      body: JSON.stringify({ password: middlewareCredentialReveal.password })
    })
    Object.assign(middlewareCredential, item)
    middlewareCredentialReveal.revealed = true
    middlewareCredentialReveal.password = ''
    middlewareCredentialMessage.value = '已通过二次校验，密码/密钥仍以掩码输入框展示'
  } catch (err) {
    middlewareCredentialMessage.value = `无法查看密码/密钥：${err.message}`
  }
}

async function saveMiddlewareCredential() {
  if (!selectedMiddleware.value) return
  if (isSampleRecord(selectedMiddleware.value)) {
    middlewareCredentialMessage.value = '样例实例不支持保存账号密码'
    return
  }
  try {
    const item = await api(`/middleware/${selectedMiddleware.value.id}/credential`, {
      method: 'PUT',
      body: JSON.stringify(middlewareCredential)
    })
    Object.assign(middlewareCredential, item, { secret: '' })
    Object.assign(middlewareCredentialReveal, { password: '', revealed: false })
    middlewareCredentialMessage.value = '实例账号密码已保存'
  } catch (err) {
    middlewareCredentialMessage.value = `保存实例账号密码失败：${err.message}`
  }
}

async function saveTask() {
  error.value = ''
  if (!newTask.title.trim()) {
    error.value = '请填写任务标题'
    return
  }
  try {
    const method = newTask.id ? 'PUT' : 'POST'
    const path = newTask.id ? `/tasks/${newTask.id}` : '/tasks'
    const item = await api(path, { method, body: JSON.stringify(newTask) })
    const existingIndex = state.tasks.findIndex((entry) => entry.id === item.id)
    if (existingIndex >= 0) {
      state.tasks.splice(existingIndex, 1, item)
    } else {
      state.tasks.unshift(item)
    }
    selectedTask.value = item
    closeTaskForm()
  } catch (err) {
    error.value = `保存任务失败：${err.message}`
  }
}

function openTaskForm() {
  resetTaskForm()
  taskFormOpen.value = true
}

function closeTaskForm() {
  resetTaskForm()
  taskFormOpen.value = false
}

function editTask(task) {
  Object.assign(newTask, { ...task })
  taskFormOpen.value = true
}

function resetTaskForm() {
  Object.assign(newTask, { id: null, title: '', type: '任务', assignee: '', status: '待处理', dueAt: '', description: '' })
}

async function deleteTask(task) {
  if (!task) return
  await requestConfirm({
    title: '删除任务',
    message: '删除后无法恢复，任务处理记录将从列表中移除。',
    target: `${task.title || task.id} · ${task.status || '未知状态'}`,
    confirmLabel: '确认删除任务'
  }, async () => {
    error.value = ''
    try {
      await api(`/tasks/${task.id}`, { method: 'DELETE' })
      state.tasks = state.tasks.filter((item) => item.id !== task.id)
      if (selectedTask.value?.id === task.id) {
        selectedTask.value = null
      }
      if (newTask.id === task.id) {
        closeTaskForm()
      }
    } catch (err) {
      error.value = `删除任务失败：${err.message}`
    }
  })
}

async function updateTaskStatus(task, status) {
  if (isSampleRecord(task)) return
  const previous = task.status
  error.value = ''
  try {
    await api(`/tasks/${task.id}`, { method: 'PATCH', body: JSON.stringify({ status }) })
    task.status = status
  } catch (err) {
    task.status = previous
    error.value = `更新任务状态失败：${err.message}`
  }
}

async function saveIncident() {
  error.value = ''
  if (!newIncident.title.trim()) {
    error.value = '请填写事件标题'
    return
  }
  try {
    const method = newIncident.id ? 'PUT' : 'POST'
    const path = newIncident.id ? `/incidents/${newIncident.id}` : '/incidents'
    const item = await api(path, { method, body: JSON.stringify(newIncident) })
    const existingIndex = state.incidents.findIndex((entry) => entry.id === item.id)
    if (existingIndex >= 0) {
      state.incidents.splice(existingIndex, 1, item)
    } else {
      state.incidents.unshift(item)
    }
    selectedIncident.value = item
    closeIncidentForm()
  } catch (err) {
    error.value = `保存事件失败：${err.message}`
  }
}

function openIncidentForm() {
  resetIncidentForm()
  incidentFormOpen.value = true
}

function closeIncidentForm() {
  resetIncidentForm()
  incidentFormOpen.value = false
}

function editIncident(incident) {
  Object.assign(newIncident, { ...incident })
  incidentFormOpen.value = true
}

function resetIncidentForm() {
  Object.assign(newIncident, { id: null, title: '', level: 'P3', status: '新建', owner: '', business: '', startedAt: '', recoveredAt: '', summary: '' })
}

async function deleteIncident(incident) {
  if (!incident) return
  await requestConfirm({
    title: '删除事件',
    message: '删除后无法恢复，事件跟进和闭环信息将从列表中移除。',
    target: `${incident.title || incident.id} · ${incident.level || 'P3'}`,
    confirmLabel: '确认删除事件'
  }, async () => {
    error.value = ''
    try {
      await api(`/incidents/${incident.id}`, { method: 'DELETE' })
      state.incidents = state.incidents.filter((item) => item.id !== incident.id)
      if (selectedIncident.value?.id === incident.id) {
        selectedIncident.value = null
      }
      if (newIncident.id === incident.id) {
        closeIncidentForm()
      }
    } catch (err) {
      error.value = `删除事件失败：${err.message}`
    }
  })
}

async function updateIncidentStatus(incident, status) {
  if (isSampleRecord(incident)) return
  const previous = incident.status
  error.value = ''
  try {
    await api(`/incidents/${incident.id}`, { method: 'PATCH', body: JSON.stringify({ status }) })
    incident.status = status
  } catch (err) {
    incident.status = previous
    error.value = `更新事件状态失败：${err.message}`
  }
}

function selectCopilotProvider(provider) {
  copilotConfig.provider = provider.id
  copilotConfig.endpoint = provider.endpoint
  if (provider.id === 'local') {
    copilotConfig.localEndpoint = provider.endpoint
    copilotConfig.localModel = provider.models[0]
  } else {
    copilotConfig.model = provider.models[0]
  }
  resetCopilotConnection()
}

function applyCopilotConfig(config) {
  Object.assign(copilotConfig, {
    provider: config.provider || 'openai',
    endpoint: config.endpoint || 'https://api.openai.com/v1',
    model: config.model || 'gpt-4.1',
    apiKey: '',
    hasApiKey: Boolean(config.hasApiKey),
    localEndpoint: config.localEndpoint || 'http://host.docker.internal:11434',
    localModel: config.localModel || 'qwen2.5:7b',
    temperature: config.temperature || '0.2',
    maxTokens: config.maxTokens || '4096',
    enableAssetContext: Boolean(config.enableAssetContext),
    enableIncidentContext: Boolean(config.enableIncidentContext),
    enableTaskContext: Boolean(config.enableTaskContext),
    enableOncallContext: Boolean(config.enableOncallContext),
    auditEnabled: Boolean(config.auditEnabled)
  })
  resetCopilotConnection()
}

function selectCopilotProviderByID() {
  selectCopilotProvider(selectedCopilotProvider.value)
}

function resetCopilotConnection() {
  Object.assign(copilotConnection, { testing: false, ok: null, message: '', latencyMs: null, statusCode: null })
}

async function testCopilotConnection() {
  if (!canManageUsers.value) {
    error.value = '只有超级管理员可以测试 AI Copilot 模型连接'
    return
  }
  Object.assign(copilotConnection, { testing: true, ok: null, message: '', latencyMs: null, statusCode: null })
  try {
    const result = await api('/copilot/test-connection', {
      method: 'POST',
      body: JSON.stringify({
        provider: copilotConfig.provider,
        endpoint: copilotConfig.endpoint,
        model: copilotConfig.model,
        apiKey: copilotConfig.apiKey,
        localEndpoint: copilotConfig.localEndpoint,
        localModel: copilotConfig.localModel
      })
    })
    Object.assign(copilotConnection, {
      testing: false,
      ok: result.ok,
      message: result.message,
      latencyMs: result.latencyMs,
      statusCode: result.statusCode || null
    })
    error.value = result.ok ? 'AI Copilot 连接测试通过' : `AI Copilot 连接测试未通过：${result.message}`
  } catch (err) {
    Object.assign(copilotConnection, {
      testing: false,
      ok: false,
      message: err.message || '连接测试失败',
      latencyMs: null,
      statusCode: null
    })
    error.value = `AI Copilot 连接测试失败：${copilotConnection.message}`
  }
}

async function saveCopilotConfig() {
  if (!canManageUsers.value) {
    error.value = '只有超级管理员可以保存 AI Copilot 配置'
    return
  }
  try {
    const saved = await api('/copilot/config', {
      method: 'PUT',
      body: JSON.stringify(copilotConfig)
    })
    applyCopilotConfig(saved)
    error.value = saved.hasApiKey ? 'AI Copilot 配置已保存，API Key 已由后端加密托管。' : 'AI Copilot 配置已保存；当前未配置托管 API Key。'
  } catch (err) {
    error.value = `AI Copilot 配置保存失败：${err.message}`
  }
}

async function saveUser() {
  error.value = ''
  if (!canManageUsers.value) {
    error.value = '当前角色无权管理用户与角色'
    return
  }
  if (!newUser.username || !newUser.displayName || (!newUser.id && !newUser.password)) {
    error.value = '请补齐账号、姓名和初始密码'
    return
  }
  if (newUser.password && newUser.password.length < 8) {
    error.value = '用户密码至少需要 8 位'
    return
  }
  try {
    const payload = {
      username: newUser.username,
      displayName: newUser.displayName,
      password: newUser.password,
      mustChangePassword: newUser.mustChangePassword,
      roles: [newUser.role]
    }
    const method = newUser.id ? 'PUT' : 'POST'
    const path = newUser.id ? `/users/${newUser.id}` : '/users'
    const item = await api(path, { method, body: JSON.stringify(payload) })
    const existingIndex = state.users.findIndex((entry) => entry.id === item.id)
    if (existingIndex >= 0) {
      state.users.splice(existingIndex, 1, item)
    } else {
      state.users.unshift(item)
    }
    if (item.id === currentUserID()) {
      auth.user = { ...auth.user, username: item.username, displayName: item.displayName, roles: [...item.roles], mustChangePassword: item.mustChangePassword }
    }
    closeUserForm()
  } catch (err) {
    error.value = `保存用户失败：${err.message}`
  }
}

function openUserForm() {
  if (!canManageUsers.value) {
    error.value = '当前角色无权管理用户与角色'
    return
  }
  resetUserForm()
  userFormOpen.value = true
}

function closeUserForm() {
  resetUserForm()
  userFormOpen.value = false
}

function editUser(user) {
  if (!canManageUsers.value) {
    error.value = '当前角色无权管理用户与角色'
    return
  }
  Object.assign(newUser, {
    id: user.id,
    username: user.username,
    displayName: user.displayName,
    password: '',
    mustChangePassword: user.mustChangePassword,
    role: user.roles?.[0] || 'ops_engineer'
  })
  userFormOpen.value = true
}

function resetUserForm() {
  Object.assign(newUser, { id: null, username: '', displayName: '', password: '', mustChangePassword: true, role: 'ops_engineer' })
}

async function deleteUser(user) {
  if (!canManageUsers.value) {
    error.value = '当前角色无权管理用户与角色'
    return
  }
  if (!user) return
  await requestConfirm({
    title: '删除用户',
    message: '删除后该账号将无法登录，相关角色关系也会被移除。',
    target: `${user.username} · ${user.displayName || '未设置姓名'}`,
    confirmLabel: '确认删除用户'
  }, async () => {
    error.value = ''
    try {
      await api(`/users/${user.id}`, { method: 'DELETE' })
      state.users = state.users.filter((item) => item.id !== user.id)
      if (newUser.id === user.id) {
        closeUserForm()
      }
    } catch (err) {
      error.value = `删除用户失败：${err.message}`
    }
  })
}

async function saveCredentialVerificationPassword() {
  credentialVerification.message = ''
  if (!canManageUsers.value) {
    credentialVerification.message = '当前角色无权设置二次校验密码'
    return
  }
  if (!credentialVerification.password || !credentialVerification.confirm) {
    credentialVerification.message = '请输入并确认统一二次校验密码'
    return
  }
  if (credentialVerification.password !== credentialVerification.confirm) {
    credentialVerification.message = '两次输入的校验密码不一致'
    return
  }
  if (credentialVerification.password.length < 8) {
    credentialVerification.message = '校验密码至少需要 8 位'
    return
  }
  try {
    const result = await api('/security/credential-verification', {
      method: 'PUT',
      body: JSON.stringify({ password: credentialVerification.password })
    })
    credentialVerification.hasPassword = Boolean(result.hasPassword)
    credentialVerification.password = ''
    credentialVerification.confirm = ''
    credentialVerification.message = '统一二次校验密码已更新'
  } catch (err) {
    credentialVerification.message = `设置失败：${err.message}`
  }
}

function chooseTask(task) {
  selectedTask.value = task
}

function hideTaskDetail() {
  selectedTask.value = null
}

function chooseIncident(incident) {
  selectedIncident.value = incident
}

function hideIncidentDetail() {
  selectedIncident.value = null
}

function hideCopilot() {
  copilotOpen.value = false
  copilotExpanded.value = false
}

function toggleCopilotSize() {
  copilotExpanded.value = !copilotExpanded.value
}

function topItems(items, count = 3) {
  return items.slice(0, count).map((item) => item.title || item.assetNo || item.name || item.primary).filter(Boolean)
}

function itemMatchesQuestion(question, values) {
  const haystack = values.join(' ').toLowerCase()
  const query = question.toLowerCase()
  return haystack.includes(query) || values.some((value) => {
    const text = String(value || '').trim().toLowerCase()
    return text.length >= 2 && query.includes(text)
  })
}

function answerCopilot(question) {
  const query = question.trim().toLowerCase()
  const p1Incidents = activeIncidents.value.filter((item) => item.level === 'P1')
  const pendingTasks = displayTasks.value.filter((item) => ['待处理', '处理中', '待确认'].includes(item.status))
  const oncall = todayOncall.value
  const matchedAssets = displayAssets.value.filter((asset) => {
    return query && itemMatchesQuestion(query, [asset.assetNo, asset.business, asset.ipv4, asset.ipv6, asset.owner, asset.deploymentInfo, asset.networkZone])
  })
  const matchedMiddleware = displayMiddleware.value.filter((item) => {
    return query && itemMatchesQuestion(query, [item.name, item.kind, item.business, item.endpoint, item.networkZone, associatedAssetName(item)])
  })

  if (query.includes('p1') || query.includes('事件') || query.includes('异常')) {
    const names = topItems(p1Incidents)
    return p1Incidents.length
      ? `当前有 ${p1Incidents.length} 个 P1 活跃事件：${names.join('、')}。建议先确认影响业务、关联资产和值班负责人，再推进恢复与关闭流程。`
      : `当前没有 P1 活跃事件。仍有 ${activeIncidents.value.length} 个未关闭事件，建议继续关注处理中和已恢复未关闭的记录。`
  }

  if (query.includes('值班') || query.includes('主值') || query.includes('备值')) {
    return `今日主值：${oncall.primary || '未配置'}，备值：${oncall.backup || '未配置'}，规则：${oncall.ruleType === 'weekly' ? '按周轮换' : '按天轮换'}。如发生 P1/P2，建议优先拉起主值并同步备值。`
  }

  if (query.includes('任务') || query.includes('待办')) {
    const names = topItems(pendingTasks)
    return `当前未关闭任务 ${pendingTasks.length} 个，其中处理中 ${taskStatusCounts.value['处理中'] || 0} 个、待确认 ${taskStatusCounts.value['待确认'] || 0} 个。优先关注：${names.join('、') || '暂无高优先任务'}。`
  }

  if (query.includes('资产') || query.includes('cmdb') || matchedAssets.length || matchedMiddleware.length) {
    return `查询到资产 ${matchedAssets.length} 条、实例 ${matchedMiddleware.length} 条。${matchedAssets.length ? `资产示例：${topItems(matchedAssets).join('、')}。` : ''}${matchedMiddleware.length ? `实例示例：${topItems(matchedMiddleware).join('、')}。` : ''}建议结合环境、网络区域和所属业务判断影响范围。`
  }

  if (query.includes('凭据') || query.includes('密码') || query.includes('权限')) {
    return canManageCredentials.value
      ? '当前账号具备敏感凭据管理权限。资产和实例密码默认加密存储，查看时仍需输入当前登录密码二次校验。'
      : '当前账号无敏感凭据查看权限。运维工程师可维护资产和实例基础信息，但账号密码默认不可见。'
  }

  return `当前健康概览：纳管资产 ${dashboardAssetCount.value} 个，活跃事件 ${activeIncidents.value.length} 个，未关闭任务 ${pendingTasks.length} 个，今日值班 ${oncall.primary || '未配置主值'} / ${oncall.backup || '未配置备值'}。建议先看活跃事件影响，再跟进任务闭环。`
}

function askCopilot() {
  const question = copilotQuestion.value.trim()
  if (!question) return
  copilotMessages.value.push({ role: 'user', text: question })
  copilotMessages.value.push({ role: 'ai', text: answerCopilot(question) })
  copilotQuestion.value = ''
}

watch([activeView, permissionTab], () => {
  const nextHash = routeHash()
  if (window.location.hash !== nextHash) {
    window.history.replaceState(null, '', nextHash)
  }
})

onMounted(() => {
  syncRouteFromHash()
  window.addEventListener('hashchange', syncRouteFromHash)
  loadAll()
})

onUnmounted(() => {
  window.removeEventListener('hashchange', syncRouteFromHash)
  setSessionExpiredHandler(null)
})
</script>

<template>
  <div class="app" :class="{ 'auth-app': !hasAppAccess, 'sidebar-collapsed': hasAppAccess && sidebarCollapsed && !mobileNavOpen }">
    <button
      v-if="hasAppAccess && mobileNavOpen"
      class="mobile-nav-backdrop"
      type="button"
      aria-label="关闭导航"
      @click="mobileNavOpen = false"
    />
    <SidebarNav
      v-if="hasAppAccess"
      :active-view="activeView"
      :permission-tab="permissionTab"
      :collapsed="sidebarCollapsed && !mobileNavOpen"
      :mobile-open="mobileNavOpen"
      @toggle="toggleSidebarNavigation"
      @navigate="goToView"
    />

    <main class="main">
      <Topbar
        v-if="hasAppAccess"
        :breadcrumb="activeBreadcrumb"
        :title="activeTitle"
        :subtitle="activeSubtitle"
        :error="error"
        :loading="loading"
        :username="auth.user?.username || 'admin'"
        :authenticated="Boolean(auth.token)"
        @open-navigation="mobileNavOpen = true"
        @open-copilot="copilotOpen = true"
        @logout="logout"
      />

      <AuthView
        v-if="!hasAppAccess"
        :auth="auth"
        :error="error"
        :loading="loading"
        :needs-initial-password="needsInitialPassword"
        :password-init="passwordInit"
        :pending="authPending"
        @change-password="changeInitialPassword"
        @logout="logout"
        @submit-login="submitLogin"
      />

      <template v-else>
        <DashboardView
          v-if="activeView === 'dashboard'"
          :asset-count="dashboardAssetCount"
          :oncall-count="dashboardOncallCount"
          :task-count="dashboardTaskCount"
          :incident-count="dashboardIncidentCount"
          :asset-bars="assetKpiBars"
          :today-oncall="todayOncall"
          :task-cards="taskKpiCards"
          :incident-levels="incidentKpiLevels"
          :metrics="dashboardMetrics"
          :priority-items="dashboardPriorityItems"
          @navigate="goToView"
          @open-priority="openPriorityItem"
        />

        <AssetView
          v-if="activeView === 'cmdb'"
          :businesses="assetBusinesses"
          :can-delete="canDeleteAsset"
          :can-manage-credentials="canManageCredentials"
          :can-write="canWriteAssets"
          :credential="credential"
          :credential-message="credentialMessage"
          :credential-reveal="credentialReveal"
          :filters="assetFilters"
          :form="newAsset"
          :form-credential="assetFormCredential"
          :form-open="assetFormOpen"
          :is-sample="isSampleRecord"
          :network-zones="assetNetworkZones"
          :page="assetPager.page"
          :page-count="assetPageCount"
          :page-size="assetPager.pageSize"
          :paged-assets="pagedAssets"
          :selected-asset="selectedAsset"
          :spec="assetSpec"
          :total="filteredAssets.length"
          @choose="chooseAsset"
          @close-form="closeAssetForm"
          @delete="deleteAsset"
          @edit="editAsset"
          @export-item="exportAsset"
          @hide-detail="hideAssetDetail"
          @load-credential="loadCredential"
          @open-form="openAssetForm"
          @reset-filters="resetAssetFilters"
          @reveal-credential="revealCredential"
          @save="saveAsset"
          @save-credential="saveCredential"
          @update:page="assetPager.page = $event"
          @update:page-size="assetPager.pageSize = $event"
        />

        <MiddlewareView
          v-if="activeView === 'middleware'"
          :associated-asset-name="associatedAssetName"
          :businesses="middlewareBusinesses"
          :can-manage-credentials="canManageCredentials"
          :can-write="canWriteAssets"
          :credential="middlewareCredential"
          :credential-message="middlewareCredentialMessage"
          :credential-reveal="middlewareCredentialReveal"
          :filters="middlewareFilters"
          :form="newMiddleware"
          :form-credential="middlewareFormCredential"
          :form-open="middlewareFormOpen"
          :is-sample="isSampleRecord"
          :network-zones="middlewareNetworkZones"
          :page="middlewarePager.page"
          :page-count="middlewarePageCount"
          :page-size="middlewarePager.pageSize"
          :paged-items="pagedMiddleware"
          :selected-item="selectedMiddleware"
          :total="filteredMiddleware.length"
          @choose="chooseMiddleware"
          @close-form="closeMiddlewareForm"
          @delete="deleteMiddleware"
          @edit="editMiddleware"
          @export-item="exportMiddleware"
          @hide-detail="hideMiddlewareDetail"
          @load-credential="loadMiddlewareCredential"
          @open-form="openMiddlewareForm"
          @reset-filters="resetMiddlewareFilters"
          @reveal-credential="revealMiddlewareCredential"
          @save="saveMiddleware"
          @save-credential="saveMiddlewareCredential"
          @update:page="middlewarePager.page = $event"
          @update:page-size="middlewarePager.pageSize = $event"
        />

        <DutyManagementView
          v-show="activeView === 'oncall'"
          :active="activeView === 'oncall'"
          :can-write="canWriteOncall"
          :request-confirm="requestConfirm"
          :users="displayUsers"
          @error="error = $event"
        />

        <TaskView
          v-if="activeView === 'tasks'"
          :active-task-count="activeTasks.length"
          :current-task="currentTask"
          :form="newTask"
          :form-open="taskFormOpen"
          :is-sample="isSampleRecord"
          :page="taskPager.page"
          :page-count="taskPageCount"
          :page-size="taskPager.pageSize"
          :paged-tasks="pagedTasks"
          :status-counts="taskStatusCounts"
          :total="displayTasks.length"
          @choose="chooseTask"
          @close-form="closeTaskForm"
          @delete="deleteTask"
          @edit="editTask"
          @hide-detail="hideTaskDetail"
          @open-form="openTaskForm"
          @save="saveTask"
          @update-status="updateTaskStatus"
          @update:page="taskPager.page = $event"
          @update:page-size="taskPager.pageSize = $event"
        />

        <IncidentView
          v-if="activeView === 'incidents'"
          :active-incident-count="activeIncidents.length"
          :current-incident="currentIncident"
          :form="newIncident"
          :form-open="incidentFormOpen"
          :is-sample="isSampleRecord"
          :level-counts="incidentLevelCounts"
          :page="incidentPager.page"
          :page-count="incidentPageCount"
          :page-size="incidentPager.pageSize"
          :paged-incidents="pagedIncidents"
          :total="displayIncidents.length"
          @choose="chooseIncident"
          @close-form="closeIncidentForm"
          @delete="deleteIncident"
          @edit="editIncident"
          @hide-detail="hideIncidentDetail"
          @open-form="openIncidentForm"
          @save="saveIncident"
          @update-status="updateIncidentStatus"
          @update:page="incidentPager.page = $event"
          @update:page-size="incidentPager.pageSize = $event"
        />

        <PermissionsView
          v-if="activeView === 'permissions'"
          :can-manage="canManageUsers"
          :credential-verification="credentialVerification"
          :current-user-id="currentUserID()"
          :display-users="displayUsers"
          :is-sample="isSampleRecord"
          :menu-permission-rows="menuPermissionRows"
          :new-user="newUser"
          :permission-rows="permissionRows"
          :permission-tab="permissionTab"
          :role-cards="roleCards"
          :user-form-open="userFormOpen"
          @close-user-form="closeUserForm"
          @delete-user="deleteUser"
          @edit-user="editUser"
          @navigate-tab="goToView('permissions', $event)"
          @open-user-form="openUserForm"
          @save-credential-password="saveCredentialVerificationPassword"
          @save-user="saveUser"
        />

        <CopilotSettingsView
          v-if="activeView === 'copilot-settings'"
          :can-manage="canManageUsers"
          :config="copilotConfig"
          :connection="copilotConnection"
          :providers="copilotProviders"
          :selected-provider="selectedCopilotProvider"
          @save="saveCopilotConfig"
          @select-provider="selectCopilotProvider"
          @select-provider-id="selectCopilotProviderByID"
          @test="testCopilotConnection"
        />

      </template>
    </main>

    <CopilotWidget
      v-if="hasAppAccess"
      v-model:question="copilotQuestion"
      :open="copilotOpen"
      :expanded="copilotExpanded"
      :messages="copilotMessages"
      @hide="hideCopilot"
      @toggle-size="toggleCopilotSize"
      @send="askCopilot"
    />
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
  </div>
</template>
