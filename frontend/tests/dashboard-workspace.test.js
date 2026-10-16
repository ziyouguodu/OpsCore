import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import { buildPreviewPath, dateISO, formatRate, greeting, normalizePreview, normalizeWorkspace, searchWorkspace, trendGeometry } from '../src/dashboard-workspace.js'
import { iconPath } from '../src/ui/icons.js'

const dashboard = await readFile(new URL('../src/components/DashboardView.vue', import.meta.url), 'utf8')
const dialog = await readFile(new URL('../src/components/WorkspaceDialog.vue', import.meta.url), 'utf8')
const layout = await readFile(new URL('../src/workspace-layout.css', import.meta.url), 'utf8')
const css = await readFile(new URL('../src/review-framework.css', import.meta.url), 'utf8')
const prototype = await readFile(new URL('../../deliverables/home-glass-framework-review.html', import.meta.url), 'utf8')

test('never lets preview data replace real events or duty', () => {
  assert.throws(() => normalizeWorkspace({ source: 'demo', incidents: [{ id: 'fake' }] }), /来源/)
  assert.throws(() => normalizeWorkspace({ source: 'live' }), /来源/)
  assert.throws(() => normalizePreview({ source: 'real' }), /来源/)
  const real = normalizeWorkspace({ source: 'real', generatedAt: '2026-10-14T08:00:00+08:00', incidents: [{ id: 123 }], statistics: { activeIncidentCount: 25 }, duty: { source: 'real', primary: null, backup: { name: '备值', since: '09:15', until: '19:45' }, currentPeople: [{ name: '不能按数组顺序推主班' }], pendingCount: 7, handovers: [{ id: 1, from: '甲', to: '乙', time: '2026-10-14 09:15', content: '交接', complete: false }] } })
  assert.equal(real.incidents[0].id, 123)
  assert.equal(real.statistics.activeIncidentCount, 25)
  assert.equal(real.duty.primary, null)
  assert.equal(real.duty.backup.name, '备值')
  assert.equal(real.duty.pendingCount, 7)
  assert.equal(real.duty.handovers[0].time, '2026-10-14 09:15')
  assert.equal(real.trend.source, 'real')
  assert.equal(real.trend.status, 'unavailable')
  assert.match(dashboard, /props\.workspace\.incidents/)
  assert.match(dashboard, /props\.workspace\.duty\.primary/)
  assert.match(dashboard, /props\.workspace\.duty\.backup/)
  assert.match(dashboard, /workspace\.duty\.pendingCount/)
  assert.doesNotMatch(dashboard, /people\[0\]|people\[1\]|priorityItems|sampleIncidents|sampleOncalls/)
})

test('builds the exact system, period, from and to preview contract', () => {
  assert.equal(buildPreviewPath(), '/dashboard/preview?period=today&system=all')
  const url = new URL(buildPreviewPath({ period: 'custom', system: 'payment', from: '2026-10-01', to: '2026-10-14' }), 'http://localhost')
  assert.deepEqual([...url.searchParams.keys()], ['period', 'system', 'from', 'to'])
  assert.equal(url.searchParams.get('system'), 'payment')
  assert.equal(url.searchParams.get('from'), '2026-10-01')
  assert.equal(url.searchParams.get('to'), '2026-10-14')
  for (const period of ['today', '7', '30']) assert.equal(buildPreviewPath({ period, system: 'orders', from: 'ignored', to: 'ignored' }), `/dashboard/preview?period=${period}&system=orders`)
})

test('retains null observations and breaks the trend line instead of filling zero', () => {
  const points = [99.94, null, 99.96].map((availability, hour) => ({ time: `2026-10-14T0${hour}:00:00+08:00`, label: `0${hour}:00`, availability }))
  const geometry = trendGeometry(points, 360)
  assert.equal(geometry.nodes[1].y, null)
  assert.equal(geometry.segments.length, 2)
  assert.ok(Math.abs(geometry.mean - 99.95) < 1e-10)
  assert.equal(trendGeometry([{ time: points[0].time, label: '00:00', availability: null }], 360).mean, null)
  assert.equal(formatRate(null), '--')
  assert.equal(formatRate(0), '0.00%')
})

test('resizes coordinates rather than stretching SVG text', () => {
  const points = Array.from({ length: 24 }, (_, hour) => ({ time: `2026-10-14T${String(hour).padStart(2, '0')}:00:00+08:00`, label: `${String(hour).padStart(2, '0')}:00`, availability: 99.96 }))
  const narrow = trendGeometry(points, 360)
  const wide = trendGeometry(points, 900)
  assert.ok(wide.nodes.at(-1).x > narrow.nodes.at(-1).x)
  assert.ok(narrow.labels.length < wide.labels.length)
  assert.equal(narrow.labels[0].label, '00:00')
  assert.equal(narrow.labels.at(-1).label, '23:00')
  assert.equal(trendGeometry([{ time: '2026-10-01T00:00:00+08:00', label: '2026-10-01', availability: 99.9 }], 360, 245, '7').labels[0].label, '10-01')
  assert.match(dashboard, /new ResizeObserver/)
  assert.doesNotMatch(dashboard, /preserveAspectRatio="none"|v-html/)
})

test('uses the local calendar date and actual greeting hour', () => {
  assert.equal(dateISO(new Date(2026, 9, 14, 0, 15)), '2026-10-14')
  assert.equal(greeting(new Date(2026, 9, 14, 9)), '上午好')
  assert.equal(greeting(new Date(2026, 9, 14, 15)), '下午好')
  assert.equal(greeting(new Date(2026, 9, 14, 21)), '晚上好')
  assert.doesNotMatch(dashboard, /上午好，admin|当前环境：生产/)
})

test('queries existing resources with keyword and preserves permission failures', async () => {
  const calls = []
  const groups = await searchWorkspace(async path => {
    calls.push(path)
    if (path.startsWith('/middleware')) throw Object.assign(new Error('forbidden'), { status: 403 })
    return { items: [{ title: '<script>not executed</script>' }], total: 17 }
  }, '生产 & 订单')
  assert.equal(calls.length, 4)
  assert.equal(new URL(calls[0], 'http://localhost').searchParams.get('keyword'), '生产 & 订单')
  assert.equal(groups[1].error, '无访问权限')
  assert.equal(groups[0].total, 17)
  await assert.rejects(searchWorkspace(async () => { throw Object.assign(new Error('expired'), { status: 401 }) }, '订单'), /expired/)
})

test('uses native dialog focus lifecycle for Escape, cancel and focus return', () => {
  assert.match(dialog, /<dialog/)
  assert.match(dialog, /showModal\(\)/)
  assert.match(dialog, /@close="onClose"/)
  assert.match(dialog, /returnFocus\.focus\(\)/)
  assert.match(dashboard, /validDates/)
  assert.match(dashboard, /确认演示意向/)
  assert.match(dashboard, /source: demo/)
  assert.match(dashboard, /没有调用真实 AI/)
})

test('keeps exact final prototype theme values and glass border shape', () => {
  for (const marker of [':root{--bg:#071524;', ':root[data-theme="graphite"]', ':root[data-theme="aurora"]']) {
    const declaration = prototype.slice(prototype.indexOf(marker), prototype.indexOf('}', prototype.indexOf(marker)) + 1)
    assert.ok(css.includes(declaration), marker)
  }
  assert.match(css, /mask-composite:exclude/)
  assert.match(css, /border-radius:10px/)
  assert.match(layout, /grid-template-columns: 188px minmax\(0,1fr\)/)
  assert.match(layout, /grid-template-columns: 172px minmax\(0,1fr\)/)
  assert.match(layout, /grid-template-columns: 158px minmax\(0,1fr\)/)
  assert.doesNotMatch(layout, /overflow-x:\s*hidden/)
  assert.ok(css.includes('.review-shell :where(.review-dashboard,.review-topbar,.review-sidebar,.review-dialog) button'))
  assert.ok(!css.includes('.review-shell :is(.review-dashboard,.review-topbar,.review-sidebar,.review-dialog) button'))
})

test('resolves actual activity, copilot and shield paths, not dashboard fallback', () => {
  for (const name of ['activity', 'copilot', 'shield']) assert.notDeepEqual(iconPath(name), iconPath('dashboard'))
})

test('keeps backend preview catalogs, analysis and succeeded counts intact', () => {
  const preview = normalizePreview({ source: 'demo', executionAllowed: false, notice: '虚构演示', businessSystems: [{ id: 'payment', name: '支付交易平台', assetCount: 24, serviceCount: 8 }], automation: { source: 'demo', succeeded: 148, failed: 2, successRate: 98.7 }, insights: [{ riskId: 'risk-1', analysis: '预设分析', plan: '只读核验；人工审批', status: 'suggestionOnly' }], trend: { source: 'demo', includesAssetHealth: false, timezone: 'Asia/Shanghai', points: [{ time: '2026-10-14T00:00:00+08:00', label: '00:00', availability: null }] } })
  assert.equal(preview.executionAllowed, false)
  assert.equal(preview.businessSystems[0].id, 'payment')
  assert.equal(preview.automation.succeeded, 148)
  assert.equal(preview.insights[0].analysis, '预设分析')
  assert.equal(preview.trend.includesAssetHealth, false)
  assert.equal(trendGeometry(preview.trend.points, 360).mean, null)
  assert.match(dashboard, /selectedDemoSystem = ref\('all'\)/)
  assert.match(dashboard, /system: selectedDemoSystem\.value, from: start\.value, to: end\.value/)
  assert.match(dashboard, /insight\?\.analysis/)
  assert.match(dashboard, /automation\?\.succeeded/)
  assert.doesNotMatch(dashboard, /demo\?\.availability|source: live/)
})
