import { computed } from 'vue'

import { summarizeDashboardMetrics } from '../dashboard-metrics'

function countBy(items, field, defaults = []) {
  const counts = Object.fromEntries(defaults.map((item) => [item, 0]))
  for (const item of items) {
    const key = item[field] || '未设置'
    counts[key] = (counts[key] || 0) + 1
  }
  return counts
}

export function useDashboardViewModel({
  demoDataEnabled,
  state,
  listMeta,
  displayAssets,
  displayMiddleware,
  displayOncalls,
  displayTasks,
  displayIncidents
}) {
  const dashboardMetrics = computed(() => {
    if (demoDataEnabled && state.dashboard.assetCount === 0) {
      return summarizeDashboardMetrics({
        assets: displayAssets.value,
        middleware: displayMiddleware.value,
        tasks: displayTasks.value,
        incidents: displayIncidents.value
      })
    }
    const managedTotal = state.dashboard.assetHealthyCount + state.dashboard.assetAbnormalCount
    const taskTotal = state.dashboard.taskClosedCount + state.dashboard.taskOpenCount
    const incidentTotal = state.dashboard.incidentClosedCount + state.dashboard.activeIncidentCount
    return {
      assetHealthy: state.dashboard.assetHealthyCount,
      assetAbnormal: state.dashboard.assetAbnormalCount,
      assetHealthRate: managedTotal ? Math.round((state.dashboard.assetHealthyCount / managedTotal) * 100) : null,
      taskClosed: state.dashboard.taskClosedCount,
      taskOpen: state.dashboard.taskOpenCount,
      taskClosureRate: taskTotal ? Math.round((state.dashboard.taskClosedCount / taskTotal) * 100) : null,
      incidentClosed: state.dashboard.incidentClosedCount,
      incidentActive: state.dashboard.activeIncidentCount,
      incidentClosureRate: incidentTotal ? Math.round((state.dashboard.incidentClosedCount / incidentTotal) * 100) : null,
      averageResponseMinutes: state.dashboard.responseSampleCount ? Math.round(state.dashboard.responseMinutesTotal / state.dashboard.responseSampleCount) : null,
      responseSampleCount: state.dashboard.responseSampleCount
    }
  })

  const taskStatusCounts = computed(() => demoDataEnabled && !listMeta.tasks.total
    ? countBy(displayTasks.value, 'status', ['待处理', '处理中', '待确认', '已完成', '已关闭'])
    : { 待处理: 0, 处理中: 0, 待确认: 0, 已完成: 0, 已关闭: 0, ...listMeta.tasks.counts })
  const incidentLevelCounts = computed(() => demoDataEnabled && !listMeta.incidents.total
    ? countBy(displayIncidents.value, 'level', ['P1', 'P2', 'P3', 'P4'])
    : { P1: 0, P2: 0, P3: 0, P4: 0, ...listMeta.incidents.counts })
  const activeTasks = computed(() => displayTasks.value.filter((item) => !['已完成', '已关闭'].includes(item.status)))
  const activeIncidents = computed(() => displayIncidents.value.filter((item) => item.status !== '已关闭'))
  const todayOncall = computed(() => displayOncalls.value[0] || {})

  const dashboardAssetCount = computed(() => {
    if (demoDataEnabled && !state.assets.length && !state.middleware.length && state.dashboard.assetCount === 0) {
      return displayAssets.value.length + displayMiddleware.value.length
    }
    return state.dashboard.assetCount
  })
  const dashboardOncallCount = computed(() => demoDataEnabled && !state.oncalls.length && state.dashboard.todayOnCallCount === 0
    ? displayOncalls.value.length
    : state.dashboard.todayOnCallCount)
  const dashboardTaskCount = computed(() => demoDataEnabled && !state.tasks.length && state.dashboard.activeTaskCount === 0
    ? activeTasks.value.length
    : state.dashboard.activeTaskCount)
  const dashboardIncidentCount = computed(() => demoDataEnabled && !state.incidents.length && state.dashboard.activeIncidentCount === 0
    ? activeIncidents.value.length
    : state.dashboard.activeIncidentCount)

  const assetKpiBars = computed(() => {
    const counts = state.dashboard.assetTypeCounts || {}
    const databaseKinds = ['MySQL', 'PostgreSQL', '达梦']
    const serverCount = demoDataEnabled && state.dashboard.assetCount === 0 ? displayAssets.value.length : (counts['服务器'] || 0)
    const databaseCount = demoDataEnabled && state.dashboard.assetCount === 0 ? displayMiddleware.value.filter((item) => databaseKinds.includes(item.kind)).length : (counts['数据库'] || 0)
    const middlewareCount = demoDataEnabled && state.dashboard.assetCount === 0 ? Math.max(displayMiddleware.value.length - databaseCount, 0) : (counts['中间件'] || 0)
    const items = [
      { label: '服务器', value: serverCount, tone: 'server' },
      { label: '数据库', value: databaseCount, tone: 'database' },
      { label: '中间件', value: middlewareCount, tone: 'middleware' }
    ]
    const max = Math.max(...items.map((item) => item.value), 1)
    return items.map((item) => ({ ...item, height: `${Math.max(24, Math.round((item.value / max) * 78))}%` }))
  })
  const taskKpiCards = computed(() => {
    const items = [
      { label: '待处理', value: taskStatusCounts.value['待处理'] || 0, color: '#f59e0b' },
      { label: '处理中', value: taskStatusCounts.value['处理中'] || 0, color: '#2563eb' },
      { label: '待确认', value: taskStatusCounts.value['待确认'] || 0, color: '#14b8a6' }
    ]
    const total = Math.max(items.reduce((sum, item) => sum + item.value, 0), 1)
    return items.map((item) => ({ ...item, width: `${Math.max(10, Math.round((item.value / total) * 100))}%` }))
  })
  const activeIncidentLevelCounts = computed(() => demoDataEnabled && state.dashboard.activeIncidentCount === 0
    ? countBy(activeIncidents.value, 'level', ['P1', 'P2', 'P3', 'P4'])
    : { P1: 0, P2: 0, P3: 0, P4: 0, ...(state.dashboard.incidentLevelCounts || {}) })
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

  return {
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
  }
}
