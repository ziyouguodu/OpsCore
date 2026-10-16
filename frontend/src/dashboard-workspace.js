export const periods = [{ value: 'today', label: '今日' }, { value: '7', label: '近 7 天' }, { value: '30', label: '近 30 天' }, { value: 'custom', label: '自定义' }]

export function emptyWorkspace() {
  return { source: 'real', statistics: {}, businessSystems: [], incidents: [], tasks: [], duty: { source: 'real', primary: null, backup: null, currentPeople: [], handovers: [], pendingCount: 0 }, trend: { source: 'real', status: 'unavailable', points: [] } }
}

export function normalizeWorkspace(payload) {
  if (payload?.source !== 'real') throw new Error('首页业务数据来源不匹配')
  const empty = emptyWorkspace()
  return { ...empty, ...payload, statistics: payload.statistics || {}, duty: { ...empty.duty, ...payload.duty }, trend: { ...empty.trend, ...payload.trend } }
}

export function normalizePreview(payload) {
  if (payload?.source !== 'demo') throw new Error('演示数据来源不匹配')
  return { ...payload, businessSystems: payload.businessSystems || [], risks: payload.risks || [], services: payload.services || [], insights: payload.insights || [], notifications: payload.notifications || [], trend: { points: [], ...payload.trend } }
}

export function buildPreviewPath({ period = 'today', system = 'all', from = '', to = '' } = {}) {
  const query = new URLSearchParams({ period, system })
  if (period === 'custom') { query.set('from', from); query.set('to', to) }
  return `/dashboard/preview?${query}`
}

export function formatRate(value) {
  return Number.isFinite(value) ? `${value.toFixed(2)}%` : '--'
}

export function greeting(date = new Date()) {
  const hour = date.getHours()
  return hour < 6 ? '凌晨好' : hour < 12 ? '上午好' : hour < 18 ? '下午好' : '晚上好'
}

export function dateISO(date = new Date()) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

export function trendGeometry(points, width, height = 245, period = 'today') {
  const left = 67, right = Math.max(left + 1, width - 24), top = 28, bottom = height - 36
  const values = points.map(point => point.availability).filter(Number.isFinite)
  const minimum = Math.min(99.85, ...values), maximum = Math.max(100, ...values)
  const y = value => bottom - (value - minimum) / (maximum - minimum) * (bottom - top)
  const x = index => left + index * (right - left) / Math.max(1, points.length - 1)
  const nodes = points.map((point, index) => ({ ...point, x: x(index), y: Number.isFinite(point.availability) ? y(point.availability) : null }))
  const segments = []
  let current = []
  for (const point of nodes) {
    if (point.y === null) { if (current.length) segments.push(current); current = [] }
    else current.push(point)
  }
  if (current.length) segments.push(current)
  const step = Math.max(1, Math.ceil(points.length / Math.max(2, Math.floor((right - left) / 64))))
  const labels = nodes.filter((_, index) => index % step === 0 || index === nodes.length - 1).map(point => ({
    x: point.x,
    label: period === 'today' ? point.label : point.label.slice(5)
  }))
  return { nodes, segments: segments.map(segment => segment.map(point => `${point.x},${point.y}`).join(' ')), labels, ticks: Array.from({ length: 4 }, (_, index) => { const value = minimum + (maximum - minimum) * index / 3; return { value, y: y(value) } }), left, right, bottom, targetY: y(99.9), mean: values.length ? values.reduce((sum, value) => sum + value, 0) / values.length : null }
}

export async function searchWorkspace(request, keyword) {
  const resources = [{ path: 'assets', label: '资产', target: 'cmdb' }, { path: 'middleware', label: '实例', target: 'middleware' }, { path: 'tasks', label: '任务', target: 'tasks' }, { path: 'incidents', label: '事件', target: 'incidents' }]
  return Promise.all(resources.map(async resource => {
    try {
      const result = await request(`/${resource.path}?${new URLSearchParams({ keyword, page: '1', pageSize: '10' })}`)
      return { ...resource, source: 'live', items: result.items || [], total: result.total || 0 }
    } catch (error) {
      if (error.status === 401) throw error
      return { ...resource, source: 'live', items: [], error: error.status === 403 ? '无访问权限' : error.message }
    }
  }))
}
