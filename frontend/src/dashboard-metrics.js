const healthyStatuses = new Set(['运行中', '正常', '启用'])
const closedTaskStatuses = new Set(['已完成', '已关闭'])

function rate(part, total) {
  return total > 0 ? Math.round((part / total) * 100) : null
}

function responseMinutes(incident) {
  if (!incident.startedAt || !incident.recoveredAt) return null
  const started = Date.parse(incident.startedAt)
  const recovered = Date.parse(incident.recoveredAt)
  if (!Number.isFinite(started) || !Number.isFinite(recovered) || recovered < started) return null
  return (recovered - started) / 60000
}

export function summarizeDashboardMetrics({ assets = [], middleware = [], tasks = [], incidents = [] }) {
  const managed = [...assets, ...middleware]
  const assetHealthy = managed.filter((item) => healthyStatuses.has(item.status)).length
  const taskClosed = tasks.filter((item) => closedTaskStatuses.has(item.status)).length
  const incidentClosed = incidents.filter((item) => item.status === '已关闭').length
  const responseSamples = incidents.map(responseMinutes).filter((value) => value !== null)
  const averageResponseMinutes = responseSamples.length
    ? Math.round(responseSamples.reduce((sum, value) => sum + value, 0) / responseSamples.length)
    : null

  return {
    assetHealthy,
    assetAbnormal: managed.length - assetHealthy,
    assetHealthRate: rate(assetHealthy, managed.length),
    taskClosed,
    taskOpen: tasks.length - taskClosed,
    taskClosureRate: rate(taskClosed, tasks.length),
    incidentClosed,
    incidentActive: incidents.length - incidentClosed,
    incidentClosureRate: rate(incidentClosed, incidents.length),
    averageResponseMinutes,
    responseSampleCount: responseSamples.length
  }
}
