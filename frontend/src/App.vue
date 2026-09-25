<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { api, clearToken, getToken, login as loginApi, setSessionExpiredHandler } from './api'
import { copilotProviders, menuPermissionRows, permissionRows, roleCards } from './app-config'
import AssetView from './components/AssetView.vue'
import { bulkSchemas, downloadText, parseCsv, toCsv } from './bulk-data'
import AuditView from './components/AuditView.vue'
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
import ToastStack from './components/ToastStack.vue'
import Topbar from './components/Topbar.vue'
import { useConfirmDialog } from './composables/useConfirmDialog'
import { useCopilotChat } from './composables/useCopilotChat'
import { useDashboardViewModel } from './composables/useDashboardViewModel'
import { useDeferredLoader } from './composables/useDeferredLoader'
import { useToast } from './composables/useToast'
import { useWorkspaceRoute } from './composables/useWorkspaceRoute'
import { toAPITimestamp, toDateTimeLocal } from './date-time'
import { buildListPath } from './list-query'
import { controlPrinciple, viewMeta } from './navigation'
import { sampleAssets, sampleIncidents, sampleMiddleware, sampleOncalls, sampleTasks, sampleUsers } from './sample-data'

const demoDataEnabled = import.meta.env.VITE_ENABLE_DEMO_DATA === 'true'

const { activeView, permissionTab, goToView } = useWorkspaceRoute({
  beforeNavigate: () => closeOpenEditors(),
  afterNavigate: ({ clearFeedback }) => {
    mobileNavOpen.value = false
    if (clearFeedback) error.value = ''
  }
})
const auth = reactive({
  token: getToken(),
  user: null,
  username: '',
  password: ''
})
const {
  open: copilotOpen,
  expanded: copilotExpanded,
  question: copilotQuestion,
  messages: copilotMessages,
  busy: copilotBusy,
  hide: hideCopilot,
  toggleSize: toggleCopilotSize,
  setUserID: setCopilotUserID,
  send: askCopilot
} = useCopilotChat()
watch(
  () => auth.user?.id || auth.user?.uid || auth.user?.userID || auth.user?.UserID || '',
  setCopilotUserID,
  { immediate: true }
)
const sidebarCollapsed = ref(true)
const mobileNavOpen = ref(false)
const loading = ref(false)
const error = ref('')
const { messages: toastMessages, notify, dismiss: dismissToast } = useToast()
const { confirmState, requestConfirm, cancelConfirm, acceptConfirm } = useConfirmDialog((err) => {
  error.value = err.message || '操作失败'
})
const selectedAsset = ref(null)
const selectedMiddleware = ref(null)
const selectedTask = ref(null)
const selectedIncident = ref(null)
const selectedAssetItems = ref([])
const selectedMiddlewareItems = ref([])
const assetFormOpen = ref(false)
const assetBulkBusy = ref(false)
const middlewareBulkBusy = ref(false)
const middlewareFormOpen = ref(false)
const taskFormOpen = ref(false)
const incidentFormOpen = ref(false)
const userFormOpen = ref(false)
const copilotEditorMode = ref('closed')
const copilotProfiles = ref([])
const credential = reactive({ loginUrl: '', username: '', secret: '', hasSecret: false, notes: '' })
const credentialReveal = reactive({ password: '', revealed: false })
const credentialMessage = ref('')
const assetFormCredential = reactive({ loginUrl: '', username: '', secret: '', notes: '' })
const middlewareCredential = reactive({ loginUrl: '', username: '', secret: '', hasSecret: false, notes: '' })
const middlewareCredentialReveal = reactive({ password: '', revealed: false })
const middlewareCredentialMessage = ref('')
const middlewareFormCredential = reactive({ loginUrl: '', username: '', secret: '', notes: '' })
const credentialVerification = reactive({ hasPassword: false, password: '', confirm: '', message: '' })
const passwordInit = reactive({
  currentPassword: '',
  newPassword: '',
  confirmPassword: ''
})
const hasAppAccess = computed(() => Boolean(auth.token && auth.user && !auth.user.mustChangePassword))
const needsInitialPassword = computed(() => Boolean(auth.token && auth.user?.mustChangePassword))
const authPending = computed(() => Boolean(auth.token && !auth.user))
const emptyDashboard = {
  assetCount: 0,
  todayOnCallCount: 0,
  activeTaskCount: 0,
  activeIncidentCount: 0,
  assetTypeCounts: {},
  incidentLevelCounts: {},
  taskStatusCounts: {},
  assetHealthyCount: 0,
  assetAbnormalCount: 0,
  taskClosedCount: 0,
  taskOpenCount: 0,
  incidentClosedCount: 0,
  responseMinutesTotal: 0,
  responseSampleCount: 0
}

const state = reactive({
  dashboard: { ...emptyDashboard },
  assets: [],
  middleware: [],
  oncalls: [],
  tasks: [],
  incidents: [],
  users: [],
  userDirectory: [],
  auditEvents: []
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

const newTask = reactive({ id: null, title: '', type: '任务', assignee: '', assigneeUserId: '', status: '待处理', dueAt: '', description: '' })
const newIncident = reactive({ id: null, title: '', level: 'P3', status: '新建', owner: '', ownerUserId: '', business: '', startedAt: '', recoveredAt: '', summary: '' })
const newUser = reactive({ id: null, username: '', displayName: '', password: '', mustChangePassword: true, role: 'ops_engineer' })
const copilotConfig = reactive({
  id: null,
  name: '',
  provider: 'openai',
  model: 'gpt-4.1',
  endpoint: 'https://api.openai.com/v1',
  localEndpoint: 'http://host.docker.internal:11434',
  localModel: 'qwen2.5:7b',
  apiKey: '',
  hasApiKey: false,
  temperature: '0.2',
  maxTokens: '2048',
  enableAssetContext: true,
  enableIncidentContext: true,
  enableTaskContext: true,
  enableOncallContext: true,
  auditEnabled: true,
  isActive: false
})
const copilotConnection = reactive({
  testing: false,
  ok: null,
  message: '',
  latencyMs: null,
  statusCode: null
})
const assetFilters = reactive({ keyword: '', ips: '', type: '', environment: '', business: '', networkZone: '', advanced: false })
const middlewareFilters = reactive({ keyword: '', ips: '', kind: '', environment: '', business: '', networkZone: '', status: '', advanced: false })
const assetPager = reactive({ page: 1, pageSize: 10 })
const middlewarePager = reactive({ page: 1, pageSize: 10 })
const taskPager = reactive({ page: 1, pageSize: 10 })
const incidentPager = reactive({ page: 1, pageSize: 10 })
const listMeta = reactive({
  assets: { total: 0, pageCount: 1, options: { business: [], networkZone: [] } },
  middleware: { total: 0, pageCount: 1, options: { business: [], networkZone: [] } },
  tasks: { total: 0, pageCount: 1, counts: {} },
  incidents: { total: 0, pageCount: 1, counts: {} }
})

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
const canBulkDeleteAssets = computed(() => selectedAssetItems.value.length > 0 && selectedAssetItems.value.every(canDeleteAsset))
const canBulkDeleteMiddleware = computed(() => canWriteAssets.value && selectedMiddlewareItems.value.length > 0 && selectedMiddlewareItems.value.every((item) => !isSampleRecord(item)))
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

function toggleSelection(selection, item) {
  const index = selection.value.findIndex((selected) => selected.id === item.id)
  if (index >= 0) {
    selection.value = selection.value.filter((selected) => selected.id !== item.id)
    return
  }
  if (selection.value.length >= 500) {
    notify('最多可同时选择 500 条记录')
    return
  }
  selection.value = [...selection.value, item]
}

function setPageSelection(selection, items, checked) {
  const pageIds = new Set(items.map((item) => item.id))
  if (!checked) {
    selection.value = selection.value.filter((item) => !pageIds.has(item.id))
    return
  }
  const selectedIds = new Set(selection.value.map((item) => item.id))
  const additions = items.filter((item) => !selectedIds.has(item.id))
  const remaining = Math.max(0, 500 - selection.value.length)
  selection.value = [...selection.value, ...additions.slice(0, remaining)]
  if (additions.length > remaining) notify('最多可同时选择 500 条记录，已选择前 500 条')
}

function selectAssetPage(checked, items) {
  setPageSelection(selectedAssetItems, items, checked)
}

function selectMiddlewarePage(checked, items) {
  setPageSelection(selectedMiddlewareItems, items, checked)
}

function toggleAssetSelection(item) {
  toggleSelection(selectedAssetItems, item)
}

function toggleMiddlewareSelection(item) {
  toggleSelection(selectedMiddlewareItems, item)
}

function clearAssetSelection() {
  selectedAssetItems.value = []
}

function clearMiddlewareSelection() {
  selectedMiddlewareItems.value = []
}

function isSampleRecord(item) {
  return Boolean(item?.__sample)
}

const displayAssets = computed(() => state.assets.length ? state.assets : (demoDataEnabled ? sampleAssets : []))
const displayMiddleware = computed(() => state.middleware.length ? state.middleware : (demoDataEnabled ? sampleMiddleware : []))
const displayOncalls = computed(() => state.oncalls.length ? state.oncalls : (demoDataEnabled ? sampleOncalls : []))
const displayTasks = computed(() => state.tasks.length ? state.tasks : (demoDataEnabled ? sampleTasks : []))
const displayIncidents = computed(() => state.incidents.length ? state.incidents : (demoDataEnabled ? sampleIncidents : []))
const displayUsers = computed(() => state.users.length ? state.users : (demoDataEnabled ? sampleUsers : []))
const currentTask = computed(() => selectedTask.value || null)
const currentIncident = computed(() => selectedIncident.value || null)
const {
  assetKpiBars,
  dashboardAssetCount,
  dashboardIncidentCount,
  dashboardMetrics,
  dashboardOncallCount,
  dashboardPriorityItems,
  dashboardTaskCount,
  incidentKpiLevels,
  incidentLevelCounts,
  taskKpiCards,
  taskStatusCounts,
  todayOncall
} = useDashboardViewModel({
  demoDataEnabled,
  state,
  listMeta,
  displayAssets,
  displayMiddleware,
  displayOncalls,
  displayTasks,
  displayIncidents
})
const assetBusinesses = computed(() => listMeta.assets.options.business.length ? listMeta.assets.options.business : uniqueOptions(displayAssets.value, 'business'))
const assetNetworkZones = computed(() => listMeta.assets.options.networkZone.length ? listMeta.assets.options.networkZone : uniqueOptions(displayAssets.value, 'networkZone'))
const middlewareBusinesses = computed(() => listMeta.middleware.options.business.length ? listMeta.middleware.options.business : uniqueOptions(displayMiddleware.value, 'business'))
const middlewareNetworkZones = computed(() => listMeta.middleware.options.networkZone.length ? listMeta.middleware.options.networkZone : uniqueOptions(displayMiddleware.value, 'networkZone'))
const assetPageCount = computed(() => listMeta.assets.pageCount)
const middlewarePageCount = computed(() => listMeta.middleware.pageCount)
const taskPageCount = computed(() => listMeta.tasks.pageCount)
const incidentPageCount = computed(() => listMeta.incidents.pageCount)
const pagedAssets = computed(() => displayAssets.value)
const pagedMiddleware = computed(() => displayMiddleware.value)
const pagedTasks = computed(() => displayTasks.value)
const pagedIncidents = computed(() => displayIncidents.value)

const selectedCopilotProvider = computed(() => copilotProviders.find((item) => item.id === copilotConfig.provider) || copilotProviders[0])

function uniqueOptions(items, field) {
  return [...new Set(items.map((item) => item[field]).filter(Boolean))]
}

function resetAssetFilters() {
  Object.assign(assetFilters, { keyword: '', ips: '', type: '', environment: '', business: '', networkZone: '', advanced: false })
  assetPager.page = 1
}

function resetMiddlewareFilters() {
  Object.assign(middlewareFilters, { keyword: '', ips: '', kind: '', environment: '', business: '', networkZone: '', status: '', advanced: false })
  middlewarePager.page = 1
}

function assetSpec(asset) {
  return [asset.cpu, asset.memory, asset.disk].filter(Boolean).join(' / ') || '-'
}

function associatedAssetName(item) {
  if (!item.assetId) return '未关联'
  if (item.assetNo) return item.assetNo
  const asset = displayAssets.value.find((entry) => entry.id === item.assetId)
  return asset ? asset.assetNo : `资产 ID ${item.assetId}`
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
  if (copilotEditorMode.value !== 'closed') closeCopilotEditor()
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

function applyPageResult(key, result, pager) {
  state[key] = Array.isArray(result?.items) ? result.items : []
  listMeta[key].total = Number(result?.total || 0)
  listMeta[key].pageCount = Number(result?.pageCount || 1)
  if (result?.counts) listMeta[key].counts = result.counts
  if (result?.options) listMeta[key].options = result.options
  if (result?.page && pager.page !== result.page) pager.page = result.page
}

async function loadDashboard() {
  state.dashboard = { ...emptyDashboard, ...(await api('/dashboard')) }
}

async function loadAssetPage() {
  applyPageResult('assets', await api(buildListPath('assets', assetPager, assetFilters)), assetPager)
}

async function loadMiddlewarePage() {
  applyPageResult('middleware', await api(buildListPath('middleware', middlewarePager, middlewareFilters)), middlewarePager)
}

async function loadTaskPage() {
  applyPageResult('tasks', await api(buildListPath('tasks', taskPager)), taskPager)
}

async function loadIncidentPage() {
  applyPageResult('incidents', await api(buildListPath('incidents', incidentPager)), incidentPager)
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
    const [dashboard, assets, middleware, oncalls, tasks, incidents, userDirectory] = await Promise.all([
      api('/dashboard'),
      api(buildListPath('assets', assetPager, assetFilters)),
      api(buildListPath('middleware', middlewarePager, middlewareFilters)),
      api('/oncall'),
      api(buildListPath('tasks', taskPager)),
      api(buildListPath('incidents', incidentPager)),
      api('/user-directory')
    ])
    state.dashboard = { ...emptyDashboard, ...dashboard }
    applyPageResult('assets', assets, assetPager)
    applyPageResult('middleware', middleware, middlewarePager)
    state.oncalls = oncalls
    applyPageResult('tasks', tasks, taskPager)
    applyPageResult('incidents', incidents, incidentPager)
    state.userDirectory = userDirectory
    if ((me.roles || []).includes('super_admin')) {
      const [users, verification, modelProfiles, auditEvents] = await Promise.allSettled([
        api('/users'),
        api('/security/credential-verification'),
        api('/copilot/configs'),
        api('/audit-events?limit=100')
      ])
      state.users = users.status === 'fulfilled' ? users.value : []
      credentialVerification.hasPassword = verification.status === 'fulfilled' && Boolean(verification.value.hasPassword)
      state.auditEvents = auditEvents.status === 'fulfilled' ? auditEvents.value : []
      copilotProfiles.value = modelProfiles.status === 'fulfilled' ? modelProfiles.value : []
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
    await Promise.all([loadAssetPage(), loadDashboard()])
    selectedAsset.value = state.assets.find(asset => asset.id === item.id) || item
    notify(method === 'PUT' ? '资产修改已保存' : '资产已纳管')
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
      selectedAssetItems.value = selectedAssetItems.value.filter((item) => item.id !== asset.id)
      if (selectedAsset.value?.id === asset.id) {
        selectedAsset.value = null
      }
      if (newAsset.id === asset.id) {
        closeAssetForm()
      }
      await Promise.all([loadAssetPage(), loadDashboard()])
      notify('资产已删除')
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
    await Promise.all([loadMiddlewarePage(), loadDashboard()])
    selectedMiddleware.value = state.middleware.find(entry => entry.id === item.id) || item
    notify(method === 'PUT' ? '实例修改已保存' : '实例已创建')
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
      selectedMiddlewareItems.value = selectedMiddlewareItems.value.filter((entry) => entry.id !== item.id)
      if (selectedMiddleware.value?.id === item.id) {
        selectedMiddleware.value = null
        resetMiddlewareCredentialState()
      }
      if (newMiddleware.id === item.id) {
        closeMiddlewareForm()
      }
      await Promise.all([loadMiddlewarePage(), loadDashboard()])
    } catch (err) {
      error.value = `删除实例失败：${err.message}`
    }
  })
}

function exportMiddleware(item) {
  if (!item) return
  exportJSON(`${item.name || `middleware-${item.id}`}.json`, item)
}

function downloadBulkTemplate(resource) {
  downloadText(`${resource}-import-template.csv`, toCsv([], bulkSchemas[resource]))
}

async function importCsvFile(resource, file) {
  const busy = resource === 'assets' ? assetBulkBusy : middlewareBulkBusy
  const label = resource === 'assets' ? '资产' : '实例'
  busy.value = true
  error.value = ''
  try {
    const records = parseCsv(await file.text(), bulkSchemas[resource])
    if (records.length > 500) throw new Error('单次最多导入 500 行，请拆分文件后重试')
    const confirmed = await requestConfirm({
      title: `确认批量导入${label}`,
      message: resource === 'assets'
        ? `将处理 ${records.length} 行基础资产数据。相同资产编号会更新现有资产；导入文件不包含凭据。`
        : `将创建 ${records.length} 条实例记录。导入文件不包含凭据。`,
      target: file.name,
      confirmLabel: '开始导入'
    }, async () => {})
    if (!confirmed) return
    let cursor = 0
    const failures = []
    let successCount = 0
    const worker = async () => {
      while (cursor < records.length) {
        const record = records[cursor++]
        const { __row, ...payload } = record
        if (resource === 'middleware' && payload.assetId) {
          const assetId = Number(payload.assetId)
          if (!Number.isSafeInteger(assetId) || assetId < 1) {
            failures.push({ row: __row, message: '关联资产 ID 必须是正整数' })
            continue
          }
          payload.assetId = assetId
        } else if (resource === 'middleware') {
          delete payload.assetId
        }
        try {
          await api(`/${resource}`, { method: 'POST', body: JSON.stringify(payload) })
          successCount += 1
        } catch (err) {
          failures.push({ row: __row, message: err.message })
        }
      }
    }
    await Promise.all(Array.from({ length: Math.min(5, records.length) }, worker))
    if (successCount) {
      if (resource === 'assets') await Promise.all([loadAssetPage(), loadDashboard()])
      else await Promise.all([loadMiddlewarePage(), loadDashboard()])
    }
    if (failures.length) {
      const rowsByNumber = new Map(records.map((record) => [record.__row, record]))
      const reportSchema = [['__row', 'CSV行号'], ['__error', '失败原因'], ...bulkSchemas[resource]]
      const reportRows = failures.map(({ row, message }) => ({ __row: row, __error: message, ...rowsByNumber.get(row) }))
      downloadText(`${resource}-import-errors.csv`, toCsv(reportRows, reportSchema))
      error.value = `批量导入${label}完成：成功 ${successCount} 条，失败 ${failures.length} 条。${failures.slice(0, 5).map(({ row, message }) => `第 ${row} 行：${message}`).join('；')}`
      notify('失败行明细已下载为 CSV')
    } else {
      notify(`批量导入${label}成功，共 ${successCount} 条`)
    }
  } catch (err) {
    error.value = `批量导入${label}失败：${err.message}`
  } finally {
    busy.value = false
  }
}

function exportSelected(resource, items) {
  if (!items.length) {
    error.value = '请先选择需要导出的记录'
    return
  }
  const prefix = resource === 'assets' ? 'assets-selected' : 'middleware-selected'
  downloadText(`${prefix}-${new Date().toISOString().slice(0, 10)}.csv`, toCsv(items, bulkSchemas[resource]))
  notify(`已导出所选 ${items.length} 条记录`)
}

async function deleteSelected(resource, selection) {
  const items = [...selection.value]
  if (!items.length) return
  const assets = resource === 'assets'
  if (assets && !items.every(canDeleteAsset)) {
    error.value = '所选资产包含无删除权限的记录，无法批量删除'
    return
  }
  if (!assets && (!canWriteAssets.value || items.some((item) => isSampleRecord(item)))) {
    error.value = '当前角色或所选样例记录不支持批量删除'
    return
  }
  const label = assets ? '资产' : '实例'
  const busy = assets ? assetBulkBusy : middlewareBulkBusy
  await requestConfirm({
    title: `批量删除${label}`,
    message: `此操作不可恢复，将尝试删除所选的 ${items.length} 条${label}记录。关联凭据也会一并清理。`,
    target: items.slice(0, 5).map((item) => assets ? item.assetNo : item.name).join('、') + (items.length > 5 ? ` 等共 ${items.length} 项` : ''),
    confirmLabel: `确认删除 ${items.length} 项`
  }, async () => {
    busy.value = true
    error.value = ''
    const deleted = []
    const failures = []
    let cursor = 0
    const worker = async () => {
      while (cursor < items.length) {
        const item = items[cursor++]
        try {
          await api(`/${resource}/${item.id}`, { method: 'DELETE' })
          deleted.push(item)
        } catch (err) {
          failures.push({ item, message: err.message })
        }
      }
    }
    try {
      await Promise.all(Array.from({ length: Math.min(5, items.length) }, worker))
      const deletedIds = new Set(deleted.map((item) => item.id))
      selection.value = selection.value.filter((item) => !deletedIds.has(item.id))
      if (assets) {
        state.assets = state.assets.filter((item) => !deletedIds.has(item.id))
        if (selectedAsset.value && deletedIds.has(selectedAsset.value.id)) selectedAsset.value = null
        await Promise.all([loadAssetPage(), loadDashboard()])
      } else {
        state.middleware = state.middleware.filter((item) => !deletedIds.has(item.id))
        if (selectedMiddleware.value && deletedIds.has(selectedMiddleware.value.id)) {
          selectedMiddleware.value = null
          resetMiddlewareCredentialState()
        }
        await Promise.all([loadMiddlewarePage(), loadDashboard()])
      }
      if (failures.length) {
        error.value = `已删除 ${deleted.length} 项，${failures.length} 项失败：${failures.slice(0, 5).map(({ item, message }) => `${assets ? item.assetNo : item.name}：${message}`).join('；')}`
      } else {
        notify(`已批量删除 ${deleted.length} 条${label}记录`)
      }
    } finally {
      busy.value = false
    }
  })
}

function deleteSelectedAssets() {
  return deleteSelected('assets', selectedAssetItems)
}

function deleteSelectedMiddleware() {
  return deleteSelected('middleware', selectedMiddlewareItems)
}

function importAssetsFile(file) {
  return importCsvFile('assets', file)
}

function importMiddlewareFile(file) {
  return importCsvFile('middleware', file)
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
	const assignee = state.userDirectory.find(user => user.id === Number(newTask.assigneeUserId))
	const payload = { ...newTask, dueAt: toAPITimestamp(newTask.dueAt), assigneeUserId: newTask.assigneeUserId ? Number(newTask.assigneeUserId) : null, assignee: assignee?.displayName || newTask.assignee }
    const method = newTask.id ? 'PUT' : 'POST'
    const path = newTask.id ? `/tasks/${newTask.id}` : '/tasks'
    const item = await api(path, { method, body: JSON.stringify(payload) })
    const existingIndex = state.tasks.findIndex((entry) => entry.id === item.id)
    if (existingIndex >= 0) {
      state.tasks.splice(existingIndex, 1, item)
    } else {
      state.tasks.unshift(item)
    }
    selectedTask.value = item
    closeTaskForm()
    await Promise.all([loadTaskPage(), loadDashboard()])
    selectedTask.value = state.tasks.find(entry => entry.id === item.id) || item
    notify(method === 'PUT' ? '任务修改已保存' : '任务已创建')
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
  Object.assign(newTask, { ...task, assigneeUserId: task.assigneeUserId || '', dueAt: toDateTimeLocal(task.dueAt) })
  taskFormOpen.value = true
}

function resetTaskForm() {
  Object.assign(newTask, { id: null, title: '', type: '任务', assignee: '', assigneeUserId: '', status: '待处理', dueAt: '', description: '' })
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
      await Promise.all([loadTaskPage(), loadDashboard()])
      notify('任务已删除')
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
    await Promise.all([loadTaskPage(), loadDashboard()])
    selectedTask.value = state.tasks.find(entry => entry.id === task.id) || null
    notify(`任务状态已更新为“${status}”`, 'info')
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
	const owner = state.userDirectory.find(user => user.id === Number(newIncident.ownerUserId))
	const payload = { ...newIncident, startedAt: toAPITimestamp(newIncident.startedAt), recoveredAt: toAPITimestamp(newIncident.recoveredAt), ownerUserId: newIncident.ownerUserId ? Number(newIncident.ownerUserId) : null, owner: owner?.displayName || newIncident.owner }
    const method = newIncident.id ? 'PUT' : 'POST'
    const path = newIncident.id ? `/incidents/${newIncident.id}` : '/incidents'
    const item = await api(path, { method, body: JSON.stringify(payload) })
    const existingIndex = state.incidents.findIndex((entry) => entry.id === item.id)
    if (existingIndex >= 0) {
      state.incidents.splice(existingIndex, 1, item)
    } else {
      state.incidents.unshift(item)
    }
    selectedIncident.value = item
    closeIncidentForm()
    await Promise.all([loadIncidentPage(), loadDashboard()])
    selectedIncident.value = state.incidents.find(entry => entry.id === item.id) || item
    notify(method === 'PUT' ? '事件修改已保存' : '事件已创建')
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
  Object.assign(newIncident, {
    ...incident,
    ownerUserId: incident.ownerUserId || '',
    startedAt: toDateTimeLocal(incident.startedAt),
    recoveredAt: toDateTimeLocal(incident.recoveredAt)
  })
  incidentFormOpen.value = true
}

function resetIncidentForm() {
  Object.assign(newIncident, { id: null, title: '', level: 'P3', status: '新建', owner: '', ownerUserId: '', business: '', startedAt: '', recoveredAt: '', summary: '' })
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
      await Promise.all([loadIncidentPage(), loadDashboard()])
      notify('事件已删除')
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
    await Promise.all([loadIncidentPage(), loadDashboard()])
    selectedIncident.value = state.incidents.find(entry => entry.id === incident.id) || null
    notify(`事件状态已更新为“${status}”`, 'info')
  } catch (err) {
    incident.status = previous
    error.value = `更新事件状态失败：${err.message}`
  }
}

function resetCopilotProfile(provider = copilotProviders[1]) {
  Object.assign(copilotConfig, {
    id: null,
    name: '',
    provider: provider.id,
    endpoint: provider.id === 'local' ? '' : provider.endpoint,
    model: provider.id === 'local' ? '' : provider.models[0],
    apiKey: '',
    hasApiKey: false,
    localEndpoint: provider.id === 'local' ? provider.endpoint : '',
    localModel: provider.id === 'local' ? provider.models[0] : '',
    temperature: '0.2',
    maxTokens: '2048',
    enableAssetContext: true,
    enableIncidentContext: true,
    enableTaskContext: true,
    enableOncallContext: true,
    auditEnabled: true,
    isActive: false
  })
  resetCopilotConnection()
}

function applyCopilotConfig(config) {
  Object.assign(copilotConfig, {
    id: config.id,
    name: config.name || '',
    provider: config.provider || 'openai',
    endpoint: config.endpoint || '',
    model: config.model || '',
    apiKey: '',
    hasApiKey: Boolean(config.hasApiKey),
    localEndpoint: config.localEndpoint || '',
    localModel: config.localModel || '',
    temperature: config.temperature || '0.2',
    maxTokens: config.maxTokens || '2048',
    enableAssetContext: Boolean(config.enableAssetContext),
    enableIncidentContext: Boolean(config.enableIncidentContext),
    enableTaskContext: Boolean(config.enableTaskContext),
    enableOncallContext: Boolean(config.enableOncallContext),
    auditEnabled: true,
    isActive: Boolean(config.isActive)
  })
  resetCopilotConnection()
}

function openCopilotCreate(provider = copilotProviders[1]) {
  resetCopilotProfile(provider)
  copilotEditorMode.value = 'create'
}

function editCopilotProfile(profile) {
  applyCopilotConfig(profile)
  copilotEditorMode.value = 'edit'
}

function closeCopilotEditor() {
  copilotEditorMode.value = 'closed'
  resetCopilotConnection()
}

function selectCopilotProvider(provider) {
  if (copilotEditorMode.value === 'closed') {
    openCopilotCreate(provider)
    return
  }
  const currentName = copilotConfig.name
  const currentID = copilotConfig.id
  const currentActive = copilotConfig.isActive
  resetCopilotProfile(provider)
  Object.assign(copilotConfig, { name: currentName, id: currentID, isActive: currentActive })
}

function selectCopilotProviderByID() {
  selectCopilotProvider(selectedCopilotProvider.value)
}

async function loadCopilotProfiles() {
  copilotProfiles.value = await api('/copilot/configs')
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
    if (result.ok) {
      error.value = ''
      notify('AI Copilot 连接测试通过')
    } else {
      error.value = `AI Copilot 连接测试未通过：${result.message}`
    }
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

async function testSavedCopilotProfile(profile) {
  if (!canManageUsers.value) return
  try {
    const result = await api(`/copilot/configs/${profile.id}/test-connection`, { method: 'POST' })
    if (result.ok) {
      error.value = ''
      notify(`${profile.name} 连接测试通过`)
    } else {
      error.value = `${profile.name} 连接测试未通过：${result.message}`
    }
  } catch (err) {
    error.value = `${profile.name} 连接测试失败：${err.message}`
  }
}

async function saveCopilotConfig() {
  if (!canManageUsers.value) {
    error.value = '只有超级管理员可以保存 AI Copilot 配置'
    return
  }
  try {
    const editing = copilotEditorMode.value === 'edit' && copilotConfig.id
    const saved = await api(editing ? `/copilot/configs/${copilotConfig.id}` : '/copilot/configs', {
      method: editing ? 'PUT' : 'POST',
      body: JSON.stringify(copilotConfig)
    })
    await loadCopilotProfiles()
    closeCopilotEditor()
    error.value = ''
    notify(saved.hasApiKey ? '模型配置已保存，API Key 已加密托管' : '模型配置已保存')
  } catch (err) {
    error.value = `模型配置保存失败：${err.message}`
  }
}

async function activateCopilotProfile(profile) {
  try {
    await api(`/copilot/configs/${profile.id}/activate`, { method: 'POST' })
    await loadCopilotProfiles()
    error.value = ''
    notify(`已启用模型配置“${profile.name}”`)
  } catch (err) {
    error.value = `启用模型配置失败：${err.message}`
  }
}

async function deleteCopilotProfile(profile) {
  if (profile.isActive) return
  await requestConfirm({
    title: '删除模型配置',
    message: '删除后无法恢复，已托管的 API Key 也会一并清理。',
    target: `${profile.name} · ${profile.model || profile.localModel}`,
    confirmLabel: '确认删除配置'
  }, async () => {
    await api(`/copilot/configs/${profile.id}`, { method: 'DELETE' })
    if (copilotConfig.id === profile.id) closeCopilotEditor()
    await loadCopilotProfiles()
    notify(`模型配置“${profile.name}”已删除`, 'info')
  })
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
    notify(method === 'PUT' ? '用户修改已保存' : '用户已创建')
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
      notify('用户已删除')
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

async function loadAuditEvents() {
  if (!canManageUsers.value) return
  try {
    state.auditEvents = await api('/audit-events?limit=100')
  } catch (err) {
    error.value = `加载操作审计失败：${err.message}`
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

const { schedule: reloadList } = useDeferredLoader(
  () => hasAppAccess.value,
  (message) => { error.value = message }
)

watch(
  () => [assetFilters.keyword, assetFilters.ips, assetFilters.type, assetFilters.environment, assetFilters.business, assetFilters.networkZone],
  () => {
    if (assetPager.page !== 1) assetPager.page = 1
    else reloadList(loadAssetPage, '资产列表', 250)
  }
)
watch(
  () => [middlewareFilters.keyword, middlewareFilters.ips, middlewareFilters.kind, middlewareFilters.environment, middlewareFilters.business, middlewareFilters.networkZone, middlewareFilters.status],
  () => {
    if (middlewarePager.page !== 1) middlewarePager.page = 1
    else reloadList(loadMiddlewarePage, '实例列表', 250)
  }
)
watch(() => assetPager.page, () => reloadList(loadAssetPage, '资产列表'))
watch(() => middlewarePager.page, () => reloadList(loadMiddlewarePage, '实例列表'))
watch(() => taskPager.page, () => reloadList(loadTaskPage, '任务列表'))
watch(() => incidentPager.page, () => reloadList(loadIncidentPage, '事件列表'))
watch(() => assetPager.pageSize, () => {
  if (assetPager.page !== 1) assetPager.page = 1
  else reloadList(loadAssetPage, '资产列表')
})
watch(() => middlewarePager.pageSize, () => {
  if (middlewarePager.page !== 1) middlewarePager.page = 1
  else reloadList(loadMiddlewarePage, '实例列表')
})
watch(() => taskPager.pageSize, () => {
  if (taskPager.page !== 1) taskPager.page = 1
  else reloadList(loadTaskPage, '任务列表')
})
watch(() => incidentPager.pageSize, () => {
  if (incidentPager.page !== 1) incidentPager.page = 1
  else reloadList(loadIncidentPage, '事件列表')
})

onMounted(() => {
  loadAll()
})

onUnmounted(() => {
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
          :bulk-busy="assetBulkBusy"
          :can-bulk-delete="canBulkDeleteAssets"
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
          :selected-items="selectedAssetItems"
          :spec="assetSpec"
          :total="listMeta.assets.total"
          @choose="chooseAsset"
          @close-form="closeAssetForm"
          @delete="deleteAsset"
          @edit="editAsset"
          @export-item="exportAsset"
          @import-file="importAssetsFile"
          @export-selected="exportSelected('assets', selectedAssetItems)"
          @delete-selected="deleteSelectedAssets"
          @clear-selection="clearAssetSelection"
          @toggle-select="toggleAssetSelection"
          @select-page="selectAssetPage"
          @download-template="downloadBulkTemplate('assets')"
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
          :bulk-busy="middlewareBulkBusy"
          :can-bulk-delete="canBulkDeleteMiddleware"
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
          :selected-items="selectedMiddlewareItems"
          :total="listMeta.middleware.total"
          @choose="chooseMiddleware"
          @close-form="closeMiddlewareForm"
          @delete="deleteMiddleware"
          @edit="editMiddleware"
          @export-item="exportMiddleware"
          @import-file="importMiddlewareFile"
          @export-selected="exportSelected('middleware', selectedMiddlewareItems)"
          @delete-selected="deleteSelectedMiddleware"
          @clear-selection="clearMiddlewareSelection"
          @toggle-select="toggleMiddlewareSelection"
          @select-page="selectMiddlewarePage"
          @download-template="downloadBulkTemplate('middleware')"
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
          :active-task-count="dashboardTaskCount"
          :current-task="currentTask"
          :form="newTask"
          :form-open="taskFormOpen"
          :is-sample="isSampleRecord"
          :page="taskPager.page"
          :page-count="taskPageCount"
          :page-size="taskPager.pageSize"
          :paged-tasks="pagedTasks"
          :status-counts="taskStatusCounts"
          :total="listMeta.tasks.total"
          :users="state.userDirectory"
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
          :active-incident-count="dashboardIncidentCount"
          :current-incident="currentIncident"
          :form="newIncident"
          :form-open="incidentFormOpen"
          :is-sample="isSampleRecord"
          :level-counts="incidentLevelCounts"
          :page="incidentPager.page"
          :page-count="incidentPageCount"
          :page-size="incidentPager.pageSize"
          :paged-incidents="pagedIncidents"
          :total="listMeta.incidents.total"
          :users="state.userDirectory"
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

        <AuditView
          v-if="activeView === 'audit'"
          :can-manage="canManageUsers"
          :events="state.auditEvents"
          @refresh="loadAuditEvents"
        />

        <CopilotSettingsView
          v-if="activeView === 'copilot-settings'"
          :can-manage="canManageUsers"
          :config="copilotConfig"
          :connection="copilotConnection"
          :editor-open="copilotEditorMode !== 'closed'"
          :editor-mode="copilotEditorMode"
          :profiles="copilotProfiles"
          :providers="copilotProviders"
          :selected-provider="selectedCopilotProvider"
          @activate="activateCopilotProfile"
          @close-editor="closeCopilotEditor"
          @delete="deleteCopilotProfile"
          @edit="editCopilotProfile"
          @open-create="openCopilotCreate()"
          @save="saveCopilotConfig"
          @select-provider="selectCopilotProvider"
          @select-provider-id="selectCopilotProviderByID"
          @test="testCopilotConnection"
          @test-profile="testSavedCopilotProfile"
        />

      </template>
    </main>

    <CopilotWidget
      v-if="hasAppAccess"
      v-model:question="copilotQuestion"
      :open="copilotOpen"
      :expanded="copilotExpanded"
      :messages="copilotMessages"
      :busy="copilotBusy"
      @hide="hideCopilot"
      @toggle-size="toggleCopilotSize"
      @send="askCopilot"
    />
    <ToastStack :messages="toastMessages" @dismiss="dismissToast" />
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
