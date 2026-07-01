import assert from 'node:assert/strict'
import test from 'node:test'

let helpers = {}
try {
  helpers = await import('../src/dashboard-metrics.js')
} catch {
  helpers = {}
}

test('summarizes dashboard health and closure metrics from live records', () => {
  assert.equal(typeof helpers.summarizeDashboardMetrics, 'function')
  const result = helpers.summarizeDashboardMetrics({
    assets: [{ status: '运行中' }, { status: '维护中' }],
    middleware: [{ status: '运行中' }, { status: '异常' }],
    tasks: [{ status: '已完成' }, { status: '已关闭' }, { status: '处理中' }],
    incidents: [
      { status: '已关闭', startedAt: '2026-06-29T08:00:00Z', recoveredAt: '2026-06-29T08:20:00Z' },
      { status: '处理中', startedAt: '2026-06-29T09:00:00Z', recoveredAt: '' }
    ]
  })

  assert.deepEqual(result, {
    assetHealthy: 2,
    assetAbnormal: 2,
    assetHealthRate: 50,
    taskClosed: 2,
    taskOpen: 1,
    taskClosureRate: 67,
    incidentClosed: 1,
    incidentActive: 1,
    incidentClosureRate: 50,
    averageResponseMinutes: 20,
    responseSampleCount: 1
  })
})

test('returns null rates when there is no live data', () => {
  const result = helpers.summarizeDashboardMetrics({ assets: [], middleware: [], tasks: [], incidents: [] })

  assert.equal(result.assetHealthRate, null)
  assert.equal(result.taskClosureRate, null)
  assert.equal(result.incidentClosureRate, null)
  assert.equal(result.averageResponseMinutes, null)
  assert.equal(result.responseSampleCount, 0)
})
