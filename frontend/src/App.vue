<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { api, clearToken, getToken, login as loginApi } from './api'

const routeViews = ['dashboard', 'cmdb', 'middleware', 'oncall', 'tasks', 'incidents', 'permissions', 'copilot-settings']
const permissionTabs = ['users', 'resources']
const oncallTabs = ['overview', 'calendar', 'handover', 'takeover']

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
const loading = ref(false)
const error = ref('')
const selectedAsset = ref(null)
const selectedMiddleware = ref(null)
const selectedTask = ref(null)
const selectedIncident = ref(null)
const assetFormOpen = ref(false)
const middlewareFormOpen = ref(false)
const taskFormOpen = ref(false)
const incidentFormOpen = ref(false)
const oncallFormOpen = ref(false)
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

const sampleAssets = [
  {
    id: 'sample-asset-1',
    assetNo: 'ASSET-DEMO-0001',
    type: '物理机',
    vendor: 'Dell',
    cpuArch: 'x86_64',
    sn: 'DEMO-SN-001',
    location: 'A 区 03 柜',
    business: '支付服务',
    ipv4: '10.20.3.11',
    ipv6: '',
    environment: '生产',
    os: 'Ubuntu 22.04',
    hostname: 'pay-core-01',
    networkZone: 'prod-app',
    cpu: '16C',
    memory: '64GB',
    disk: '1TB',
    deploymentInfo: '支付核心 API',
    owner: '李明',
    status: '运行中',
    connectedStatus: '已并网',
    hostMachine: '',
    __sample: true
  },
  {
    id: 'sample-asset-2',
    assetNo: 'ASSET-DEMO-0002',
    type: '虚拟机',
    vendor: 'VMware',
    cpuArch: 'x86_64',
    sn: '',
    location: '',
    business: '订单服务',
    ipv4: '10.20.5.21',
    ipv6: '',
    environment: '生产',
    os: 'Rocky Linux 9',
    hostname: 'order-worker-02',
    networkZone: 'prod-worker',
    cpu: '8C',
    memory: '32GB',
    disk: '500GB',
    deploymentInfo: '订单异步处理',
    owner: '王敏',
    status: '维护中',
    connectedStatus: '已并网',
    hostMachine: 'hv-prod-06',
    __sample: true
  }
]

const sampleMiddleware = [
  { id: 'sample-mw-1', name: 'pay-mysql-primary', kind: 'MySQL', version: '8.0', environment: '生产', networkZone: 'prod-db', endpoint: '10.20.3.16:3306', business: '支付服务', owner: '陈浩', status: '运行中', assetId: '', __sample: true },
  { id: 'sample-mw-2', name: 'pay-cache-cluster', kind: 'Redis', version: '7.2', environment: '生产', networkZone: 'prod-cache', endpoint: '10.20.4.18:6379', business: '支付服务', owner: '赵晨', status: '运行中', assetId: '', __sample: true },
  { id: 'sample-mw-3', name: 'gateway-nginx', kind: 'Nginx', version: '1.26', environment: '生产', networkZone: 'dmz', endpoint: '10.10.1.10:443', business: '网关', owner: '刘洋', status: '运行中', assetId: '', __sample: true }
]

const sampleOncalls = [
  { id: 'sample-oncall-1', ruleType: 'daily', date: '今天', week: '', primary: '李明', backup: '王敏', swapFrom: '', swapTo: '', notes: '覆盖生产与核心数据库', __sample: true },
  { id: 'sample-oncall-2', ruleType: 'weekly', date: '', week: '本周', primary: '赵晨', backup: '陈浩', swapFrom: '刘洋', swapTo: '赵晨', notes: '换班已确认', __sample: true }
]

const dutySections = [
  { id: 'overview', label: '概览' },
  { id: 'calendar', label: '排班日历' },
  { id: 'schedules', label: '排班配置' },
  { id: 'roster', label: '值班列表' },
  { id: 'handover', label: '交接班日志' },
  { id: 'escalation', label: '升级策略' }
]

const dutyTeams = ref([
  { label: '全部团队', count: 19 },
  { label: '基础运维组', count: 8 },
  { label: '网络运维组', count: 5 },
  { label: '应用支持组', count: 6 }
])

const dutyPeopleOptions = ['张伟', '李娜', '王强', '刘芳', '陈明', '赵丽', '孙磊', '周杰']
const dutySection = ref('overview')
const dutyTeamFilter = ref('全部团队')
const dutyGlobalSearch = ref('')
const dutyRosterSearch = ref('')
const dutyCalendarMonth = ref(new Date(2026, 5, 1))
const dutyToasts = ref([])
const dutyInlineAlert = ref(null)
const dutyAssignModalOpen = ref(false)
const dutyScheduleModalOpen = ref(false)
const dutyHandoverModalOpen = ref(false)
const dutyEscalationModalOpen = ref(false)
const dutyUserModalOpen = ref(false)
const dutyTeamModalOpen = ref(false)
const dutyTeamDrafts = ref([])
const dutySelectedDate = ref('2026-06-10')
const dutyScheduleModalMode = ref('create')
const dutyEditingScheduleId = ref(null)
const dutyAssignForm = reactive({ date: '2026-06-10', type: 'primary', person: '张伟', note: '', rosterId: null })
const dutyScheduleForm = reactive({ name: '新值班排班', team: '基础运维组', rotation: 'weekly', time: '08:00-20:00', members: ['张伟', '李娜', '陈明', '刘芳'] })
const dutyHandoverForm = reactive({ from: '张伟', to: '李娜', content: '', complete: true })
const dutyEscalationForm = reactive({ name: 'P1 严重告警升级流程', team: '基础运维组', severity: 'P1' })
const dutyUserForm = reactive({ userId: '', team: '基础运维组', role: '运维工程师', next: '待安排' })
const dutyEscalationLevels = ref([
  { level: 1, target: '主值班', delay: '立即通知', channel: '电话 / 短信 / IM' },
  { level: 2, target: '备份值班', delay: '5分钟后升级', channel: '电话 / IM' },
  { level: 3, target: '团队负责人', delay: '15分钟后升级', channel: '电话 / 短信' }
])

const dutyCurrentPeople = ref([
  { id: 1, name: '张伟', role: '主值班', team: '基础运维组', avatar: '张', since: '08:00', until: '20:00', phone: '138****1234', status: '在线' },
  { id: 2, name: '李娜', role: '备份值班', team: '基础运维组', avatar: '李', since: '08:00', until: '20:00', phone: '138****5678', status: '在线' },
  { id: 3, name: '王强', role: '主值班', team: '网络运维组', avatar: '王', since: '00:00', until: '24:00', phone: '139****9999', status: '在线' }
])

const dutySchedules = ref([
  { id: 1, name: '基础运维日常值班', team: '基础运维组', rotation: 'weekly', time: '08:00-20:00', members: ['张伟', '李娜', '陈明', '刘芳'], active: true },
  { id: 2, name: '网络监控夜间值班', team: '网络运维组', rotation: 'daily', time: '20:00-08:00', members: ['王强', '赵丽'], active: true },
  { id: 3, name: '应用支持周末值班', team: '应用支持组', rotation: 'weekly', time: '00:00-24:00', members: ['孙磊', '周杰'], active: false }
])

const dutyRoster = ref([
  { id: 1, name: '张伟', team: '基础运维组', role: 'SRE', count: 8, next: '2026-06-11', status: '值班中' },
  { id: 2, name: '李娜', team: '基础运维组', role: 'SRE', count: 7, next: '2026-06-12', status: '值班中' },
  { id: 3, name: '王强', team: '网络运维组', role: '网络工程师', count: 10, next: '2026-06-13', status: '值班中' },
  { id: 4, name: '陈明', team: '基础运维组', role: '运维工程师', count: 6, next: '2026-06-14', status: '空闲' },
  { id: 5, name: '刘芳', team: '基础运维组', role: 'DBA', count: 5, next: '2026-06-15', status: '空闲' },
  { id: 6, name: '赵丽', team: '网络运维组', role: '网络工程师', count: 8, next: '2026-06-16', status: '空闲' },
  { id: 7, name: '孙磊', team: '应用支持组', role: '开发工程师', count: 4, next: '2026-06-17', status: '空闲' },
  { id: 8, name: '周杰', team: '应用支持组', role: 'SRE', count: 5, next: '2026-06-18', status: '空闲' }
])

refreshDutyTeamCounts()

const dutyHandovers = ref([
  { id: 1, from: '张伟', to: '李娜', time: '2026-06-10 08:00', content: '生产环境运行正常，Redis 缓存集群有轻微延迟但不影响业务。P1 告警已处理完毕。', complete: true },
  { id: 2, from: '王强', to: '赵丽', time: '2026-06-09 20:00', content: '网络监控：核心交换机 CPU 使用率正常，防火墙策略变更已生效。', complete: true },
  { id: 3, from: '刘芳', to: '张伟', time: '2026-06-03 00:00', content: 'Kubernetes 集群升级完成，监控告警阈值调整。', complete: true }
])

const dutyAssignments = reactive({
  '2026-06-10': { primary: '张伟', backup: '李娜' },
  '2026-06-11': { primary: '陈明', backup: '刘芳' },
  '2026-06-12': { primary: '李娜', backup: '张伟' },
  '2026-06-13': { primary: '王强', backup: '赵丽' },
  '2026-06-14': { primary: '孙磊', backup: '周杰' },
  '2026-06-15': { primary: '刘芳', backup: '陈明' },
  '2026-06-16': { primary: '赵丽', backup: '王强' },
  '2026-06-17': { primary: '周杰', backup: '孙磊' },
  '2026-06-18': { primary: '张伟', backup: '李娜' },
  '2026-06-19': { primary: '陈明', backup: '刘芳' }
})

const sampleTasks = [
  { id: 'sample-task-1', title: '核心数据库巡检结果确认', type: '任务', assignee: 'SRE', status: '待确认', dueAt: '今天 18:00', description: '确认慢查询和备份状态', __sample: true },
  { id: 'sample-task-2', title: '支付服务异常峰值协同处理', type: '事件任务', assignee: '运维工程师 / 研发', status: '处理中', dueAt: '剩余 42 分钟', description: '跟进 P1 事件关联任务', __sample: true },
  { id: 'sample-task-3', title: '新增云主机归属确认', type: '资产', assignee: '运维工程师', status: '待处理', dueAt: '明天 12:00', description: '补齐业务归属与负责人', __sample: true }
]

const sampleIncidents = [
  { id: 'sample-incident-1', title: '支付服务异常峰值', level: 'P1', status: '处理中', owner: '李明', business: '支付服务', startedAt: '09:18', recoveredAt: '', summary: '接口错误率升高，已关联支付服务资产与 MySQL 主库', __sample: true },
  { id: 'sample-incident-2', title: '缓存集群连接抖动', level: 'P2', status: '已恢复', owner: '赵晨', business: '支付服务', startedAt: '08:42', recoveredAt: '09:05', summary: 'Redis 节点短时抖动，待关闭确认', __sample: true },
  { id: 'sample-incident-3', title: '网关证书到期预警', level: 'P3', status: '新建', owner: '刘洋', business: '网关', startedAt: '10:12', recoveredAt: '', summary: 'Nginx 网关证书剩余 7 天', __sample: true }
]

const sampleUsers = [
  { id: 'sample-user-1', username: 'admin', displayName: '超级管理员', roles: ['super_admin'], mustChangePassword: false, __sample: true },
  { id: 'sample-user-2', username: 'ops-demo', displayName: '运维工程师', roles: ['ops_engineer'], mustChangePassword: true, __sample: true }
]

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
  connectedStatus: '已并网',
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
const newOncall = reactive({ id: null, ruleType: 'daily', date: '', week: '', primary: '', backup: '', swapFrom: '', swapTo: '', notes: '' })
const oncallFormMode = ref('calendar')
const oncallBatch = reactive({ startDate: '', endDate: '', primary: '', backup: '', notes: '' })
const oncallWorkspaceTab = ref('overview')
const oncallTakeoverConfirmed = ref(false)
const oncallHandoverConfirmed = ref(false)
const newUser = reactive({ id: null, username: '', displayName: '', password: '', mustChangePassword: true, role: 'ops_engineer' })
const copilotConfig = reactive({
  provider: 'openai',
  model: 'gpt-4.1',
  endpoint: 'https://api.openai.com/v1',
  localEndpoint: 'http://localhost:11434',
  localModel: 'qwen2.5:7b',
  apiKey: '',
  temperature: '0.2',
  maxTokens: '4096',
  enableAssetContext: true,
  enableIncidentContext: true,
  enableTaskContext: true,
  enableOncallContext: true,
  auditEnabled: true
})
const assetFilters = reactive({ keyword: '', type: '', environment: '', business: '', networkZone: '', advanced: false })
const middlewareFilters = reactive({ keyword: '', kind: '', environment: '', business: '', networkZone: '', status: '', advanced: false })
const assetPager = reactive({ page: 1, pageSize: 10 })
const middlewarePager = reactive({ page: 1, pageSize: 10 })

const menu = [
  { id: 'dashboard', label: '首页仪表盘', icon: 'dashboard', enabled: true },
  {
    label: '资产管理',
    icon: 'asset',
    children: [
      { id: 'cmdb', label: '资产台账（CMDB）', icon: 'cmdb', enabled: true },
      { id: 'middleware', label: '中间件与数据库', icon: 'database', enabled: true },
      { label: '云资源管理', icon: 'cloud', enabled: false },
      { label: '容器平台', icon: 'container', enabled: false }
    ]
  },
  {
    label: '协同与事件响应',
    icon: 'collab',
    children: [
      { id: 'oncall', label: '值班管理', icon: 'calendar', enabled: true },
      { id: 'tasks', label: '任务跟踪', icon: 'task', enabled: true },
      { id: 'incidents', label: '事件管理', icon: 'incident', enabled: true }
    ]
  },
  {
    label: '权限管理',
    icon: 'shield',
    children: [
      { id: 'permissions', label: '用户与角色', icon: 'users', enabled: true, permissionTab: 'users' },
      { id: 'permissions', label: '菜单与资源权限', icon: 'lock', enabled: true, permissionTab: 'resources' },
      { label: '审批与操作审计', icon: 'audit', enabled: false },
      { label: 'SSO/LDAP', icon: 'key', enabled: false }
    ]
  },
  {
    label: '系统配置',
    icon: 'settings',
    children: [
      { id: 'copilot-settings', label: 'AI Copilot 配置', icon: 'bot', enabled: true },
      { label: '通知渠道', icon: 'bell', enabled: false },
      { label: '审计策略', icon: 'audit', enabled: false }
    ]
  },
  { label: 'SRE 管理', icon: 'sre', enabled: false },
  { label: '可观测性', icon: 'eye', enabled: false },
  { label: '发布管理', icon: 'deploy', enabled: false },
  { label: '混沌工程', icon: 'chaos', enabled: false },
  { label: '知识库', icon: 'book', enabled: false },
  { label: '容量规划 & FinOps', icon: 'finops', enabled: false },
  { label: '智能自治闭环', icon: 'bot', enabled: false },
  { label: '变更 & 自动化', icon: 'change', enabled: false }
]

const activeTitle = computed(() => {
  const labels = {
    dashboard: '首页健康总览',
    cmdb: '资产台账（CMDB）',
    middleware: '中间件与数据库',
    oncall: '值班管理',
    tasks: '任务跟踪',
    incidents: '事件管理',
    permissions: permissionTab.value === 'resources' ? '菜单与资源权限' : '用户与角色',
    'copilot-settings': 'AI Copilot 配置'
  }
  return labels[activeView.value]
})

const activeBreadcrumb = computed(() => {
  const labels = {
    dashboard: '工作台',
    cmdb: '资产管理',
    middleware: '资产管理',
    oncall: '协同与事件响应',
    tasks: '协同与事件响应',
    incidents: '协同与事件响应',
    permissions: '权限管理',
    'copilot-settings': '系统配置'
  }
  return labels[activeView.value] || '工作台'
})

const controlPrinciple = '首页看健康，异常看影响，告警看根因，处置看流程，复盘看改进，AI 贯穿查询、分析、建议和自动化。'
const activeSubtitle = computed(() => {
  const descriptions = {
    dashboard: '围绕业务连续性汇总健康、影响、待办和事件风险。',
    cmdb: '维护服务器资产、部署信息、网络区域和受控登录信息。',
    middleware: '管理数据库、中间件与组件实例，补齐部署位置和影响范围。',
    oncall: '确认主值、备值、轮换规则和换班记录。',
    tasks: '跟踪派发、处理、确认、完成和关闭的任务闭环。',
    incidents: '按 P1-P4 管理影响、状态、恢复和复盘动作。',
    permissions: permissionTab.value === 'resources'
      ? '查看一期菜单入口、业务资源与角色授权边界。'
      : '管理一期角色、账号密码登录和敏感凭据查看边界。',
    'copilot-settings': '配置 AI Copilot 的模型来源、上下文权限、审计和受控自动化边界。'
  }
  return descriptions[activeView.value] || controlPrinciple
})

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

const weekdayLabels = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
const rosterFallback = ['李明', '王敏', '赵晨', '陈浩', '刘洋', '周杰', '孙磊']

const displayAssets = computed(() => state.assets.length ? state.assets : sampleAssets)
const displayMiddleware = computed(() => state.middleware.length ? state.middleware : sampleMiddleware)
const displayOncalls = computed(() => state.oncalls.length ? state.oncalls : sampleOncalls)
const displayTasks = computed(() => state.tasks.length ? state.tasks : sampleTasks)
const displayIncidents = computed(() => state.incidents.length ? state.incidents : sampleIncidents)
const displayUsers = computed(() => state.users.length ? state.users : sampleUsers)
const taskStatusCounts = computed(() => countBy(displayTasks.value, 'status', ['待处理', '处理中', '待确认', '已完成', '已关闭']))
const incidentLevelCounts = computed(() => countBy(displayIncidents.value, 'level', ['P1', 'P2', 'P3', 'P4']))
const incidentStatusCounts = computed(() => countBy(displayIncidents.value, 'status', ['新建', '处理中', '已恢复', '已关闭']))
const activeTasks = computed(() => displayTasks.value.filter((item) => !['已完成', '已关闭'].includes(item.status)))
const activeIncidents = computed(() => displayIncidents.value.filter((item) => item.status !== '已关闭'))
const todayOncall = computed(() => displayOncalls.value[0] || {})
const oncallSwapCount = computed(() => displayOncalls.value.filter((item) => item.swapFrom || item.swapTo).length)
const currentTask = computed(() => selectedTask.value || null)
const currentIncident = computed(() => selectedIncident.value || null)
const dashboardAssetCount = computed(() => state.dashboard.assetCount || displayAssets.value.length + displayMiddleware.value.length)
const dashboardOncallCount = computed(() => state.dashboard.todayOnCallCount || displayOncalls.value.length)
const dashboardTaskCount = computed(() => state.dashboard.activeTaskCount || activeTasks.value.length)
const dashboardIncidentCount = computed(() => state.dashboard.activeIncidentCount || activeIncidents.value.length)
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
const oncallCalendarDays = computed(() => {
  const base = new Date()
  base.setHours(0, 0, 0, 0)
  return Array.from({ length: 14 }, (_, index) => {
    const date = new Date(base)
    date.setDate(base.getDate() + index)
    const key = formatLocalDate(date)
    const record = displayOncalls.value.find((item) => item.date === key || (index === 0 && item.date === '今天') || (index < 7 && item.week === '本周'))
    const fallbackPrimary = rosterFallback[index % rosterFallback.length]
    const fallbackBackup = rosterFallback[(index + 1) % rosterFallback.length]
    return {
      key,
      day: date.getDate(),
      month: date.getMonth() + 1,
      weekday: weekdayLabels[date.getDay()],
      primary: record?.primary || fallbackPrimary,
      backup: record?.backup || fallbackBackup,
      notes: record?.notes || (date.getDay() === 0 || date.getDay() === 6 ? '节假日值守' : '生产巡检'),
      isToday: index === 0,
      isWeekend: date.getDay() === 0 || date.getDay() === 6,
      hasSwap: Boolean(record?.swapFrom || record?.swapTo),
      source: record
    }
  })
})
const oncallHandoverItems = computed(() => [
  {
    title: '活跃事件',
    value: activeIncidents.value.length,
    detail: topItems(activeIncidents.value, 2).join('、') || '暂无未关闭事件',
    tone: activeIncidents.value.some((item) => item.level === 'P1') ? 'danger' : 'info'
  },
  {
    title: '待跟进任务',
    value: activeTasks.value.length,
    detail: topItems(activeTasks.value, 2).join('、') || '暂无待处理任务',
    tone: activeTasks.value.length ? 'warning' : 'info'
  },
  {
    title: '换班记录',
    value: oncallSwapCount.value,
    detail: oncallSwapCount.value ? '已记录换班，需确认交接信息' : '暂无换班待确认',
    tone: oncallSwapCount.value ? 'warning' : 'success'
  }
])
const dutyFilteredCurrent = computed(() => dutyTeamFilter.value === '全部团队'
  ? dutyCurrentPeople.value
  : dutyCurrentPeople.value.filter((person) => person.team === dutyTeamFilter.value))
const dutyFilteredRoster = computed(() => dutyRoster.value.filter((person) => {
  const matchesTeam = dutyTeamFilter.value === '全部团队' || person.team === dutyTeamFilter.value
  const keyword = dutyRosterSearch.value.trim().toLowerCase()
  const matchesKeyword = !keyword || [person.name, person.team, person.role, person.status].some((value) => String(value).toLowerCase().includes(keyword))
  return matchesTeam && matchesKeyword
}))
const dutyUpcomingAssignments = computed(() => Object.entries(dutyAssignments)
  .filter(([date]) => date > '2026-06-10')
  .sort(([a], [b]) => a.localeCompare(b))
  .slice(0, 3)
  .map(([date, value]) => ({ date, ...value })))
const dutyCalendarMonthLabel = computed(() => `${dutyCalendarMonth.value.getFullYear()}年${dutyCalendarMonth.value.getMonth() + 1}月`)
const dutyCalendarCells = computed(() => {
  const year = dutyCalendarMonth.value.getFullYear()
  const month = dutyCalendarMonth.value.getMonth()
  const first = new Date(year, month, 1)
  const last = new Date(year, month + 1, 0)
  const cells = Array.from({ length: first.getDay() }, (_, index) => ({ key: `blank-${index}`, blank: true }))
  for (let day = 1; day <= last.getDate(); day += 1) {
    const date = new Date(year, month, day)
    const key = formatLocalDate(date)
    const assignment = dutyAssignments[key]
    cells.push({
      key,
      day,
      isToday: key === '2026-06-10',
      isWeekend: date.getDay() === 0 || date.getDay() === 6,
      primary: assignment?.primary || '',
      backup: assignment?.backup || ''
    })
  }
  return cells
})
const dutyReports = computed(() => [
  { label: '值班均衡度', value: '82%', detail: '团队轮换分布稳定', width: '82%' },
  { label: '最长连续值班', value: '3天', detail: '未超过策略阈值', width: '60%' },
  { label: '替班次数', value: '5', detail: '本月换班记录', width: '45%' },
  { label: '用户满意度', value: '96%', detail: '事件响应评价', width: '96%' }
])
const dutySystemUserOptions = computed(() => displayUsers.value.map((user) => ({
  id: user.id,
  username: user.username,
  name: user.displayName || user.username,
  role: (user.roles || [])[0] || 'ops_engineer'
})))
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
const pagedAssets = computed(() => paginate(filteredAssets.value, assetPager))
const pagedMiddleware = computed(() => paginate(filteredMiddleware.value, middlewarePager))
const roleCards = [
  { code: 'super_admin', name: '超级管理员', icon: 'shield', tone: 'blue', desc: '拥有用户、角色、资产、协同事件、敏感凭据策略和系统配置全部权限。' },
  { code: 'ops_engineer', name: '运维工程师', icon: 'change', tone: 'green', desc: '可维护资产和实例信息，查看值班情况，处理任务，并跟进事件。' }
]
const permissionRows = [
  { scope: '用户与角色管理', admin: '全部', ops: '-', note: '仅超级管理员配置账号、角色和菜单' },
  { scope: '资产台账（CMDB）', admin: '全部', ops: '查看 / 修改', note: '运维工程师可查看与配置修改' },
  { scope: '中间件与数据库', admin: '全部', ops: '查看 / 修改', note: '实例基础信息与关联资产管理' },
  { scope: '资产登录信息查看', admin: '校验后查看', ops: '-', note: '查看前输入超级管理员设置的校验密码' },
  { scope: '值班情况', admin: '全部', ops: '查看', note: '运维工程师可查看值班安排' },
  { scope: '任务处理', admin: '全部', ops: '处理', note: '处理本人或分派任务' },
  { scope: '事件跟进', admin: '全部', ops: '跟进', note: '查看协同事件并更新进展' }
]

const menuPermissionRows = [
  { menu: '首页健康总览', stage: '一期启用', superAdmin: '可查看', opsEngineer: '可查看', note: '聚合健康、影响、值班和待办' },
  { menu: '资产台账（CMDB）', stage: '一期启用', superAdmin: '全部', opsEngineer: '查看 / 修改', note: '登录信息需额外校验权限' },
  { menu: '中间件与数据库', stage: '一期启用', superAdmin: '全部', opsEngineer: '查看 / 修改', note: '实例账号密码默认不可见' },
  { menu: '值班管理', stage: '一期启用', superAdmin: '全部', opsEngineer: '查看', note: '新增、编辑、删除仅超级管理员' },
  { menu: '任务跟踪', stage: '一期启用', superAdmin: '全部', opsEngineer: '处理', note: '支持状态流转与闭环跟进' },
  { menu: '事件管理', stage: '一期启用', superAdmin: '全部', opsEngineer: '跟进', note: '支持 P1-P4 与恢复关闭流转' },
  { menu: 'SSO/LDAP、审批审计、SRE、可观测性', stage: '灰度占位', superAdmin: '灰度', opsEngineer: '-', note: '只展示入口，不开放业务操作' }
]

const copilotProviders = [
  { id: 'local', name: '本地模型', badge: 'Local', endpoint: 'http://localhost:11434', models: ['qwen2.5:7b', 'deepseek-r1:7b', 'llama3.1:8b'], desc: '适合内网部署、敏感数据不出域和离线推理场景。' },
  { id: 'openai', name: 'OpenAI GPT', badge: 'GPT', endpoint: 'https://api.openai.com/v1', models: ['gpt-4.1', 'gpt-4o', 'o4-mini'], desc: '适合综合分析、复杂推理和通用 Copilot 能力。' },
  { id: 'anthropic', name: 'Anthropic Claude', badge: 'Claude', endpoint: 'https://api.anthropic.com', models: ['claude-3-7-sonnet', 'claude-3-5-sonnet'], desc: '适合长上下文总结、变更评审和事件复盘。' },
  { id: 'google', name: 'Google Gemini', badge: 'Gemini', endpoint: 'https://generativelanguage.googleapis.com', models: ['gemini-1.5-pro', 'gemini-1.5-flash'], desc: '适合多模态、知识检索和轻量分析场景。' },
  { id: 'compatible', name: 'OpenAI 兼容接口', badge: 'API', endpoint: 'https://llm.example.com/v1', models: ['custom-chat', 'ops-model'], desc: '适配私有云网关、代理平台或统一模型路由。' }
]

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

function setPage(pager, page, totalPages) {
  pager.page = Math.min(Math.max(page, 1), totalPages)
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

function syncRouteFromHash() {
  const next = parseRouteHash()
  if (activeView.value !== next.view || permissionTab.value !== next.permissionTab) {
    closeOpenEditors()
  }
  activeView.value = next.view
  permissionTab.value = next.permissionTab
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
  error.value = ''
}

function closeOpenEditors() {
  if (assetFormOpen.value) closeAssetForm()
  if (middlewareFormOpen.value) closeMiddlewareForm()
  if (taskFormOpen.value) closeTaskForm()
  if (incidentFormOpen.value) closeIncidentForm()
  if (oncallFormOpen.value) closeOncallForm()
  if (userFormOpen.value) closeUserForm()
}

function closeEditorOnFocusOut(event, closeFn) {
  const nextTarget = event.relatedTarget
  if (nextTarget && event.currentTarget.contains(nextTarget)) return
  window.requestAnimationFrame(() => {
    const active = document.activeElement
    if (active && event.currentTarget.contains(active)) return
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
        const [users, verification] = await Promise.all([
          api('/users'),
          api('/security/credential-verification')
        ])
        state.users = users
        credentialVerification.hasPassword = Boolean(verification.hasPassword)
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
  clearToken()
  auth.token = ''
  auth.user = null
  auth.username = ''
  auth.password = ''
  Object.assign(passwordInit, { currentPassword: '', newPassword: '', confirmPassword: '' })
}

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
    connectedStatus: '已并网',
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
  if (!asset || !window.confirm(`确认删除资产 ${asset.assetNo || asset.id}？`)) return
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
  if (!item || !window.confirm(`确认删除实例 ${item.name || item.id}？`)) return
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
  if (!task || !window.confirm(`确认删除任务 ${task.title || task.id}？`)) return
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
  if (!incident || !window.confirm(`确认删除事件 ${incident.title || incident.id}？`)) return
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

async function saveOncall() {
  error.value = ''
  if (!canWriteOncall.value) {
    error.value = '当前角色只能查看值班安排，不能修改值班记录'
    return
  }
  if (!newOncall.primary.trim()) {
    error.value = '请填写主值人员'
    return
  }
  if (newOncall.ruleType === 'daily' && !newOncall.date.trim()) {
    error.value = '按天值班需要填写日期'
    return
  }
  if (newOncall.ruleType === 'weekly' && !newOncall.week.trim()) {
    error.value = '按周值班需要填写周信息'
    return
  }
  try {
    const method = newOncall.id ? 'PUT' : 'POST'
    const path = newOncall.id ? `/oncall/${newOncall.id}` : '/oncall'
    const item = await api(path, { method, body: JSON.stringify(newOncall) })
    const existingIndex = state.oncalls.findIndex((entry) => entry.id === item.id)
    if (existingIndex >= 0) {
      state.oncalls.splice(existingIndex, 1, item)
    } else {
      state.oncalls.unshift(item)
    }
    closeOncallForm()
  } catch (err) {
    error.value = `保存值班失败：${err.message}`
  }
}

function parseLocalDate(value) {
  if (!value) return null
  const [year, month, day] = value.split('-').map(Number)
  if (!year || !month || !day) return null
  return new Date(year, month - 1, day)
}

function formatLocalDate(date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function dateRangeValues(startValue, endValue) {
  const start = parseLocalDate(startValue)
  const end = parseLocalDate(endValue || startValue)
  if (!start || !end || start > end) return []
  const days = []
  const cursor = new Date(start)
  while (cursor <= end && days.length < 31) {
    days.push(formatLocalDate(cursor))
    cursor.setDate(cursor.getDate() + 1)
  }
  return days
}

const oncallBatchPreview = computed(() => dateRangeValues(oncallBatch.startDate, oncallBatch.endDate))

async function saveOncallBatch() {
  error.value = ''
  if (!canWriteOncall.value) {
    error.value = '当前角色只能查看值班安排，不能修改值班记录'
    return
  }
  if (!oncallBatch.startDate || !oncallBatch.endDate) {
    error.value = '请填写批量排班的起止日期'
    return
  }
  const dates = dateRangeValues(oncallBatch.startDate, oncallBatch.endDate)
  if (!dates.length) {
    error.value = '排班日期范围不合法'
    return
  }
  const start = parseLocalDate(oncallBatch.startDate)
  const end = parseLocalDate(oncallBatch.endDate)
  if (start && end && (end - start) / 86400000 > 30) {
    error.value = '一次批量排班最多支持 31 天'
    return
  }
  if (!oncallBatch.primary.trim()) {
    error.value = '请填写主值人员'
    return
  }
  try {
    const created = []
    for (const date of dates) {
      const item = await api('/oncall', {
        method: 'POST',
        body: JSON.stringify({
          ruleType: 'daily',
          date,
          week: '',
          primary: oncallBatch.primary,
          backup: oncallBatch.backup,
          swapFrom: '',
          swapTo: '',
          notes: oncallBatch.notes
        })
      })
      created.push(item)
    }
    state.oncalls.unshift(...created.reverse())
    closeOncallForm()
  } catch (err) {
    error.value = `保存批量排班失败：${err.message}`
  }
}

function confirmOncallTakeover() {
  oncallTakeoverConfirmed.value = true
  error.value = ''
}

function confirmOncallHandover() {
  oncallHandoverConfirmed.value = true
  error.value = ''
}

function setOncallWorkspaceTab(tab) {
  if (oncallTabs.includes(tab)) {
    oncallWorkspaceTab.value = tab
  }
}

function openOncallForm() {
  if (!canWriteOncall.value) {
    error.value = '当前角色只能查看值班安排，不能新增值班记录'
    return
  }
  resetOncallForm()
  resetOncallBatch()
  oncallFormMode.value = 'calendar'
  oncallFormOpen.value = true
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
}

function saveCopilotConfig() {
  error.value = 'AI Copilot 配置已在当前前端会话中保存；后端持久化和密钥托管需在下一步接入。'
}

function closeOncallForm() {
  resetOncallForm()
  resetOncallBatch()
  oncallFormOpen.value = false
}

function editOncall(item) {
  if (!canWriteOncall.value) {
    error.value = '当前角色只能查看值班安排，不能修改值班记录'
    return
  }
  Object.assign(newOncall, { ...item })
  oncallFormMode.value = 'single'
  oncallFormOpen.value = true
}

function resetOncallForm() {
  Object.assign(newOncall, { id: null, ruleType: 'daily', date: '', week: '', primary: '', backup: '', swapFrom: '', swapTo: '', notes: '' })
}

function resetOncallBatch() {
  Object.assign(oncallBatch, { startDate: '', endDate: '', primary: '', backup: '', notes: '' })
}

async function deleteOncall(item) {
  if (!canWriteOncall.value) {
    error.value = '当前角色只能查看值班安排，不能删除值班记录'
    return
  }
  if (!item || !window.confirm(`确认删除值班记录 ${item.date || item.week || item.id}？`)) return
  error.value = ''
  try {
    await api(`/oncall/${item.id}`, { method: 'DELETE' })
    state.oncalls = state.oncalls.filter((entry) => entry.id !== item.id)
    if (newOncall.id === item.id) {
      closeOncallForm()
    }
  } catch (err) {
    error.value = `删除值班失败：${err.message}`
  }
}

function pushDutyToast(message, tone = 'success') {
  const toast = { id: Date.now() + Math.random(), message, tone }
  dutyToasts.value.push(toast)
  window.setTimeout(() => {
    dutyToasts.value = dutyToasts.value.filter((item) => item.id !== toast.id)
  }, 2600)
}

function setDutySection(section) {
  if (dutySections.some((item) => item.id === section)) {
    dutySection.value = section
  }
}

function setDutyTeam(team) {
  dutyTeamFilter.value = team
  pushDutyToast(`已切换到${team}`, 'info')
}

function refreshDutyTeamCounts() {
  const teamLabels = dutyTeams.value
    .filter((team) => team.label !== '全部团队')
    .map((team) => team.label)
  dutyTeams.value = [
    { label: '全部团队', count: dutyRoster.value.length },
    ...teamLabels.map((label) => ({
      label,
      count: dutyRoster.value.filter((person) => person.team === label).length
    }))
  ]
}

function openDutyTeamModal() {
  if (!canWriteOncall.value) {
    error.value = '当前角色只能查看值班安排，不能维护团队配置'
    return
  }
  refreshDutyTeamCounts()
  dutyTeamDrafts.value = dutyTeams.value
    .filter((team) => team.label !== '全部团队')
    .map((team) => ({ original: team.label, label: team.label, count: team.count }))
  dutyTeamModalOpen.value = true
}

function addDutyTeamDraft() {
  const index = dutyTeamDrafts.value.length + 1
  dutyTeamDrafts.value.push({ original: '', label: `新运维组 ${index}`, count: 0 })
}

function removeDutyTeamDraft(index) {
  const target = dutyTeamDrafts.value[index]
  if (!target) return
  if (Number(target.count || 0) > 0) {
    pushDutyToast('该团队仍有关联人员，先调整人员团队后再删除', 'warning')
    return
  }
  dutyTeamDrafts.value.splice(index, 1)
  pushDutyToast('空团队已移除', 'info')
}

function saveDutyTeams() {
  const labels = dutyTeamDrafts.value.map((team) => team.label.trim()).filter(Boolean)
  if (!labels.length) {
    pushDutyToast('至少保留一个团队', 'warning')
    return
  }
  if (new Set(labels).size !== labels.length) {
    pushDutyToast('团队名称不能重复', 'warning')
    return
  }

  dutyTeamDrafts.value.forEach((team) => {
    const nextLabel = team.label.trim()
    if (!team.original || team.original === nextLabel) return
    dutyRoster.value.forEach((person) => {
      if (person.team === team.original) person.team = nextLabel
    })
    dutyCurrentPeople.value.forEach((person) => {
      if (person.team === team.original) person.team = nextLabel
    })
    dutySchedules.value.forEach((schedule) => {
      if (schedule.team === team.original) schedule.team = nextLabel
    })
  })

  dutyTeams.value = [
    { label: '全部团队', count: dutyRoster.value.length },
    ...labels.map((label) => ({ label, count: 0 }))
  ]
  if (!dutyTeams.value.some((team) => team.label === dutyTeamFilter.value)) {
    dutyTeamFilter.value = '全部团队'
  }
  refreshDutyTeamCounts()
  dutyTeamModalOpen.value = false
  pushDutyToast('团队配置已更新')
}

function dutySearchSubmit() {
  const keyword = dutyGlobalSearch.value.trim()
  if (!keyword) return
  const person = dutyRoster.value.find((item) => item.name.includes(keyword))
  if (person) {
    dutyRosterSearch.value = keyword
    dutySection.value = 'roster'
    pushDutyToast(`已定位到 ${person.name}`, 'info')
    return
  }
  const date = Object.entries(dutyAssignments).find(([, value]) => value.primary.includes(keyword) || value.backup.includes(keyword))
  if (date) {
    dutySelectedDate.value = date[0]
    dutySection.value = 'calendar'
    pushDutyToast(`已定位到 ${date[0]} 的排班`, 'info')
    return
  }
  pushDutyToast('未找到匹配的值班人员或排班', 'warning')
}

function moveDutyMonth(offset) {
  const next = new Date(dutyCalendarMonth.value)
  next.setMonth(next.getMonth() + offset)
  dutyCalendarMonth.value = next
}

function resetDutyCalendar() {
  dutyCalendarMonth.value = new Date(2026, 5, 1)
  pushDutyToast('日历已重置到6月', 'info')
}

function openDutyAssignModal(date = dutySelectedDate.value, type = 'primary') {
  if (!canWriteOncall.value) {
    error.value = '当前角色只能查看值班安排，不能分配值班'
    return
  }
  const assignment = dutyAssignments[date] || {}
  Object.assign(dutyAssignForm, {
    date,
    type,
    person: type === 'primary' ? (assignment.primary || dutyPeopleOptions[0]) : (assignment.backup || dutyPeopleOptions[1]),
    note: '',
    rosterId: null
  })
  dutyAssignModalOpen.value = true
}

function confirmDutyAssign() {
  if (!dutyAssignForm.date || !dutyAssignForm.person) return
  const existing = dutyAssignments[dutyAssignForm.date] || { primary: '', backup: '' }
  dutyAssignments[dutyAssignForm.date] = {
    ...existing,
    [dutyAssignForm.type]: dutyAssignForm.person
  }
  const rosterItem = dutyRoster.value.find((person) => person.id === dutyAssignForm.rosterId || person.name === dutyAssignForm.person)
  if (rosterItem) {
    rosterItem.status = '值班中'
    rosterItem.next = dutyAssignForm.date
    rosterItem.count = Number(rosterItem.count || 0) + 1
    if (!dutyCurrentPeople.value.some((person) => person.name === rosterItem.name)) {
      dutyCurrentPeople.value.push({
        id: Date.now(),
        name: rosterItem.name,
        role: dutyAssignForm.type === 'primary' ? '主值班' : '备份值班',
        team: rosterItem.team,
        avatar: rosterItem.name.slice(0, 1),
        since: '08:00',
        until: '20:00',
        phone: '待补充',
        status: '在线'
      })
    }
    refreshDutyTeamCounts()
  }
  dutySelectedDate.value = dutyAssignForm.date
  dutyAssignModalOpen.value = false
  pushDutyToast(`${dutyAssignForm.person} 已分配为${dutyAssignForm.type === 'primary' ? '主值班' : '备份值班'}`)
}

function openCreateDutyScheduleModal() {
  if (!canWriteOncall.value) {
    error.value = '当前角色只能查看值班安排，不能创建排班模板'
    return
  }
  dutyScheduleModalMode.value = 'create'
  dutyEditingScheduleId.value = null
  Object.assign(dutyScheduleForm, { name: '新值班排班', team: '基础运维组', rotation: 'weekly', time: '08:00-20:00', members: ['张伟', '李娜', '陈明', '刘芳'] })
  dutyScheduleModalOpen.value = true
}

function saveDutyScheduleTemplate() {
  const payload = {
    name: dutyScheduleForm.name || '未命名排班',
    team: dutyScheduleForm.team,
    rotation: dutyScheduleForm.rotation,
    time: dutyScheduleForm.time || '08:00-20:00',
    members: dutyScheduleForm.members.length ? [...dutyScheduleForm.members] : [dutyPeopleOptions[0]],
    active: true
  }
  if (dutyScheduleModalMode.value === 'edit') {
    const target = dutySchedules.value.find((item) => item.id === dutyEditingScheduleId.value)
    if (target) {
      Object.assign(target, payload, { active: target.active })
    }
    pushDutyToast('排班模板已更新')
  } else {
    dutySchedules.value.unshift({ id: Date.now(), ...payload })
    pushDutyToast('新排班模板创建成功')
  }
  dutyScheduleModalOpen.value = false
}

function editDutySchedule(schedule) {
  dutyScheduleModalMode.value = 'edit'
  dutyEditingScheduleId.value = schedule.id
  Object.assign(dutyScheduleForm, {
    name: schedule.name,
    team: schedule.team,
    rotation: schedule.rotation,
    time: schedule.time,
    members: [...schedule.members]
  })
  dutyScheduleModalOpen.value = true
}

function toggleDutySchedule(schedule) {
  schedule.active = !schedule.active
  pushDutyToast(schedule.active ? '排班已启用' : '排班已停用', schedule.active ? 'success' : 'warning')
}

function deleteDutySchedule(schedule) {
  if (!window.confirm('确定删除此排班模板吗？')) return
  dutySchedules.value = dutySchedules.value.filter((item) => item.id !== schedule.id)
  pushDutyToast('排班模板已删除', 'warning')
}

function quickDutyTakeover() {
  const person = dutyRoster.value.find((item) => item.status === '空闲')
  if (!person) {
    pushDutyToast('暂无可接班人员', 'warning')
    return
  }
  dutyCurrentPeople.value = [
    ...dutyCurrentPeople.value,
    { id: Date.now(), name: person.name, role: '接班值班', team: person.team, avatar: person.name.slice(0, 1), since: '现在', until: '20:00', phone: '待补充', status: '在线' }
  ]
  person.status = '值班中'
  person.next = '2026-06-10'
  pushDutyToast(`${person.name} 已接班`)
}

function ackDutyAlert() {
  if (!dutyInlineAlert.value) return
  dutyInlineAlert.value.status = '已确认'
  pushDutyToast('告警已确认')
}

function escalateDutyAlert() {
  if (!dutyInlineAlert.value) return
  dutyInlineAlert.value.status = '已升级至备份值班'
  pushDutyToast('已升级到第二级（备份值班）', 'warning')
}

function quickAssignDutyPerson(person) {
  const targetDate = /^\d{4}-\d{2}-\d{2}$/.test(person.next) ? person.next : dutySelectedDate.value
  openDutyAssignModal(targetDate, 'primary')
  dutyAssignForm.person = person.name
  dutyAssignForm.rosterId = person.id
}

function openAddDutyPerson() {
  const firstUser = dutySystemUserOptions.value[0]
  Object.assign(dutyUserForm, {
    userId: firstUser?.id || '',
    team: '基础运维组',
    role: firstUser?.role === 'super_admin' ? 'SRE 负责人' : '运维工程师',
    next: '待安排'
  })
  dutyUserModalOpen.value = true
}

function saveDutyUser() {
  const selected = dutySystemUserOptions.value.find((item) => String(item.id) === String(dutyUserForm.userId))
  if (!selected) {
    pushDutyToast('请选择系统用户', 'warning')
    return
  }
  const exists = dutyRoster.value.some((item) => item.userId && String(item.userId) === String(selected.id))
  if (exists) {
    pushDutyToast('该系统用户已在值班列表中', 'warning')
    return
  }
  dutyRoster.value.push({
    id: Date.now(),
    userId: selected.id,
    username: selected.username,
    name: selected.name,
    team: dutyUserForm.team,
    role: dutyUserForm.role,
    count: 0,
    next: dutyUserForm.next || '待安排',
    status: '空闲'
  })
  refreshDutyTeamCounts()
  dutyUserModalOpen.value = false
  pushDutyToast('系统用户已加入值班列表')
}

function openDutyHandoverModal() {
  Object.assign(dutyHandoverForm, {
    from: dutyCurrentPeople.value[0]?.name || '张伟',
    to: dutyCurrentPeople.value[1]?.name || '李娜',
    content: '',
    complete: true
  })
  dutyHandoverModalOpen.value = true
}

function submitDutyHandover() {
  dutyHandovers.value.unshift({
    id: Date.now(),
    from: dutyHandoverForm.from,
    to: dutyHandoverForm.to,
    time: '2026-06-10 20:00',
    content: dutyHandoverForm.content || '系统运行正常，无未交接风险。',
    complete: dutyHandoverForm.complete
  })
  dutyHandoverModalOpen.value = false
  pushDutyToast('交接班记录已提交')
}

function saveDutyEscalation() {
  dutyEscalationModalOpen.value = false
  pushDutyToast(`${dutyEscalationForm.severity} 升级策略已更新`)
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
  if (!user || !window.confirm(`确认删除用户 ${user.username}？`)) return
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

function openCopilot() {
  copilotOpen.value = true
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

function iconPath(name) {
  const paths = {
    dashboard: ['M4 13h7V4H4z', 'M13 20h7V4h-7z', 'M4 20h7v-5H4z'],
    asset: ['M4 8l8-4 8 4-8 4z', 'M4 8v8l8 4 8-4V8', 'M12 12v8'],
    cmdb: ['M4 5h16v14H4z', 'M8 9h8', 'M8 13h5'],
    database: ['M5 5c0 2 3 3 7 3s7-1 7-3-3-3-7-3-7 1-7 3z', 'M5 5v10c0 2 3 4 7 4s7-2 7-4V5', 'M5 10c0 2 3 3 7 3s7-1 7-3'],
    cloud: ['M6 18h12a4 4 0 0 0 0-8 6 6 0 0 0-11-2A5 5 0 0 0 6 18z'],
    container: ['M12 3l8 4v10l-8 4-8-4V7z', 'M4 7l8 4 8-4', 'M12 11v10'],
    collab: ['M7 8h10', 'M7 12h6', 'M5 20l3-3h9a3 3 0 0 0 3-3V7a3 3 0 0 0-3-3H7a3 3 0 0 0-3 3v7'],
    calendar: ['M8 2v4', 'M16 2v4', 'M4 8h16', 'M4 4h16v16H4z'],
    task: ['M9 11l2 2 4-5', 'M5 4h14v16H5z', 'M8 17h8'],
    incident: ['M12 9v4', 'M12 17h.01', 'M10 3h4l7 12-2 4H5l-2-4z'],
    shield: ['M12 3l8 4v5c0 5-3.5 8-8 9-4.5-1-8-4-8-9V7z', 'M9 12l2 2 4-5'],
    users: ['M9 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z', 'M3 19c1-4 4-6 6-6s5 2 6 6', 'M16 11h5'],
    lock: ['M12 3l8 4v5c0 5-3.5 8-8 9-4.5-1-8-4-8-9V7z', 'M9 12h6'],
    audit: ['M6 4h12v16H6z', 'M9 12l2 2 4-5'],
    key: ['M8 12a3 3 0 1 0 0-6 3 3 0 0 0 0 6z', 'M11 9h9', 'M17 9v3'],
    sre: ['M12 3v3', 'M12 18v3', 'M3 12h3', 'M18 12h3', 'M12 7a5 5 0 1 0 0 10 5 5 0 0 0 0-10z'],
    eye: ['M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12z', 'M12 9a3 3 0 1 0 0 6 3 3 0 0 0 0-6z'],
    bell: ['M18 8a6 6 0 0 0-12 0c0 7-3 7-3 7h18s-3 0-3-7', 'M10 19a2 2 0 0 0 4 0'],
    deploy: ['M12 19V5', 'M5 12l7-7 7 7', 'M5 21h14'],
    chaos: ['M4 7h4l3 10h2l3-10h4', 'M4 17h4', 'M16 17h4'],
    book: ['M4 5a3 3 0 0 1 3-3h13v17H7a3 3 0 0 0-3 3z', 'M8 6h8', 'M8 10h7'],
    finops: ['M4 19V5', 'M4 19h16', 'M8 16v-4', 'M12 16V8', 'M16 16v-6'],
    bot: ['M12 8V4', 'M8 4h8', 'M5 8h14v10H5z', 'M9 13h.01', 'M15 13h.01'],
    change: ['M4 7h10', 'M14 7l-3-3', 'M14 7l-3 3', 'M20 17H10', 'M10 17l3-3', 'M10 17l3 3'],
    settings: ['M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8z', 'M4 12h2', 'M18 12h2', 'M12 4v2', 'M12 18v2', 'M6.6 6.6l1.4 1.4', 'M16 16l1.4 1.4', 'M17.4 6.6 16 8', 'M8 16l-1.4 1.4'],
    minimize: ['M5 12h14'],
    maximize: ['M5 5h14v14H5z'],
    restore: ['M8 8h11v11H8z', 'M5 5h11v3', 'M5 5v11h3']
  }
  return paths[name] || paths.dashboard
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
</script>

<template>
  <div class="app" :class="{ 'auth-app': !hasAppAccess, 'sidebar-collapsed': hasAppAccess && sidebarCollapsed }">
    <aside v-if="hasAppAccess" class="sidebar">
      <div class="brand">
        <img src="/logo.png" alt="OpsCore logo" />
        <span>OpsCore</span>
        <button class="sidebar-toggle" type="button" :aria-label="sidebarCollapsed ? '展开菜单' : '收起菜单'" @click="sidebarCollapsed = !sidebarCollapsed">
          {{ sidebarCollapsed ? '>' : '<' }}
        </button>
      </div>
      <div class="side-section">工作台</div>
      <button class="nav-row" :class="{ active: activeView === 'dashboard' }" title="首页仪表盘" @click="goToView('dashboard')">
        <span class="nav-icon"><svg viewBox="0 0 24 24"><path v-for="path in iconPath('dashboard')" :key="path" :d="path" /></svg></span>
        <span class="nav-label">首页仪表盘</span>
      </button>

      <div class="side-section">功能模块</div>
      <div v-for="group in menu.slice(1)" :key="group.label" class="nav-group" :class="{ disabled: group.enabled === false }">
        <div class="nav-parent" :title="group.label">
          <span class="nav-icon"><svg viewBox="0 0 24 24"><path v-for="path in iconPath(group.icon)" :key="path" :d="path" /></svg></span>
          <strong>{{ group.label }}</strong>
          <span class="nav-state">{{ group.enabled === false ? '灰度' : '展开' }}</span>
        </div>
        <div v-if="group.children" class="nav-children">
          <button
            v-for="child in group.children"
            :key="`${group.label}-${child.label}`"
            class="nav-child"
            :class="{ active: activeView === child.id && (!child.permissionTab || permissionTab === child.permissionTab), disabled: !child.enabled }"
            :disabled="!child.enabled"
            :title="child.label"
            @click="goToView(child.id, child.permissionTab)"
          >
            <span class="child-icon"><svg viewBox="0 0 24 24"><path v-for="path in iconPath(child.icon)" :key="path" :d="path" /></svg></span>
            <span>{{ child.label }}</span>
            <em>{{ child.enabled ? '一期' : '灰度' }}</em>
          </button>
        </div>
      </div>
    </aside>

    <main class="main">
      <header v-if="hasAppAccess && activeView !== 'oncall'" class="topbar">
        <div>
          <span class="breadcrumb">{{ activeBreadcrumb }}</span>
          <h1>{{ activeTitle }}</h1>
          <p>{{ activeSubtitle }}</p>
        </div>
        <div class="top-actions">
          <span v-if="error" class="error">{{ error }}</span>
          <span v-if="loading" class="muted">加载中...</span>
          <template v-if="auth.token">
            <span class="user-pill">{{ auth.user?.username || 'admin' }}</span>
            <button @click="logout">退出</button>
          </template>
        </div>
      </header>

      <section v-if="!auth.token" class="auth-screen">
        <div class="auth-hero">
          <div class="auth-brand">
            <img src="/logo.png" alt="OpsCore logo" />
            <span>OpsCore</span>
          </div>
          <h1>智能运维中枢指挥平台</h1>
          <p>连接资产、可观测、事件、变更、知识与自动化能力，构建以业务连续性为核心、以 AI 辅助决策和受控自动化为增强的统一运维控制平面。</p>
          <div class="auth-command-map" aria-label="OpsCore 智能运维中枢示意">
            <div class="command-orbit"></div>
            <div class="command-axis"></div>
            <div class="command-core">
              <span>OpsCore</span>
              <strong>智能运维中枢</strong>
            </div>
            <span class="command-node data">统一运维数据底座</span>
            <span class="command-node continuity">业务连续性保障</span>
            <span class="command-node automation">自动化处置闭环</span>
            <span class="command-node governance">治理与审计</span>
            <span class="command-node ai">AI 决策中枢</span>
            <div class="command-flow">
              <span>健康</span>
              <i></i>
              <span>影响</span>
              <i></i>
              <span>根因</span>
              <i></i>
              <span>处置</span>
              <i></i>
              <span>复盘</span>
            </div>
          </div>
        </div>
        <form class="login-card auth-card" @submit.prevent="submitLogin">
          <small>账号密码登录</small>
          <h2>登录 OpsCore</h2>
          <p>使用一期账号密码体系进入控制台。</p>
          <span v-if="error" class="error">{{ error }}</span>
          <label>账号<input v-model="auth.username" autocomplete="username" /></label>
          <label>密码<input v-model="auth.password" type="password" autocomplete="current-password" /></label>
          <button class="primary" type="submit">进入控制台</button>
          <div class="login-meta" aria-label="登录安全能力">
            <span>凭据加密</span>
            <span>RBAC</span>
            <span>审计预留</span>
          </div>
        </form>
      </section>

      <section v-else-if="authPending" class="auth-screen auth-loading">
        <div class="auth-hero">
          <div class="auth-brand">
            <img src="/logo.png" alt="OpsCore logo" />
            <span>OpsCore</span>
          </div>
          <h1>正在恢复控制台会话</h1>
          <p>正在校验登录状态并加载资产、值班、任务与事件数据。</p>
        </div>
        <div class="login-card auth-card">
          <small>Session Check</small>
          <h2>连接 OpsCore</h2>
          <p>请稍候，登录态校验完成后会自动进入控制台。</p>
          <span v-if="error" class="error">{{ error }}</span>
          <button type="button" @click="logout">重新登录</button>
        </div>
      </section>

      <section v-else-if="needsInitialPassword" class="auth-screen">
        <div class="auth-hero">
          <div class="auth-brand">
            <img src="/logo.png" alt="OpsCore logo" />
            <span>OpsCore</span>
          </div>
          <h1>初始化安全访问</h1>
          <p>首次进入控制台前，需要完成超级管理员初始化密码，确保敏感凭据和运维操作受控。</p>
          <div class="auth-command-map compact" aria-label="OpsCore 安全初始化示意">
            <div class="command-orbit"></div>
            <div class="command-axis"></div>
            <div class="command-core">
              <span>OpsCore</span>
              <strong>安全访问</strong>
            </div>
            <span class="command-node data">凭据加密</span>
            <span class="command-node continuity">权限隔离</span>
            <span class="command-node automation">操作闭环</span>
            <span class="command-node governance">治理审计</span>
            <span class="command-node ai">受控接入</span>
          </div>
        </div>
        <form class="login-card auth-card password-card" @submit.prevent="changeInitialPassword">
          <small>首次登录</small>
          <h2>初始化管理员密码</h2>
          <p>检测到当前账号仍在使用初始化密码。请先修改密码，再进入 OpsCore。</p>
          <span v-if="error" class="error">{{ error }}</span>
          <label>当前密码<input v-model="passwordInit.currentPassword" type="password" autocomplete="current-password" /></label>
          <label>新密码<input v-model="passwordInit.newPassword" type="password" placeholder="至少 8 位" autocomplete="new-password" /></label>
          <label>确认新密码<input v-model="passwordInit.confirmPassword" type="password" autocomplete="new-password" /></label>
          <button class="primary" :disabled="loading" type="submit">完成初始化</button>
          <button type="button" @click="logout">返回登录</button>
        </form>
      </section>

      <template v-else>
        <section v-if="activeView === 'dashboard'" class="dashboard">
          <div class="kpis">
            <button class="kpi-card kpi-assets" @click="goToView('cmdb')">
              <span class="kpi-icon"><svg viewBox="0 0 24 24"><path v-for="path in iconPath('asset')" :key="path" :d="path" /></svg></span>
              <span class="kpi-copy"><small>纳管资产</small></span>
              <span class="kpi-value"><strong>{{ dashboardAssetCount }}</strong><small>项</small></span>
              <span class="kpi-mini asset-bars">
                <span v-for="item in assetKpiBars" :key="item.label" :class="['asset-bar', item.tone]">
                  <b>{{ item.label }}</b>
                  <i :style="{ height: item.height }"><em>{{ item.value }}</em></i>
                </span>
              </span>
              <span class="kpi-status">纳管覆盖正常</span>
            </button>
            <button class="kpi-card kpi-oncall" @click="goToView('oncall')">
              <span class="kpi-icon"><svg viewBox="0 0 24 24"><path v-for="path in iconPath('calendar')" :key="path" :d="path" /></svg></span>
              <span class="kpi-copy"><small>今日值班</small></span>
              <span class="kpi-value"><strong>{{ dashboardOncallCount }}</strong><small>人</small></span>
              <span class="kpi-mini oncall-roster">
                <span><small>主值</small><b>{{ todayOncall.primary || '未配置' }}</b></span>
                <span><small>备值</small><b>{{ todayOncall.backup || '未配置' }}</b></span>
                <em>{{ todayOncall.ruleType === 'weekly' ? '本周轮换' : '08:00-20:00' }}</em>
              </span>
              <span class="kpi-status">响应窗口已覆盖</span>
            </button>
            <button class="kpi-card kpi-tasks" @click="goToView('tasks')">
              <span class="kpi-icon"><svg viewBox="0 0 24 24"><path v-for="path in iconPath('task')" :key="path" :d="path" /></svg></span>
              <span class="kpi-copy"><small>进行中任务</small></span>
              <span class="kpi-value"><strong>{{ dashboardTaskCount }}</strong><small>项</small></span>
              <span class="kpi-mini task-status-grid">
                <span v-for="item in taskKpiCards" :key="item.label" class="task-status-item">
                  <small><i :style="{ background: item.color }"></i>{{ item.label }}</small>
                  <b>{{ item.value }}</b>
                  <em><i :style="{ width: item.width, background: item.color }"></i></em>
                </span>
              </span>
              <span class="kpi-status">闭环节奏稳定</span>
            </button>
            <button class="kpi-card kpi-incidents" @click="goToView('incidents')">
              <span class="kpi-icon"><svg viewBox="0 0 24 24"><path v-for="path in iconPath('incident')" :key="path" :d="path" /></svg></span>
              <span class="kpi-copy"><small>活跃事件</small></span>
              <span class="kpi-value"><strong>{{ dashboardIncidentCount }}</strong><small>起</small></span>
              <span class="kpi-mini incident-levels">
                <span v-for="item in incidentKpiLevels" :key="item.label" :class="item.tone">
                  <b>{{ item.label }}</b>
                  <strong>{{ item.value }}</strong>
                  <em>{{ item.desc }}</em>
                </span>
              </span>
              <span class="kpi-status">风险持续跟进</span>
            </button>
          </div>
          <div class="metric-grid">
            <article class="metric metric-gauge">
              <div><span>资产健康率</span><strong>96.8%</strong><small>可用 1,241 · 异常 12</small></div>
              <div class="gauge" style="--value: 96.8"><b>96.8</b></div>
            </article>
            <article class="metric metric-bars">
              <div><span>事件平均响应</span><strong>18m</strong><small>P1/P2 优先拉起</small></div>
              <div class="response-bars"><i v-for="item in [{h:34,l:'P1'},{h:28,l:'P2'},{h:20,l:'P3'},{h:14,l:'P4'}]" :key="item.l" :style="{height: item.h + 'px'}"><em>{{ item.l }}</em></i></div>
            </article>
            <article class="metric metric-stack">
              <div><span>任务按时完成</span><strong>89%</strong><small>临近 5 · 逾期 2</small></div>
              <div class="stacked-progress"><i class="ok" style="width: 72%"></i><i class="warn" style="width: 17%"></i><i class="bad" style="width: 11%"></i></div>
              <div class="metric-legend"><span>按时</span><span>临近</span><span>逾期</span></div>
            </article>
            <article class="metric metric-risk">
              <div><span>风险收敛率</span><strong>92%</strong><small>事件关闭与复盘改进</small></div>
              <div class="risk-orbit"><i></i><i></i><i></i><i></i><b>闭环</b></div>
            </article>
          </div>
          <div class="split">
            <section class="panel">
              <h3>事件与任务优先级</h3>
              <div class="priority-list">
                <button v-for="item in dashboardPriorityItems" :key="item.id" class="priority-row" @click="openPriorityItem(item)">
                  <span :class="['pill', item.type === '事件' ? 'danger' : '']">{{ item.type }} {{ item.badge }}</span>
                  <strong>{{ item.title }}</strong>
                  <span>{{ item.owner }} · {{ item.status }} · {{ item.meta }}</span>
                </button>
                <p v-if="!dashboardPriorityItems.length" class="empty">暂无待关注事件或任务</p>
              </div>
            </section>
            <section class="panel">
              <h3>事件响应流程</h3>
              <div class="flow-diagram" aria-label="事件响应流程图">
                <div class="flow-node active"><span>01</span><strong>创建事件</strong><small>分级与关联资产</small></div>
                <i class="flow-arrow">→</i>
                <div class="flow-node"><span>02</span><strong>协同处置</strong><small>主值/负责人跟进</small></div>
                <i class="flow-arrow">→</i>
                <div class="flow-node"><span>03</span><strong>恢复确认</strong><small>验证业务影响</small></div>
                <i class="flow-arrow">→</i>
                <div class="flow-node done"><span>04</span><strong>关闭复盘</strong><small>沉淀改进项</small></div>
              </div>
              <div class="flow-legend"><span><i class="legend-dot danger"></i>P1/P2 拉起响应</span><span><i class="legend-dot success"></i>恢复后关闭</span></div>
            </section>
          </div>
        </section>

        <section v-if="activeView === 'cmdb'" class="panel">
          <div class="section-head">
            <div>
              <p class="muted">录入服务器与基础资产信息，维护配置规格、网络区域、所属业务、部署信息、负责人、状态和受控登录信息。</p>
            </div>
            <div class="page-actions">
              <button disabled><span class="btn-icon">↥</span>导入</button>
              <button v-if="selectedAsset" @click="exportAsset(selectedAsset)"><span class="btn-icon">↧</span>导出</button>
              <button class="primary" :disabled="!canWriteAssets" @click="openAssetForm"><span class="btn-icon">＋</span>新增资产</button>
            </div>
          </div>
          <div class="cmdb-layout" :class="{ 'single-column': !selectedAsset }">
            <section>
              <div class="query-card">
                <div class="query-main">
                  <input v-model="assetFilters.keyword" placeholder="搜索资产编号、业务、IP、负责人、部署信息" @input="assetPager.page = 1" />
                  <select v-model="assetFilters.type" @change="assetPager.page = 1"><option value="">全部类型</option><option>物理机</option><option>虚拟机</option></select>
                  <select v-model="assetFilters.environment" @change="assetPager.page = 1"><option value="">全部环境</option><option>生产</option><option>仿真</option><option>研发</option></select>
                  <button class="primary" @click="assetPager.page = 1">查询</button>
                  <button @click="resetAssetFilters">重置</button>
                  <button class="link" @click="assetFilters.advanced = !assetFilters.advanced">{{ assetFilters.advanced ? '收起高级搜索' : '高级搜索' }}</button>
                </div>
                <div v-if="assetFilters.advanced" class="query-extra">
                  <select v-model="assetFilters.business" @change="assetPager.page = 1">
                    <option value="">全部所属业务</option>
                    <option v-for="item in assetBusinesses" :key="item">{{ item }}</option>
                  </select>
                  <select v-model="assetFilters.networkZone" @change="assetPager.page = 1">
                    <option value="">全部网络区域</option>
                    <option v-for="item in assetNetworkZones" :key="item">{{ item }}</option>
                  </select>
                </div>
              </div>
              <section v-if="assetFormOpen" class="editor-panel" tabindex="-1" @focusout="closeEditorOnFocusOut($event, closeAssetForm)">
                <h3>{{ newAsset.id ? '编辑资产' : '新增资产' }}</h3>
                <div class="form-grid cmdb-form">
                <input v-model="newAsset.assetNo" placeholder="资产编号（留空自动生成）" />
                <select v-model="newAsset.type"><option>物理机</option><option>虚拟机</option></select>
                <input v-model="newAsset.vendor" placeholder="厂商" />
                <input v-model="newAsset.cpuArch" placeholder="CPU 架构" />
                <input v-model="newAsset.sn" placeholder="SN" />
                <input v-model="newAsset.location" placeholder="物理位置" />
                <input v-model="newAsset.business" placeholder="所属业务" />
                <input v-model="newAsset.ipv4" placeholder="IPv4" />
                <input v-model="newAsset.ipv6" placeholder="IPv6" />
                <select v-model="newAsset.environment"><option>生产</option><option>仿真</option><option>研发</option></select>
                <input v-model="newAsset.os" placeholder="操作系统" />
                <input v-model="newAsset.hostname" placeholder="主机名" />
                <input v-model="newAsset.networkZone" placeholder="网络区域" />
                <input v-model="newAsset.cpu" placeholder="CPU 规格" />
                <input v-model="newAsset.memory" placeholder="内存规格" />
                <input v-model="newAsset.disk" placeholder="磁盘规格" />
                <input v-model="newAsset.deploymentInfo" placeholder="部署信息" />
                <input v-model="newAsset.owner" placeholder="负责人" />
                <input v-model="newAsset.hostMachine" placeholder="所在宿主机（虚拟机可填）" />
                <select v-model="newAsset.status"><option>运行中</option><option>维护中</option><option>停用</option><option>故障</option></select>
                <section v-if="canManageCredentials" class="credential-inline form-wide">
                  <div>
                    <strong>登录信息</strong>
                    <span>保存后加密存储，列表不展示；查看密码需统一二次校验。</span>
                  </div>
                  <div class="form-grid credential-form-inline">
                    <input v-model="assetFormCredential.loginUrl" placeholder="登录地址（可选）" />
                    <input v-model="assetFormCredential.username" placeholder="登录用户名" />
                    <input v-model="assetFormCredential.secret" placeholder="登录密码 / 密钥" type="password" />
                    <input v-model="assetFormCredential.notes" placeholder="备注" />
                  </div>
                </section>
                <button class="primary" :disabled="!canWriteAssets" @click="saveAsset">{{ newAsset.id ? '保存修改' : '保存资产' }}</button>
                <button @click="closeAssetForm">取消</button>
                </div>
              </section>
              <div class="table-wrap">
                <table>
                  <thead><tr><th>资产编号</th><th>类型</th><th>环境</th><th>网络区域</th><th>IP</th><th>配置规格</th><th>所属业务</th><th>部署信息</th><th>状态</th><th>负责人</th><th>操作</th></tr></thead>
                  <tbody>
                    <tr v-for="asset in pagedAssets" :key="asset.id" :class="{ selected: selectedAsset?.id === asset.id }" class="clickable-row" tabindex="0" @click="chooseAsset(asset)" @keyup.enter="chooseAsset(asset)">
                      <td>{{ asset.assetNo }}</td><td>{{ asset.type }}</td><td>{{ asset.environment }}</td><td>{{ asset.networkZone }}</td><td>{{ asset.ipv4 || asset.ipv6 }}</td><td>{{ assetSpec(asset) }}</td><td>{{ asset.business }}</td><td>{{ asset.deploymentInfo || '-' }}</td><td>{{ asset.status }}</td><td>{{ asset.owner || '-' }}</td>
                      <td class="row-actions"><button class="link" @click.stop="chooseAsset(asset)">详情</button><button class="link" :disabled="!canWriteAssets || isSampleRecord(asset)" @click.stop="editAsset(asset)">编辑</button><button class="link danger-text" :disabled="!canDeleteAsset(asset)" @click.stop="deleteAsset(asset)">删除</button></td>
                    </tr>
                    <tr v-if="!pagedAssets.length"><td colspan="11" class="empty">未找到符合条件的资产</td></tr>
                  </tbody>
                </table>
              </div>
              <div class="pager">
                <span>共 {{ filteredAssets.length }} 条，每页显示：</span>
                <select v-model.number="assetPager.pageSize" @change="assetPager.page = 1"><option :value="10">10 条/页</option><option :value="20">20 条/页</option><option :value="50">50 条/页</option></select>
                <button @click="setPage(assetPager, assetPager.page - 1, assetPageCount)">‹</button>
                <button v-for="page in assetPageCount" :key="page" :class="{ active: page === assetPager.page }" @click="setPage(assetPager, page, assetPageCount)">{{ page }}</button>
                <button @click="setPage(assetPager, assetPager.page + 1, assetPageCount)">›</button>
              </div>
            </section>
            <aside v-if="selectedAsset" class="asset-detail">
              <div class="detail-title">
                <h3>资产详情</h3>
                <button class="detail-close" aria-label="隐藏资产详情" @click="hideAssetDetail">隐藏</button>
              </div>
              <dl>
                <dt>资产编号</dt><dd>{{ selectedAsset.assetNo }}</dd>
                <dt>厂商 / SN</dt><dd>{{ selectedAsset.vendor || '-' }} / {{ selectedAsset.sn || '-' }}</dd>
                <dt>配置规格</dt><dd>{{ assetSpec(selectedAsset) }}</dd>
                <dt>部署信息</dt><dd>{{ selectedAsset.deploymentInfo || '-' }}</dd>
                <dt>所在宿主机</dt><dd>{{ selectedAsset.hostMachine || '-' }}</dd>
                <dt>负责人</dt><dd>{{ selectedAsset.owner || '-' }}</dd>
              </dl>
              <div class="detail-actions">
                <button class="primary" :disabled="!canWriteAssets || isSampleRecord(selectedAsset)" @click="editAsset(selectedAsset)">编辑资产</button>
                <button class="danger-action" :disabled="!canDeleteAsset(selectedAsset)" @click="deleteAsset(selectedAsset)">删除资产</button>
              </div>
              <div class="credential-box">
                <h4>登录信息</h4>
                <template v-if="canManageCredentials">
                  <p class="muted">登录地址、账号与备注可维护；密码/密钥默认隐藏，需输入权限管理中配置的统一校验密码后查看。</p>
                  <div class="credential-actions">
                    <button @click="loadCredential">加载登录信息</button>
                    <span v-if="credential.hasSecret" class="pill danger">已保存密钥</span>
                    <span v-else class="pill">未保存密钥</span>
                  </div>
                  <div class="form-grid credential-form">
                    <input v-model="credential.loginUrl" placeholder="登录地址" />
                    <input v-model="credential.username" placeholder="账号" />
                    <input v-model="credential.secret" placeholder="密码 / 密钥（留空则保留原值）" type="password" />
                    <input v-model="credential.notes" placeholder="备注" />
                    <button class="primary" @click="saveCredential">保存登录信息</button>
                  </div>
                  <div class="credential-reveal">
                    <input v-model="credentialReveal.password" placeholder="输入统一二次校验密码查看密码/密钥" type="password" @keyup.enter="revealCredential" />
                    <button @click="revealCredential">二次校验查看</button>
                    <span v-if="credentialReveal.revealed" class="pill success">已校验</span>
                  </div>
                </template>
                <p v-else class="muted">当前角色无权查看登录信息。运维工程师默认不可见。</p>
                <p v-if="credentialMessage" class="error inline">{{ credentialMessage }}</p>
              </div>
            </aside>
          </div>
        </section>

        <section v-if="activeView === 'middleware'" class="panel">
          <div class="section-head">
            <div>
              <p class="muted">一期类型：MySQL、Redis、Kafka、PostgreSQL、达梦、Nginx、ElasticSearch、Nacos、RocketMQ、MinIO。</p>
            </div>
            <div class="page-actions">
              <button disabled><span class="btn-icon">↥</span>导入</button>
              <button v-if="selectedMiddleware" @click="exportMiddleware(selectedMiddleware)"><span class="btn-icon">↧</span>导出</button>
              <button class="primary" :disabled="!canWriteAssets" @click="openMiddlewareForm"><span class="btn-icon">＋</span>新增实例</button>
            </div>
          </div>
          <div class="work-layout" :class="{ 'single-column': !selectedMiddleware }">
            <section>
              <div class="query-card">
                <div class="query-main">
                  <input v-model="middlewareFilters.keyword" placeholder="搜索实例名称、类型、访问地址、业务、负责人" @input="middlewarePager.page = 1" />
                  <select v-model="middlewareFilters.kind" @change="middlewarePager.page = 1">
                    <option value="">全部类型</option>
                    <option>MySQL</option><option>Redis</option><option>Kafka</option><option>PostgreSQL</option><option>达梦</option><option>Nginx</option><option>ElasticSearch</option><option>Nacos</option><option>RocketMQ</option><option>MinIO</option>
                  </select>
                  <select v-model="middlewareFilters.environment" @change="middlewarePager.page = 1"><option value="">全部环境</option><option>生产</option><option>仿真</option><option>研发</option></select>
                  <button class="primary" @click="middlewarePager.page = 1">查询</button>
                  <button @click="resetMiddlewareFilters">重置</button>
                  <button class="link" @click="middlewareFilters.advanced = !middlewareFilters.advanced">{{ middlewareFilters.advanced ? '收起高级搜索' : '高级搜索' }}</button>
                </div>
                <div v-if="middlewareFilters.advanced" class="query-extra">
                  <select v-model="middlewareFilters.business" @change="middlewarePager.page = 1">
                    <option value="">全部所属业务</option>
                    <option v-for="item in middlewareBusinesses" :key="item">{{ item }}</option>
                  </select>
                  <select v-model="middlewareFilters.networkZone" @change="middlewarePager.page = 1">
                    <option value="">全部网络区域</option>
                    <option v-for="item in middlewareNetworkZones" :key="item">{{ item }}</option>
                  </select>
                  <select v-model="middlewareFilters.status" @change="middlewarePager.page = 1"><option value="">全部状态</option><option>运行中</option><option>维护中</option><option>停用</option><option>故障</option></select>
                </div>
              </div>
              <section v-if="middlewareFormOpen" class="editor-panel" tabindex="-1" @focusout="closeEditorOnFocusOut($event, closeMiddlewareForm)">
                <h3>{{ newMiddleware.id ? '编辑实例' : '新增实例' }}</h3>
                <div class="form-grid">
                <input v-model="newMiddleware.name" placeholder="实例名称" />
                <select v-model="newMiddleware.kind">
                  <option>MySQL</option><option>Redis</option><option>Kafka</option><option>PostgreSQL</option><option>达梦</option><option>Nginx</option><option>ElasticSearch</option><option>Nacos</option><option>RocketMQ</option><option>MinIO</option>
                </select>
                <input v-model="newMiddleware.version" placeholder="版本" />
                <select v-model="newMiddleware.environment"><option>生产</option><option>仿真</option><option>研发</option></select>
                <input v-model="newMiddleware.networkZone" placeholder="网络区域" />
                <input v-model="newMiddleware.endpoint" placeholder="访问地址 / 端口" />
                <input v-model="newMiddleware.business" placeholder="所属业务" />
                <input v-model="newMiddleware.owner" placeholder="负责人" />
                <input v-model="newMiddleware.assetId" placeholder="关联资产 ID（非必填）" />
                <select v-model="newMiddleware.status"><option>运行中</option><option>维护中</option><option>停用</option><option>故障</option></select>
                <section v-if="canManageCredentials" class="credential-inline form-wide">
                  <div>
                    <strong>实例登录信息</strong>
                    <span>保存后加密存储，列表不展示；查看密码需统一二次校验。</span>
                  </div>
                  <div class="form-grid credential-form-inline">
                    <input v-model="middlewareFormCredential.loginUrl" placeholder="管理地址 / 连接入口（可选）" />
                    <input v-model="middlewareFormCredential.username" placeholder="登录用户名" />
                    <input v-model="middlewareFormCredential.secret" placeholder="登录密码 / 密钥" type="password" />
                    <input v-model="middlewareFormCredential.notes" placeholder="备注" />
                  </div>
                </section>
                <button class="primary" :disabled="!canWriteAssets" @click="saveMiddleware">{{ newMiddleware.id ? '保存修改' : '保存实例' }}</button>
                <button @click="closeMiddlewareForm">取消</button>
                </div>
              </section>
              <div class="table-wrap">
                <table>
                  <thead><tr><th>实例名称</th><th>类型</th><th>环境</th><th>网络区域</th><th>访问地址</th><th>所属业务</th><th>关联资产</th><th>状态</th><th>操作</th></tr></thead>
                  <tbody>
                    <tr v-for="item in pagedMiddleware" :key="item.id" :class="{ selected: selectedMiddleware?.id === item.id }" class="clickable-row" tabindex="0" @click="chooseMiddleware(item)" @keyup.enter="chooseMiddleware(item)">
                      <td>{{ item.name }}</td><td>{{ item.kind }}</td><td>{{ item.environment }}</td><td>{{ item.networkZone || '-' }}</td><td>{{ item.endpoint }}</td><td>{{ item.business }}</td><td>{{ associatedAssetName(item) }}</td><td>{{ item.status }}</td>
                      <td class="row-actions"><button class="link" @click.stop="chooseMiddleware(item)">详情</button><button class="link" :disabled="!canWriteAssets || isSampleRecord(item)" @click.stop="editMiddleware(item)">编辑</button><button class="link danger-text" :disabled="!canWriteAssets || isSampleRecord(item)" @click.stop="deleteMiddleware(item)">删除</button></td>
                    </tr>
                    <tr v-if="!pagedMiddleware.length"><td colspan="9" class="empty">未找到符合条件的实例</td></tr>
                  </tbody>
                </table>
              </div>
              <div class="pager">
                <span>共 {{ filteredMiddleware.length }} 条，每页显示：</span>
                <select v-model.number="middlewarePager.pageSize" @change="middlewarePager.page = 1"><option :value="10">10 条/页</option><option :value="20">20 条/页</option><option :value="50">50 条/页</option></select>
                <button @click="setPage(middlewarePager, middlewarePager.page - 1, middlewarePageCount)">‹</button>
                <button v-for="page in middlewarePageCount" :key="page" :class="{ active: page === middlewarePager.page }" @click="setPage(middlewarePager, page, middlewarePageCount)">{{ page }}</button>
                <button @click="setPage(middlewarePager, middlewarePager.page + 1, middlewarePageCount)">›</button>
              </div>
            </section>
            <aside v-if="selectedMiddleware" class="detail-card">
              <div class="detail-title">
                <h3>{{ selectedMiddleware.name }}</h3>
                <button class="detail-close" aria-label="隐藏实例详情" @click="hideMiddlewareDetail">隐藏</button>
              </div>
              <dl>
                <dt>类型 / 版本</dt><dd>{{ selectedMiddleware.kind }} / {{ selectedMiddleware.version || '-' }}</dd>
                <dt>环境</dt><dd>{{ selectedMiddleware.environment }}</dd>
                <dt>网络区域</dt><dd>{{ selectedMiddleware.networkZone || '-' }}</dd>
                <dt>访问地址</dt><dd>{{ selectedMiddleware.endpoint }}</dd>
                <dt>所属业务</dt><dd>{{ selectedMiddleware.business }}</dd>
                <dt>负责人</dt><dd>{{ selectedMiddleware.owner || '-' }}</dd>
                <dt>关联资产</dt><dd>{{ associatedAssetName(selectedMiddleware) }}</dd>
                <dt>状态</dt><dd>{{ selectedMiddleware.status }}</dd>
              </dl>
              <div class="detail-actions">
                <button class="primary" :disabled="!canWriteAssets || isSampleRecord(selectedMiddleware)" @click="editMiddleware(selectedMiddleware)">编辑实例</button>
                <button :disabled="!canWriteAssets || isSampleRecord(selectedMiddleware)" @click="editMiddleware(selectedMiddleware)">编辑关联资产</button>
                <button class="danger-action" :disabled="!canWriteAssets || isSampleRecord(selectedMiddleware)" @click="deleteMiddleware(selectedMiddleware)">删除实例</button>
              </div>
              <div class="credential-box">
                <h4>实例账号密码</h4>
                <template v-if="canManageCredentials">
                  <p class="muted">用于保存数据库、中间件或组件实例的访问账号；密码/密钥默认隐藏，需输入权限管理中配置的统一校验密码后查看。</p>
                  <div class="credential-actions">
                    <button @click="loadMiddlewareCredential">加载账号密码</button>
                    <span v-if="middlewareCredential.hasSecret" class="pill danger">已保存密钥</span>
                    <span v-else class="pill">未保存密钥</span>
                  </div>
                  <div class="form-grid credential-form">
                    <input v-model="middlewareCredential.loginUrl" placeholder="管理地址 / 连接入口" />
                    <input v-model="middlewareCredential.username" placeholder="账号" />
                    <input v-model="middlewareCredential.secret" placeholder="密码 / 密钥（留空则保留原值）" type="password" />
                    <input v-model="middlewareCredential.notes" placeholder="备注" />
                    <button class="primary" @click="saveMiddlewareCredential">保存账号密码</button>
                  </div>
                  <div class="credential-reveal">
                    <input v-model="middlewareCredentialReveal.password" placeholder="输入统一二次校验密码查看密码/密钥" type="password" @keyup.enter="revealMiddlewareCredential" />
                    <button @click="revealMiddlewareCredential">二次校验查看</button>
                    <span v-if="middlewareCredentialReveal.revealed" class="pill success">已校验</span>
                  </div>
                </template>
                <p v-else class="muted">当前角色无权查看实例账号密码。运维工程师默认不可见。</p>
                <p v-if="middlewareCredentialMessage" class="error inline">{{ middlewareCredentialMessage }}</p>
              </div>
            </aside>
          </div>
        </section>

        <section v-if="activeView === 'oncall'" class="duty-center">
          <div class="duty-integrated-head">
            <div>
              <h2>值班管理</h2>
              <p>面向事件响应连续性，统一管理当前值班、排班日历、交接日志与升级策略。</p>
            </div>
            <div class="duty-status-strip">
              <span class="pill success">当前值班正常</span>
              <span>今日已处理 14 个告警</span>
            </div>
          </div>

          <div class="duty-tabbar">
            <button v-for="item in dutySections" :key="item.id" :class="{ active: dutySection === item.id }" @click="setDutySection(item.id)">
              {{ item.label }}
            </button>
          </div>

          <div class="duty-filter-strip">
            <label class="duty-search">
              <input v-model="dutyGlobalSearch" placeholder="搜索值班人员、排班或告警..." @keyup.enter="dutySearchSubmit" />
              <button @click="dutySearchSubmit">查询</button>
            </label>
            <div class="duty-team-tabs">
              <button v-for="team in dutyTeams" :key="team.label" :class="{ active: dutyTeamFilter === team.label }" @click="setDutyTeam(team.label)">
                {{ team.label }} <em>{{ team.count }}</em>
              </button>
            </div>
            <button class="duty-team-config" :disabled="!canWriteOncall" @click="openDutyTeamModal">团队配置</button>
          </div>

          <main class="duty-main">
              <section v-if="dutySection === 'overview'" class="duty-view">
                <div class="duty-page-head">
                  <div>
                    <h2>值班概览</h2>
                    <p>2026年6月10日 · 今日值班情况一览</p>
                  </div>
                  <div class="duty-page-actions">
                    <button class="primary" @click="quickDutyTakeover">立即接班</button>
                  </div>
                </div>

                <div class="duty-stats">
                  <article>
                    <small>当前值班人员</small>
                    <strong>{{ dutyFilteredCurrent.length }}</strong>
                    <span>全部在线</span>
                  </article>
                  <article>
                    <small>本周值班次数</small>
                    <strong>42</strong>
                    <span>较上周 -8</span>
                  </article>
                  <article>
                    <small>平均响应时间</small>
                    <strong>4.2min</strong>
                    <span>比上周快 1.1min</span>
                  </article>
                  <article>
                    <small>待处理告警</small>
                    <strong>7</strong>
                    <span>2 级P1</span>
                  </article>
                </div>

                <div v-if="dutyInlineAlert" class="duty-alert-card">
                  <div>
                    <strong>{{ dutyInlineAlert.title }}</strong>
                    <span>{{ dutyInlineAlert.detail }}</span>
                  </div>
                  <em>{{ dutyInlineAlert.status }}</em>
                  <button @click="ackDutyAlert">确认 (ACK)</button>
                  <button @click="escalateDutyAlert">升级</button>
                </div>

                <div class="duty-overview-grid">
                  <section class="duty-panel duty-current-panel">
                    <div class="duty-panel-head">
                      <div>
                        <h3>当前值班</h3>
                        <p>ON DUTY</p>
                      </div>
                      <button @click="setDutySection('roster')">查看全部</button>
                    </div>
                    <div class="duty-current-list">
                      <article v-for="person in dutyFilteredCurrent" :key="person.id" class="duty-person-card">
                        <span class="duty-avatar">{{ person.avatar }}</span>
                        <div>
                          <strong>{{ person.name }}</strong>
                          <small>{{ person.role }} · {{ person.team }}</small>
                          <em>{{ person.since }}-{{ person.until }} · {{ person.phone }}</em>
                        </div>
                        <b>{{ person.status }}</b>
                      </article>
                    </div>
                  </section>

                  <section class="duty-panel">
                    <div class="duty-panel-head">
                      <div>
                        <h3>即将值班（未来3天）</h3>
                        <p>点击日期分配或调整值班</p>
                      </div>
                      <button @click="setDutySection('calendar')">查看完整日历</button>
                    </div>
                    <div class="duty-upcoming-list">
                      <button v-for="item in dutyUpcomingAssignments" :key="item.date" @click="openDutyAssignModal(item.date, 'primary')">
                        <span>{{ item.date }}</span>
                        <strong>{{ item.primary }}</strong>
                        <em>备份：{{ item.backup }}</em>
                      </button>
                    </div>
                  </section>
                </div>

                <section class="duty-panel">
                  <div class="duty-panel-head">
                    <div>
                      <h3>统计报表</h3>
                      <p>值班覆盖率、响应效率、替班情况和满意度分析</p>
                    </div>
                    <select><option>本月</option><option>本周</option><option>本季度</option></select>
                  </div>
                  <div class="duty-report-grid">
                    <article v-for="item in dutyReports" :key="item.label">
                      <small>{{ item.label }}</small>
                      <strong>{{ item.value }}</strong>
                      <span>{{ item.detail }}</span>
                      <em><i :style="{ width: item.width }"></i></em>
                    </article>
                  </div>
                  <div class="duty-report-charts">
                    <section><h3>本月排班趋势</h3><div class="duty-chart-bars"><i v-for="height in ['38%','55%','48%','72%','64%','82%','58%']" :key="height" :style="{ height }"></i></div></section>
                    <section><h3>团队负载分布</h3><div class="duty-donut"><b>82%</b></div></section>
                    <section><h3>响应时间走势</h3><svg viewBox="0 0 180 74"><path d="M6 58 C34 30 54 42 78 26 S125 18 174 12"/></svg></section>
                  </div>
                </section>
              </section>

              <section v-if="dutySection === 'calendar'" class="duty-view">
                <div class="duty-page-head">
                  <div>
                    <h2>排班日历</h2>
                    <p>6月 2026 · 点击日期分配或调整值班</p>
                  </div>
                  <div class="duty-calendar-tools">
                    <button @click="moveDutyMonth(-1)">上一月</button>
                    <strong>{{ dutyCalendarMonthLabel }}</strong>
                    <button @click="moveDutyMonth(1)">下一月</button>
                    <button @click="resetDutyCalendar">重置视图</button>
                  </div>
                </div>
                <div class="duty-calendar-legend">
                  <span><i></i>主值班</span>
                  <span><i class="backup"></i>备份值班</span>
                  <button class="primary" :disabled="!canWriteOncall" @click="openDutyAssignModal('2026-06-10', 'primary')">手动分配值班</button>
                </div>
                <div class="duty-calendar">
                  <strong v-for="label in weekdayLabels" :key="label" class="duty-week-label">{{ label }}</strong>
                  <button v-for="cell in dutyCalendarCells" :key="cell.key" :class="['duty-day', { blank: cell.blank, today: cell.isToday, weekend: cell.isWeekend }]" :disabled="cell.blank" @click="!cell.blank && openDutyAssignModal(cell.key, 'primary')">
                    <template v-if="!cell.blank">
                      <span>{{ cell.day }}</span>
                      <em v-if="cell.primary">主 {{ cell.primary }}</em>
                      <em v-else class="empty">+ 分配</em>
                      <small v-if="cell.backup">备 {{ cell.backup }}</small>
                    </template>
                  </button>
                </div>
              </section>

              <section v-if="dutySection === 'schedules'" class="duty-view">
                <div class="duty-page-head">
                  <div>
                    <h2>排班配置</h2>
                    <p>管理值班模板、轮换规则和人员安排</p>
                  </div>
                  <button class="primary" :disabled="!canWriteOncall" @click="openCreateDutyScheduleModal">新建排班</button>
                </div>
                <div class="duty-schedule-grid">
                  <article v-for="schedule in dutySchedules" :key="schedule.id" class="duty-schedule-card">
                    <div>
                      <strong>{{ schedule.name }}</strong>
                      <span>{{ schedule.team }} · {{ schedule.rotation === 'weekly' ? '按周轮换' : '按天轮换' }}</span>
                    </div>
                    <p>{{ schedule.time }} · {{ schedule.members.join(' / ') }}</p>
                    <em :class="{ active: schedule.active }">{{ schedule.active ? '启用中' : '已停用' }}</em>
                    <div class="duty-card-actions">
                      <button @click="editDutySchedule(schedule)">编辑</button>
                      <button @click="toggleDutySchedule(schedule)">{{ schedule.active ? '停用' : '启用' }}</button>
                      <button class="danger-text" @click="deleteDutySchedule(schedule)">删除</button>
                    </div>
                  </article>
                </div>
              </section>

              <section v-if="dutySection === 'roster'" class="duty-view">
                <div class="duty-page-head">
                  <div>
                    <h2>值班人员列表</h2>
                    <p>管理人员、团队、角色、值班次数和下一班次</p>
                  </div>
                  <button class="primary" :disabled="!canWriteOncall" @click="openAddDutyPerson">添加人员</button>
                </div>
                <div class="duty-filter-row">
                  <input v-model="dutyRosterSearch" placeholder="搜索姓名..." />
                </div>
                <div class="table-wrap duty-table-wrap">
                  <table>
                    <thead><tr><th>人员</th><th>团队</th><th>角色</th><th>本月值班</th><th>下次值班</th><th>状态</th><th>操作</th></tr></thead>
                    <tbody>
                      <tr v-for="person in dutyFilteredRoster" :key="person.id">
                        <td><span class="duty-table-person"><b>{{ person.name.slice(0, 1) }}</b>{{ person.name }}</span></td>
                        <td>{{ person.team }}</td>
                        <td>{{ person.role }}</td>
                        <td>{{ person.count }}</td>
                        <td>{{ person.next }}</td>
                        <td><span :class="['pill', person.status === '值班中' ? 'success' : '']">{{ person.status }}</span></td>
                        <td class="row-actions"><button class="link" :disabled="!canWriteOncall" @click.stop="quickAssignDutyPerson(person)">安排值班</button></td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </section>

              <section v-if="dutySection === 'handover'" class="duty-view">
                <div class="duty-page-head">
                  <div>
                    <h2>交接班日志</h2>
                    <p>记录交接内容，确保信息连续性</p>
                  </div>
                  <button class="primary" @click="openDutyHandoverModal">提交交接</button>
                </div>
                <div class="duty-handover-list">
                  <article v-for="item in dutyHandovers" :key="item.id">
                    <div>
                      <strong>{{ item.from }} → {{ item.to }}</strong>
                      <small>{{ item.time }}</small>
                    </div>
                    <p>{{ item.content }}</p>
                    <span>{{ item.complete ? '交接完成' : '待确认' }}</span>
                  </article>
                </div>
              </section>

              <section v-if="dutySection === 'escalation'" class="duty-view">
                <div class="duty-page-head">
                  <div>
                    <h2>升级策略</h2>
                    <p>配置告警升级路径与响应时限</p>
                  </div>
                  <button class="primary" @click="dutyEscalationModalOpen = true">编辑策略</button>
                </div>
                <section class="duty-panel">
                  <div class="duty-panel-head">
                    <div>
                      <h3>P1 严重告警升级流程（基础运维组）</h3>
                      <p>主值班未响应时，按时间窗口逐级升级。</p>
                    </div>
                  </div>
                  <div class="duty-escalation-flow">
                    <template v-for="(level, index) in dutyEscalationLevels" :key="level.level">
                      <article>
                        <small>Level {{ level.level }}</small>
                        <strong>{{ level.target }}</strong>
                        <span>{{ level.delay }} · {{ level.channel }}</span>
                      </article>
                      <i v-if="index < dutyEscalationLevels.length - 1"></i>
                    </template>
                  </div>
                </section>
              </section>

            </main>

          <div class="duty-toast-stack">
            <div v-for="toast in dutyToasts" :key="toast.id" :class="['duty-toast', toast.tone]">{{ toast.message }}</div>
          </div>

          <div v-if="dutyAssignModalOpen" class="duty-modal-backdrop">
            <section class="duty-modal">
              <div class="duty-modal-head"><h3>分配值班</h3><button @click="dutyAssignModalOpen = false">×</button></div>
              <label>值班日期<input v-model="dutyAssignForm.date" type="date" /></label>
              <div class="duty-radio-row">
                <label><input v-model="dutyAssignForm.type" type="radio" value="primary" />主值班</label>
                <label><input v-model="dutyAssignForm.type" type="radio" value="backup" />备份值班</label>
              </div>
              <label>值班人员<select v-model="dutyAssignForm.person"><option v-for="person in dutyPeopleOptions" :key="person">{{ person }}</option></select></label>
              <label>备注<textarea v-model="dutyAssignForm.note" placeholder="填写特殊说明或覆盖范围"></textarea></label>
              <div class="duty-modal-actions"><button @click="dutyAssignModalOpen = false">取消</button><button class="primary" @click="confirmDutyAssign">确认分配</button></div>
            </section>
          </div>

          <div v-if="dutyScheduleModalOpen" class="duty-modal-backdrop">
            <section class="duty-modal">
              <div class="duty-modal-head"><h3>{{ dutyScheduleModalMode === 'edit' ? '编辑排班模板' : '新建排班模板' }}</h3><button @click="dutyScheduleModalOpen = false">×</button></div>
              <label>模板名称<input v-model="dutyScheduleForm.name" /></label>
              <label>团队<select v-model="dutyScheduleForm.team"><option v-for="team in dutyTeams.filter((item) => item.label !== '全部团队')" :key="team.label">{{ team.label }}</option></select></label>
              <label>轮换周期<select v-model="dutyScheduleForm.rotation"><option value="weekly">按周轮换</option><option value="daily">按天轮换</option></select></label>
              <label>值班时段<input v-model="dutyScheduleForm.time" /></label>
              <label>值班成员<select v-model="dutyScheduleForm.members" multiple><option v-for="person in dutyPeopleOptions" :key="person">{{ person }}</option></select></label>
              <p class="muted">成员预览：{{ dutyScheduleForm.members.join(' / ') || '未选择成员' }}</p>
              <div class="duty-modal-actions"><button @click="dutyScheduleModalOpen = false">取消</button><button class="primary" @click="saveDutyScheduleTemplate">{{ dutyScheduleModalMode === 'edit' ? '保存修改' : '创建排班' }}</button></div>
            </section>
          </div>

          <div v-if="dutyUserModalOpen" class="duty-modal-backdrop">
            <section class="duty-modal">
              <div class="duty-modal-head"><h3>添加值班人员</h3><button @click="dutyUserModalOpen = false">×</button></div>
              <label>关联系统用户<select v-model="dutyUserForm.userId"><option value="">请选择系统用户</option><option v-for="user in dutySystemUserOptions" :key="user.id" :value="user.id">{{ user.name }}（{{ user.username }}）</option></select></label>
              <label>所属团队<select v-model="dutyUserForm.team"><option v-for="team in dutyTeams.filter((item) => item.label !== '全部团队')" :key="team.label">{{ team.label }}</option></select></label>
              <label>值班角色<select v-model="dutyUserForm.role"><option>SRE</option><option>SRE 负责人</option><option>运维工程师</option><option>网络工程师</option><option>DBA</option><option>开发工程师</option></select></label>
              <label>下次值班<input v-model="dutyUserForm.next" placeholder="例如：2026-06-20 或 待安排" /></label>
              <p class="muted">值班人员应绑定系统用户，后续可复用账号状态、角色权限、通知渠道和审计记录。</p>
              <div class="duty-modal-actions"><button @click="dutyUserModalOpen = false">取消</button><button class="primary" @click="saveDutyUser">添加人员</button></div>
            </section>
          </div>

          <div v-if="dutyTeamModalOpen" class="duty-modal-backdrop">
            <section class="duty-modal duty-modal-wide">
              <div class="duty-modal-head"><h3>团队配置</h3><button @click="dutyTeamModalOpen = false">×</button></div>
              <p class="muted">维护值班团队名称。保存后会同步更新人员列表、当前值班和排班模板中的团队归属。</p>
              <div class="duty-team-editor">
                <article v-for="(team, index) in dutyTeamDrafts" :key="team.original || index">
                  <label>团队名称<input v-model="team.label" /></label>
                  <span>{{ team.count }} 人</span>
                  <button class="duty-team-delete" :disabled="Number(team.count || 0) > 0" :title="Number(team.count || 0) > 0 ? '该团队下仍有人员，先调整人员团队后再删除' : '删除团队'" @click="removeDutyTeamDraft(index)">删除</button>
                </article>
              </div>
              <button class="duty-secondary-action" @click="addDutyTeamDraft">新增团队</button>
              <div class="duty-modal-actions"><button @click="dutyTeamModalOpen = false">取消</button><button class="primary" @click="saveDutyTeams">保存团队</button></div>
            </section>
          </div>

          <div v-if="dutyHandoverModalOpen" class="duty-modal-backdrop">
            <section class="duty-modal">
              <div class="duty-modal-head"><h3>提交交接班</h3><button @click="dutyHandoverModalOpen = false">×</button></div>
              <label>交接人<select v-model="dutyHandoverForm.from"><option v-for="person in dutyPeopleOptions" :key="person">{{ person }}</option></select></label>
              <label>接班人<select v-model="dutyHandoverForm.to"><option v-for="person in dutyPeopleOptions" :key="person">{{ person }}</option></select></label>
              <label>交接内容<textarea v-model="dutyHandoverForm.content" placeholder="填写系统状态、未完成事项、风险和注意事项"></textarea></label>
              <label class="inline-check"><input v-model="dutyHandoverForm.complete" type="checkbox" />交接已完成，系统运行正常</label>
              <div class="duty-modal-actions"><button @click="dutyHandoverModalOpen = false">取消</button><button class="primary" @click="submitDutyHandover">提交交接</button></div>
            </section>
          </div>

          <div v-if="dutyEscalationModalOpen" class="duty-modal-backdrop">
            <section class="duty-modal duty-modal-wide">
              <div class="duty-modal-head"><h3>编辑升级策略</h3><button @click="dutyEscalationModalOpen = false">×</button></div>
              <label>策略名称<input v-model="dutyEscalationForm.name" /></label>
              <label>团队<select v-model="dutyEscalationForm.team"><option v-for="team in dutyTeams.filter((item) => item.label !== '全部团队')" :key="team.label">{{ team.label }}</option></select></label>
              <label>告警等级<select v-model="dutyEscalationForm.severity"><option>P1</option><option>P2</option><option>P3</option></select></label>
              <div class="duty-level-editor">
                <article v-for="level in dutyEscalationLevels" :key="level.level">
                  <strong>Level {{ level.level }}</strong>
                  <label>通知对象<input v-model="level.target" /></label>
                  <label>升级时间<input v-model="level.delay" /></label>
                  <label>通知渠道<input v-model="level.channel" /></label>
                </article>
              </div>
              <div class="duty-modal-actions"><button @click="dutyEscalationModalOpen = false">取消</button><button class="primary" @click="saveDutyEscalation">保存策略</button></div>
            </section>
          </div>
        </section>

        <section v-if="activeView === 'tasks'" class="panel">
          <div class="section-head">
            <div>
              <p class="muted">聚焦任务派发、负责人处理、关联事件/资产和状态闭环。</p>
            </div>
            <div class="page-actions">
              <span class="pill">未关闭 {{ activeTasks.length }}</span>
              <button class="primary" @click="openTaskForm"><span class="btn-icon">＋</span>创建任务</button>
            </div>
          </div>
          <div class="status-grid task-status">
            <article v-for="status in ['待处理','处理中','待确认','已完成','已关闭']" :key="status" :class="{ active: status === '处理中' }">
              <small>{{ status }}</small>
              <strong>{{ taskStatusCounts[status] || 0 }}</strong>
              <span>{{ status === '待确认' ? '等待发起人确认' : status === '已关闭' ? '归档记录' : '任务状态' }}</span>
            </article>
          </div>
          <section v-if="taskFormOpen" class="editor-panel" tabindex="-1" @focusout="closeEditorOnFocusOut($event, closeTaskForm)">
            <h3>{{ newTask.id ? '编辑任务' : '创建任务' }}</h3>
            <div class="form-grid">
              <input v-model="newTask.title" placeholder="任务标题" />
              <input v-model="newTask.assignee" placeholder="负责人" />
              <input v-model="newTask.dueAt" placeholder="截止时间" />
              <select v-model="newTask.status"><option>待处理</option><option>处理中</option><option>待确认</option><option>已完成</option><option>已关闭</option></select>
              <textarea v-model="newTask.description" class="form-wide" rows="4" placeholder="任务说明：描述背景、处理要求、关联资产或验收标准"></textarea>
              <button class="primary" @click="saveTask">{{ newTask.id ? '保存修改' : '创建任务' }}</button>
              <button @click="closeTaskForm">取消</button>
            </div>
          </section>
          <div class="work-layout" :class="{ 'single-column': !currentTask }">
            <div class="table-wrap">
              <table>
                <thead><tr><th>标题</th><th>负责人</th><th>状态</th><th>截止时间</th><th>操作</th></tr></thead>
                <tbody>
                  <tr v-for="task in displayTasks" :key="task.id" :class="{ selected: currentTask?.id === task.id }" class="clickable-row" tabindex="0" @click="chooseTask(task)" @keyup.enter="chooseTask(task)">
                    <td>{{ task.title }}</td>
                    <td>{{ task.assignee || '-' }}</td>
                    <td><span class="pill">{{ task.status }}</span></td>
                    <td>{{ task.dueAt || '-' }}</td>
                    <td class="row-actions"><button class="link" @click.stop="chooseTask(task)">详情</button><button class="link" :disabled="isSampleRecord(task)" @click.stop="editTask(task)">编辑</button><button class="link danger-text" :disabled="isSampleRecord(task)" @click.stop="deleteTask(task)">删除</button></td>
                  </tr>
                </tbody>
              </table>
            </div>
            <aside class="detail-card" v-if="currentTask">
              <div class="detail-title">
                <h3>{{ currentTask.title }}</h3>
                <button class="detail-close" aria-label="隐藏任务详情" @click="hideTaskDetail">隐藏</button>
              </div>
              <dl>
                <dt>负责人</dt><dd>{{ currentTask.assignee || '-' }}</dd>
                <dt>当前状态</dt><dd>{{ currentTask.status }}</dd>
                <dt>截止时间</dt><dd>{{ currentTask.dueAt || '-' }}</dd>
                <dt>说明</dt><dd>{{ currentTask.description || '暂无说明' }}</dd>
              </dl>
              <label>状态流转
                <select :value="currentTask.status" :disabled="isSampleRecord(currentTask)" @change="updateTaskStatus(currentTask, $event.target.value)">
                  <option>待处理</option><option>处理中</option><option>待确认</option><option>已完成</option><option>已关闭</option>
                </select>
              </label>
              <div class="detail-actions">
                <button class="primary" :disabled="isSampleRecord(currentTask)" @click="editTask(currentTask)">编辑任务</button>
                <button class="danger-action" :disabled="isSampleRecord(currentTask)" @click="deleteTask(currentTask)">删除任务</button>
              </div>
              <ol class="flow compact"><li>待处理</li><li>处理中</li><li>待确认</li><li>已完成</li><li>已关闭</li></ol>
            </aside>
          </div>
        </section>

        <section v-if="activeView === 'incidents'" class="panel">
          <div class="section-head">
            <div>
              <p class="muted">覆盖 P1-P4、影响范围、关联资产、War Room、恢复关闭与复盘入口。</p>
            </div>
            <div class="page-actions">
              <span class="pill danger">活跃事件 {{ activeIncidents.length }}</span>
              <button class="primary" @click="openIncidentForm"><span class="btn-icon">＋</span>新建事件</button>
            </div>
          </div>
          <div class="status-grid severity-grid">
            <article v-for="level in ['P1','P2','P3','P4']" :key="level" :class="level.toLowerCase()">
              <small>{{ level }}</small>
              <strong>{{ incidentLevelCounts[level] || 0 }}</strong>
              <span>{{ level === 'P1' ? '高危' : level === 'P2' ? '重要' : level === 'P3' ? '一般' : '观察' }}</span>
            </article>
          </div>
          <section v-if="incidentFormOpen" class="editor-panel" tabindex="-1" @focusout="closeEditorOnFocusOut($event, closeIncidentForm)">
            <h3>{{ newIncident.id ? '编辑事件' : '新建事件' }}</h3>
            <div class="form-grid">
              <input v-model="newIncident.title" placeholder="事件标题" />
              <select v-model="newIncident.level"><option>P1</option><option>P2</option><option>P3</option><option>P4</option></select>
              <input v-model="newIncident.owner" placeholder="负责人" />
              <input v-model="newIncident.business" placeholder="所属业务" />
              <select v-model="newIncident.status"><option>新建</option><option>处理中</option><option>已恢复</option><option>已关闭</option></select>
              <input v-model="newIncident.startedAt" placeholder="开始时间" />
              <input v-model="newIncident.recoveredAt" placeholder="恢复时间" />
              <textarea v-model="newIncident.summary" class="form-wide" rows="4" placeholder="事件摘要：描述影响范围、初步原因、当前处置动作和下一步计划"></textarea>
              <button class="primary" @click="saveIncident">{{ newIncident.id ? '保存修改' : '创建事件' }}</button>
              <button @click="closeIncidentForm">取消</button>
            </div>
          </section>
          <div class="work-layout" :class="{ 'single-column': !currentIncident }">
            <div class="table-wrap">
              <table>
                <thead><tr><th>事件</th><th>等级</th><th>状态</th><th>负责人</th><th>业务</th><th>操作</th></tr></thead>
                <tbody>
                  <tr v-for="incident in displayIncidents" :key="incident.id" :class="{ selected: currentIncident?.id === incident.id }" class="clickable-row" tabindex="0" @click="chooseIncident(incident)" @keyup.enter="chooseIncident(incident)">
                    <td>{{ incident.title }}</td>
                    <td><span class="pill danger">{{ incident.level }}</span></td>
                    <td><span class="pill">{{ incident.status }}</span></td>
                    <td>{{ incident.owner || '-' }}</td>
                    <td>{{ incident.business || '-' }}</td>
                    <td class="row-actions"><button class="link" @click.stop="chooseIncident(incident)">详情</button><button class="link" :disabled="isSampleRecord(incident)" @click.stop="editIncident(incident)">编辑</button><button class="link danger-text" :disabled="isSampleRecord(incident)" @click.stop="deleteIncident(incident)">删除</button></td>
                  </tr>
                </tbody>
              </table>
            </div>
            <aside class="detail-card" v-if="currentIncident">
              <div class="detail-title">
                <h3>{{ currentIncident.level }} · {{ currentIncident.title }}</h3>
                <button class="detail-close" aria-label="隐藏事件详情" @click="hideIncidentDetail">隐藏</button>
              </div>
              <dl>
                <dt>当前状态</dt><dd>{{ currentIncident.status }}</dd>
                <dt>负责人</dt><dd>{{ currentIncident.owner || '-' }}</dd>
                <dt>所属业务</dt><dd>{{ currentIncident.business || '-' }}</dd>
                <dt>开始时间</dt><dd>{{ currentIncident.startedAt || '-' }}</dd>
                <dt>摘要</dt><dd>{{ currentIncident.summary || '暂无摘要' }}</dd>
              </dl>
              <label>事件状态
                <select :value="currentIncident.status" :disabled="isSampleRecord(currentIncident)" @change="updateIncidentStatus(currentIncident, $event.target.value)">
                  <option>新建</option><option>处理中</option><option>已恢复</option><option>已关闭</option>
                </select>
              </label>
              <div class="detail-actions">
                <button class="primary" disabled>War Room（灰度）</button>
                <button disabled>复盘（灰度）</button>
                <button :disabled="isSampleRecord(currentIncident)" @click="editIncident(currentIncident)">编辑事件</button>
                <button class="danger-action" :disabled="isSampleRecord(currentIncident)" @click="deleteIncident(currentIncident)">删除事件</button>
              </div>
              <ol class="flow compact"><li>新建</li><li>处理中</li><li>已恢复</li><li>已关闭</li></ol>
            </aside>
          </div>
        </section>

        <section v-if="activeView === 'permissions'" class="panel">
          <div class="section-head">
            <div>
              <p class="muted">{{ permissionTab === 'resources' ? '一期先展示菜单入口、资源范围和角色权限边界，配置能力后续再细化。' : '一期先启用超级管理员和运维工程师，其他角色与 SSO/LDAP 保留灰度入口。' }}</p>
            </div>
            <div class="page-actions">
              <span class="pill">账号密码登录</span>
              <button v-if="canManageUsers && permissionTab === 'users'" class="primary" @click="openUserForm"><span class="btn-icon">＋</span>新增用户</button>
            </div>
          </div>
          <div class="segmented-tabs">
            <button :class="{ active: permissionTab === 'users' }" @click="goToView('permissions', 'users')">用户与角色</button>
            <button :class="{ active: permissionTab === 'resources' }" @click="goToView('permissions', 'resources')">菜单与资源权限</button>
          </div>

          <template v-if="permissionTab === 'users'">
            <div class="role-grid">
              <article v-for="role in roleCards" :key="role.code" :class="['role-card', role.tone]">
                <span class="role-line-icon"><svg viewBox="0 0 24 24"><path v-for="path in iconPath(role.icon)" :key="path" :d="path" /></svg></span>
                <div>
                  <strong>{{ role.name }}</strong>
                  <small>{{ role.code }}</small>
                  <p>{{ role.desc }}</p>
                </div>
              </article>
            </div>

            <div class="work-layout permission-workspace">
              <section v-if="canManageUsers">
                <h3>账号列表</h3>
                <section v-if="userFormOpen" class="editor-panel" tabindex="-1" @focusout="closeEditorOnFocusOut($event, closeUserForm)">
                  <h3>{{ newUser.id ? '编辑用户' : '新增用户' }}</h3>
                  <div class="form-grid user-form">
                    <input v-model="newUser.username" placeholder="登录账号" />
                    <input v-model="newUser.displayName" placeholder="姓名 / 显示名" />
                    <input v-model="newUser.password" type="password" :placeholder="newUser.id ? '新密码（留空不修改）' : '初始密码'" />
                    <select v-model="newUser.role"><option value="ops_engineer">运维工程师</option><option value="super_admin">超级管理员</option></select>
                    <label class="inline-check"><input v-model="newUser.mustChangePassword" type="checkbox" />首次登录修改密码</label>
                    <button class="primary" @click="saveUser">{{ newUser.id ? '保存用户' : '新增用户' }}</button>
                    <button @click="closeUserForm">取消</button>
                  </div>
                </section>
                <div class="table-wrap">
                  <table>
                    <thead><tr><th>账号</th><th>姓名</th><th>角色</th><th>密码状态</th><th>状态</th><th>操作</th></tr></thead>
                    <tbody>
                      <tr v-for="user in displayUsers" :key="user.id">
                        <td>{{ user.username }}</td>
                        <td>{{ user.displayName }}</td>
                        <td><span v-for="role in user.roles" :key="role" class="pill">{{ role === 'super_admin' ? '超级管理员' : '运维工程师' }}</span></td>
                        <td><span :class="['pill', user.mustChangePassword ? 'danger' : 'success']">{{ user.mustChangePassword ? '需初始化' : '正常' }}</span></td>
                        <td><span class="pill success">启用</span></td>
                        <td class="row-actions"><button class="link" :disabled="isSampleRecord(user)" @click="editUser(user)">编辑</button><button class="link danger-text" :disabled="user.id === currentUserID() || isSampleRecord(user)" @click="deleteUser(user)">删除</button></td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </section>
              <section v-else class="access-note">
                <h3>当前权限</h3>
                <p class="muted">当前账号仅可查看一期权限边界说明。用户新增、角色调整、密码初始化策略和菜单资源权限由超级管理员维护。</p>
              </section>
              <aside class="detail-card">
                <h3>敏感凭据策略</h3>
                <p class="muted">资产与实例账号密码单独加密存储，列表页不显示。查看前必须输入统一二次校验密码。</p>
                <dl>
                  <dt>登录方式</dt><dd>一期账号密码</dd>
                  <dt>初始账号</dt><dd>admin / 首次登录修改初始化密码</dd>
                  <dt>SSO/LDAP</dt><dd>灰度占位，暂不接入</dd>
                  <dt>凭据查看</dt><dd>{{ credentialVerification.hasPassword ? '已配置统一校验密码' : '未配置，暂回退登录密码' }}</dd>
                </dl>
                <section v-if="canManageUsers" class="credential-policy">
                  <h4>统一二次校验密码</h4>
                  <input v-model="credentialVerification.password" type="password" placeholder="设置统一校验密码" />
                  <input v-model="credentialVerification.confirm" type="password" placeholder="再次确认校验密码" />
                  <button class="primary" @click="saveCredentialVerificationPassword">{{ credentialVerification.hasPassword ? '更新校验密码' : '设置校验密码' }}</button>
                  <p v-if="credentialVerification.message" :class="['inline', credentialVerification.message.includes('失败') || credentialVerification.message.includes('不') || credentialVerification.message.includes('至少') || credentialVerification.message.includes('无权') ? 'error' : 'muted']">{{ credentialVerification.message }}</p>
                </section>
              </aside>
            </div>
          </template>

          <template v-else>
            <section class="permission-matrix">
              <h3>菜单授权概览</h3>
              <div class="table-wrap">
                <table>
                  <thead><tr><th>菜单 / 资源</th><th>阶段</th><th>超级管理员</th><th>运维工程师</th><th>说明</th></tr></thead>
                  <tbody>
                    <tr v-for="row in menuPermissionRows" :key="row.menu">
                      <td>{{ row.menu }}</td>
                      <td><span :class="['pill', row.stage.includes('灰度') ? '' : 'success']">{{ row.stage }}</span></td>
                      <td>{{ row.superAdmin }}</td>
                      <td>{{ row.opsEngineer }}</td>
                      <td>{{ row.note }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </section>

            <section class="permission-matrix">
              <h3>资源权限矩阵</h3>
              <div class="table-wrap">
                <table>
                  <thead><tr><th>功能范围</th><th>超级管理员</th><th>运维工程师</th><th>说明</th></tr></thead>
                  <tbody>
                    <tr v-for="row in permissionRows" :key="row.scope">
                      <td>{{ row.scope }}</td>
                      <td><span class="pill success">{{ row.admin }}</span></td>
                      <td><span class="pill">{{ row.ops }}</span></td>
                      <td>{{ row.note }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </section>

            <div class="permission-grid gray-roles">
              <article><strong>灰度角色</strong><span>管理人员、SRE、研发、值班人员、只读访客先保留名称，后续再扩展细粒度权限。</span></article>
              <article><strong>认证灰度</strong><span>SSO/LDAP 保留配置入口，一期不影响账号密码登录上线。</span></article>
            </div>
          </template>
        </section>

        <section v-if="activeView === 'copilot-settings'" class="panel">
          <div class="section-head">
            <div>
              <p class="muted">一期先明确模型来源、上下文授权和审计边界；真实密钥托管、调用代理和用量统计后续接入后端。</p>
            </div>
            <div class="page-actions">
              <span class="pill success">建议态 AI</span>
              <button class="primary" @click="saveCopilotConfig">保存配置</button>
            </div>
          </div>

          <div class="copilot-config-layout">
            <section class="provider-grid">
              <article
                v-for="provider in copilotProviders"
                :key="provider.id"
                :class="['provider-card', { active: copilotConfig.provider === provider.id }]"
                @click="selectCopilotProvider(provider)"
              >
                <span>{{ provider.badge }}</span>
                <strong>{{ provider.name }}</strong>
                <small>{{ provider.desc }}</small>
              </article>
            </section>

            <section class="config-card">
              <div class="config-title">
                <span class="role-line-icon"><svg viewBox="0 0 24 24"><path v-for="path in iconPath('bot')" :key="path" :d="path" /></svg></span>
                <div>
                  <h3>{{ selectedCopilotProvider.name }}</h3>
                  <p class="muted">{{ selectedCopilotProvider.desc }}</p>
                </div>
              </div>
              <div class="form-grid copilot-form">
                <label>模型厂商
                  <select v-model="copilotConfig.provider">
                    <option value="local">本地模型</option>
                    <option value="openai">OpenAI GPT</option>
                    <option value="anthropic">Anthropic Claude</option>
                    <option value="google">Google Gemini</option>
                    <option value="compatible">OpenAI 兼容接口</option>
                  </select>
                </label>
                <label>API Endpoint
                  <input v-model="copilotConfig.endpoint" placeholder="模型服务地址" />
                </label>
                <label>模型名称
                  <input v-model="copilotConfig.model" placeholder="例如 gpt-4.1 / claude-3-7-sonnet" />
                </label>
                <label>API Key
                  <input v-model="copilotConfig.apiKey" type="password" placeholder="前端不保存真实生产密钥" />
                </label>
                <label>本地模型地址
                  <input v-model="copilotConfig.localEndpoint" placeholder="例如 http://localhost:11434" />
                </label>
                <label>本地模型
                  <input v-model="copilotConfig.localModel" placeholder="例如 qwen2.5:7b" />
                </label>
                <label>Temperature
                  <input v-model="copilotConfig.temperature" />
                </label>
                <label>Max Tokens
                  <input v-model="copilotConfig.maxTokens" />
                </label>
              </div>
            </section>

            <section class="config-card">
              <h3>上下文授权</h3>
              <div class="context-grid">
                <label class="inline-check"><input v-model="copilotConfig.enableAssetContext" type="checkbox" />资产与实例上下文</label>
                <label class="inline-check"><input v-model="copilotConfig.enableIncidentContext" type="checkbox" />事件影响上下文</label>
                <label class="inline-check"><input v-model="copilotConfig.enableTaskContext" type="checkbox" />任务闭环上下文</label>
                <label class="inline-check"><input v-model="copilotConfig.enableOncallContext" type="checkbox" />值班与交接上下文</label>
                <label class="inline-check"><input v-model="copilotConfig.auditEnabled" type="checkbox" />启用问答审计</label>
              </div>
              <div class="ai-guardrails">
                <article><strong>权限感知</strong><span>Copilot 只应读取当前账号有权访问的数据。</span></article>
                <article><strong>建议态输出</strong><span>自动化执行前必须有人审、权限、审计和回滚策略。</span></article>
                <article><strong>密钥托管</strong><span>生产 API Key 不落前端，后续由后端加密保存并代理调用。</span></article>
              </div>
            </section>
          </div>
        </section>

      </template>
    </main>

    <section v-if="hasAppAccess && copilotOpen" class="copilot" :class="{ expanded: copilotExpanded }">
      <header>
        <div class="copilot-title">
          <span class="copilot-logo"><svg viewBox="0 0 24 24"><path v-for="path in iconPath('bot')" :key="path" :d="path" /></svg></span>
          <div>
            <strong>OpsCore AI Copilot</strong>
            <small>资产、事件、值班与任务助手</small>
          </div>
        </div>
        <div class="copilot-actions">
          <button :title="copilotExpanded ? '还原窗口' : '放大窗口'" @click="toggleCopilotSize">
            <svg viewBox="0 0 24 24"><path v-for="path in iconPath(copilotExpanded ? 'restore' : 'maximize')" :key="path" :d="path" /></svg>
          </button>
          <button title="隐藏 Copilot" @click="hideCopilot">
            <svg viewBox="0 0 24 24"><path v-for="path in iconPath('minimize')" :key="path" :d="path" /></svg>
          </button>
        </div>
      </header>
      <div class="copilot-body">
        <p>我可以帮你查询 CMDB 资产、定位活跃事件、汇总今日值班和待处理任务。</p>
        <div v-for="(message, index) in copilotMessages" :key="index" :class="['chat', message.role]">{{ message.text }}</div>
      </div>
      <div class="copilot-input">
        <input v-model="copilotQuestion" placeholder="输入问题，例如：查询支付服务关联资产" @keyup.enter="askCopilot" />
        <button class="primary" @click="askCopilot">发送</button>
      </div>
    </section>
    <button v-else-if="hasAppAccess" class="copilot-button" title="打开 AI Copilot" @click="openCopilot">
      <svg viewBox="0 0 24 24"><path v-for="path in iconPath('bot')" :key="path" :d="path" /></svg>
    </button>
  </div>
</template>
