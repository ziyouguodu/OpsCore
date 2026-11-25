import { expect, test } from '@playwright/test'

const primary = { id: '1', name: '真实值班人', role: '主值班', since: '09:15', until: '19:45' }
const backup = { id: '2', name: '真实备班人', role: '备班', since: '09:15', until: '19:45' }
const workspace = { source: 'real', generatedAt: '2026-10-14T10:00:00+08:00', statistics: { assetCount: 357, activeIncidentCount: 18, activeTaskCount: 12, assetTypeCounts: { 服务器: 200, 数据库: 57, 中间件: 100 }, incidentLevelCounts: { P1: 1, P2: 2, P3: 15 } }, businessSystems: ['全局业务系统'], incidents: [{ id: 42, title: '真实事件不被演示替换', level: 'P1', owner: '真实负责人', status: '处理中' }], tasks: [], duty: { source: 'real', revision: 3, date: '2026-10-14', primary, backup, currentPeople: [backup, primary], todayAssignment: { primary, backup }, nextAssignment: null, pendingCount: 7, pendingHandovers: [], handovers: [{ id: '1', from: '交班人', to: '接班人', time: '2026-10-14 09:15', content: '真实交接记录', complete: false }] }, trend: { source: 'real', status: 'unavailable', points: [] } }
const preview = { source: 'demo', executionAllowed: false, notice: '虚构演示数据，不执行生产操作', businessSystems: [{ id: 'payment', name: '支付交易平台', assetCount: 24, serviceCount: 8 }, { id: 'orders', name: '订单中心', assetCount: 31, serviceCount: 6 }, { id: 'portal', name: '企业门户', assetCount: 18, serviceCount: 4 }], alerts: { source: 'demo', status: 'pendingIntegration', urgentCount: 3, levelCounts: { P1: 1, P2: 2 } }, automation: { source: 'demo', successRate: 98.7, succeeded: 148, failed: 2 }, risks: [{ id: 'risk-1', level: 'P1', title: '虚构风险甲', asset: 'demo-host', owner: '演示负责人', status: '待研判', impact: '演示业务', time: '8 分钟前', evidence: '演示证据甲', analysis: '虚构建议甲', plan: '人工核验；人工审批' }, { id: 'risk-2', level: 'P2', title: '虚构风险乙', asset: 'demo-host-2', owner: '演示负责人', status: '待研判', impact: '演示业务乙', time: '24 分钟前', evidence: '演示证据乙', analysis: '虚构建议乙', plan: '只读核验；评估影响' }], insights: [{ riskId: 'risk-2', title: '风险乙预设洞察', analysis: '虚构建议乙：来自洞察接口', evidence: '洞察证据乙', plan: '核实只读证据；人工审阅影响', status: 'suggestionOnly' }], services: [{ name: '演示服务', status: 'healthy', availability: 99.98, p95Ms: 48 }], notifications: [{ riskId: 'risk-1', title: '虚构通知甲', time: '8 分钟前' }] }

async function setup(page, overrides = {}) {
  await page.addInitScript(() => sessionStorage.setItem('opscore.token', 'dashboard-test-token'))
  await page.route('**/api/**', async route => {
    const url = new URL(route.request().url())
    let body
    if (url.pathname === '/api/auth/me') body = { id: 1, username: 'review-user', displayName: '评审用户', roles: ['ops_engineer'], mustChangePassword: false }
    else if (url.pathname === '/api/dashboard/workspace') {
      expect([...url.searchParams.keys()]).toEqual([])
      body = { ...workspace, ...overrides }
    }
    else if (url.pathname === '/api/dashboard/preview') {
      const period = url.searchParams.get('period'), system = url.searchParams.get('system')
      expect(['all', 'payment', 'orders', 'portal']).toContain(system)
      expect([...url.searchParams.keys()].sort()).toEqual(period === 'custom' ? ['from', 'period', 'system', 'to'] : ['period', 'system'])
      const count = period === 'today' ? 24 : 7
      body = { ...preview, trend: { source: 'demo', status: 'demo', system, period, from: url.searchParams.get('from') || '2026-10-14', to: url.searchParams.get('to') || '2026-10-14', timezone: 'Asia/Shanghai', granularity: period === 'today' ? 'hour' : 'day', metric: 'coreServiceAvailability', unit: '%', definition: '成功请求 / 总请求', includesAssetHealth: false, target: 99.9, notice: '虚构观测', points: Array.from({ length: count }, (_, index) => ({ time: `2026-10-14T${String(index).padStart(2, '0')}:00:00+08:00`, label: period === 'today' ? `${String(index).padStart(2, '0')}:00` : `2026-10-${String(index + 1).padStart(2, '0')}`, availability: index === 5 ? null : 99.94 + index / 1000 })) } }
    } else if (url.pathname === '/api/duty-center') body = { revision: 1, data: {} }
    else if (/\/(assets|middleware|tasks|incidents)$/.test(url.pathname)) body = { items: [], total: 0, pageCount: 1, options: { business: [], networkZone: [] }, counts: {} }
    else body = []
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(body) })
  })
  await page.goto('/#dashboard')
  await expect(page.locator('.review-dashboard')).toBeVisible()
}

test('preserves exact theme choices, six simple KPIs and readable 390px layout', async ({ page }) => {
  await setup(page)
  await expect(page.locator('.review-dashboard .kpi')).toHaveCount(6)
  await expect(page.locator('.review-dashboard .hero h1')).toContainText('评审用户')
  await expect(page.locator('.review-dashboard footer, .review-dashboard .footer, .brand-line')).toHaveCount(0)
  await expect(page.locator('.topbar h1')).toHaveCount(0)
  await expect(page.locator('h1')).toHaveCount(1)
  await expect(page.locator('.review-dashboard')).not.toContainText('生产环境')
  await expect(page.locator('.review-sidebar .nav.child').first()).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)')
  await expect(page.locator('.review-sidebar .nav.child').first()).toHaveCSS('border-color', 'rgba(0, 0, 0, 0)')
  await expect(page.locator('.review-sidebar .nav.child').first()).toHaveCSS('font-size', '11px')
  await expect(page.locator('.summary')).toHaveCSS('border-width', '0px 0px 0px 1px')
  await expect(page.locator('.summary')).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)')
  await expect(page.locator('.event-name').first()).toHaveCSS('border-width', '0px')
  await expect(page.locator('.dashboard-secondary-metrics')).toHaveCount(0)
  for (const [theme, background] of [['deepsea', '#071524'], ['graphite', '#11121d'], ['aurora', '#080b1d']]) {
    await page.getByLabel('选择主题色调').selectOption(theme)
    expect(await page.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue('--bg').trim())).toBe(background)
  }
  await page.setViewportSize({ width: 390, height: 844 })
  for (const card of await page.locator('.review-dashboard .kpi').all()) expect((await card.boundingBox()).width).toBeGreaterThanOrEqual(160)
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390)
  await page.getByRole('button', { name: '打开导航' }).click()
  await expect(page.locator('.review-sidebar')).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.locator('.review-sidebar')).toBeHidden()
  await expect(page.getByRole('button', { name: '打开导航' })).toBeFocused()
})

test('aligns KPI sections in each row and keeps long duty names readable', async ({ page }) => {
  const longPrimary = '欧阳长名字值班负责人'
  const longBackup = '司徒长名字备班负责人'
  await setup(page, { duty: { ...workspace.duty, primary: { ...primary, name: longPrimary }, backup: { ...backup, name: longBackup } } })
  for (const width of [1920, 1440, 1280, 390]) {
    await page.setViewportSize({ width, height: 900 })
    await page.mouse.move(0, 0)
    await page.evaluate(() => document.fonts.ready)
    const geometry = await page.locator('.kpis > .kpi').evaluateAll(cards => cards.map(card => {
      const rect = element => { const r = element.getBoundingClientRect(); return { top: r.top, left: r.left, right: r.right, bottom: r.bottom } }
      return { card: rect(card), head: rect(card.querySelector('.kpi-head')), body: rect(card.querySelector('.kpi-body')), foot: rect(card.querySelector('.kpi-foot')) }
    }))
    const rows = []
    for (const item of geometry) {
      const row = rows.find(row => Math.abs(row[0].card.top - item.card.top) <= 1)
      if (row) row.push(item)
      else rows.push([item])
    }
    expect(geometry).toHaveLength(6)
    for (const row of rows) {
      for (const part of ['head', 'body', 'foot']) {
        const positions = row.map(item => item[part].top)
        expect(Math.max(...positions) - Math.min(...positions), `${width}px ${part} alignment`).toBeLessThanOrEqual(1)
      }
    }
    await expect(page.locator('.duty-person')).toHaveText(longPrimary)
    await expect(page.locator('.duty-backup')).toHaveText(`备班 ${longBackup}`)
    for (const selector of ['.duty-person', '.duty-shift', '.duty-backup']) {
      expect(await page.locator(selector).evaluate(element => {
        const r = element.getBoundingClientRect(), c = element.closest('.kpi').getBoundingClientRect()
        const body = element.closest('.kpi-body').getBoundingClientRect()
        return Math.abs(r.left - body.left) <= 1 && element.scrollWidth <= element.clientWidth + 1 && element.scrollHeight <= element.clientHeight + 1 && r.left >= c.left && r.right <= c.right && r.bottom <= c.bottom
      }), `${width}px ${selector} not clipped`).toBe(true)
    }
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(width)
  }
})

test('global search is usable from every business workspace with permission-safe results', async ({ page }) => {
  await setup(page)
  const queries = []
  await page.route('**/api/*?*', async route => {
    const url = new URL(route.request().url())
    if (!url.searchParams.has('keyword')) return route.fallback()
    queries.push(url.searchParams.get('keyword'))
    if (url.pathname === '/api/middleware') return route.fulfill({ status: 403, contentType: 'application/json', body: JSON.stringify({ error: 'forbidden' }) })
    return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ items: url.pathname === '/api/assets' ? [{ id: 1, assetNo: '<script>inert-search-result</script>' }] : [], total: url.pathname === '/api/assets' ? 1 : 0 }) })
  })
  for (const route of ['cmdb', 'middleware', 'oncall', 'tasks', 'incidents', 'permissions/users', 'audit', 'copilot-settings']) {
    await page.goto(`/#${route}`)
    const search = page.getByRole('textbox', { name: '全局搜索', exact: true })
    await expect(search).toBeVisible()
    await search.fill('   ')
    await search.press('Enter')
    await expect(page.getByRole('dialog', { name: '全局搜索 · 真实业务数据' })).not.toBeVisible()
    const count = queries.length
    await search.fill('搜索 & 业务')
    await search.press('Enter')
    const dialog = page.getByRole('dialog', { name: '全局搜索 · 真实业务数据' })
    await expect(dialog).toBeVisible()
    await expect(dialog).toContainText('无访问权限')
    await expect(dialog).toContainText('<script>inert-search-result</script>')
    await expect(dialog.locator('script')).toHaveCount(0)
    expect(queries.slice(count)).toEqual(Array(4).fill('搜索 & 业务'))
    const hash = new URL(page.url()).hash
    await page.keyboard.press('Escape')
    await expect(search).toBeFocused()
    expect(new URL(page.url()).hash).toBe(hash)
  }
  await page.setViewportSize({ width: 390, height: 844 })
  const trigger = page.getByRole('button', { name: '打开全局搜索', exact: true })
  await trigger.click()
  const dialog = page.getByRole('dialog', { name: '全局搜索 · 真实业务数据' })
  await dialog.getByRole('textbox', { name: '搜索关键词' }).fill('移动搜索')
  await dialog.getByRole('button', { name: '搜索', exact: true }).click()
  await expect(dialog).toContainText('无访问权限')
  await dialog.getByRole('button', { name: '<script>inert-search-result</script>', exact: true }).click()
  await expect(page).toHaveURL(/#cmdb$/)
  await expect(dialog).not.toBeVisible()
})

test('keeps the duty card body unframed in every interaction state and navigates on click', async ({ page }) => {
  await setup(page)
  const duty = page.locator('.duty-open')
  async function expectUnframed() {
    await expect(duty).toHaveCSS('border-width', '0px')
    await expect(duty).toHaveCSS('padding', '6px 0px 8px')
    await expect(duty).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)')
    await expect(duty).toHaveCSS('background-image', 'none')
    await expect(duty).toHaveCSS('box-shadow', 'none')
  }
  await expectUnframed()
  await duty.hover()
  await expectUnframed()
  await page.mouse.move(0, 0)
  await duty.focus()
  await expectUnframed()
  await page.keyboard.press('Tab')
  await page.keyboard.press('Shift+Tab')
  await expect(duty).toBeFocused()
  expect(await duty.evaluate(element => element.matches(':focus-visible'))).toBe(true)
  await expect(duty).toHaveCSS('outline-style', 'solid')
  expect(await duty.evaluate(element => parseFloat(getComputedStyle(element).outlineWidth))).toBeGreaterThan(0)
  await expectUnframed()
  await duty.locator('.duty-person').click()
  await expect(page).toHaveURL(/#oncall$/)
  await expect(page.locator('.workspace-page-header h1')).toHaveText('值班管理')
})

test('explicitly loads demo risks, links preset insight and keeps real events separate', async ({ page }) => {
  await setup(page)
  await expect(page.locator('.review-dashboard')).not.toContainText('虚构风险甲')
  await expect(page.locator('.kpi-duty')).toContainText('真实值班人')
  await expect(page.locator('.kpi-duty')).toContainText('备班 真实备班人')
  await expect(page.locator('.kpi-duty')).toContainText('09:15–19:45')
  await expect(page.locator('.kpi-duty')).toContainText('待交接 7 项')
  await page.getByLabel('选择业务系统').selectOption('全局业务系统')
  const firstPreview = page.waitForRequest(request => request.url().includes('/dashboard/preview?'))
  await page.getByLabel('开启独立演示').check()
  expect(new URL((await firstPreview).url()).searchParams.get('system')).toBe('all')
  await expect(page.getByLabel('选择业务系统').locator('option[value="全局业务系统"]')).toHaveCount(0)
  await expect(page.getByLabel('选择业务系统').locator('option')).toContainText(['全部业务系统', '支付交易平台 · 演示', '订单中心 · 演示', '企业门户 · 演示'])
  const systemRequest = page.waitForRequest(request => new URL(request.url()).searchParams.get('system') === 'payment')
  await page.getByLabel('选择业务系统').selectOption('payment')
  await systemRequest
  await expect(page.locator('.kpi').filter({ hasText: '自动化执行成功率' })).toContainText('成功 148')
  await expect(page.locator('.kpi').filter({ hasText: '核心服务可用性' })).toContainText('99.95%')
  await expect(page.locator('.demo')).toContainText('source: demo')
  await expect(page.locator('#trend')).toHaveAttribute('aria-label', /今日.*演示数据/)
  await expect(page.locator('#trend text').filter({ hasText: '00:00' })).toBeVisible()
  await page.getByRole('button', { name: /P2 1/ }).click()
  await expect(page.locator('.risk-row')).toHaveCount(1)
  await expect(page.locator('#aiPanel')).toContainText('虚构风险乙')
  await page.getByRole('button', { name: '查看详情', exact: true }).click()
  const dialog = page.locator('dialog[open]')
  await expect(dialog).toContainText('source: demo')
  await expect(dialog).toContainText('虚构建议乙')
  await page.keyboard.press('Escape')
  await expect(page.getByRole('button', { name: '查看详情', exact: true })).toBeFocused()
  await page.getByRole('button', { name: '审阅预案' }).click()
  await page.getByRole('button', { name: '确认演示意向' }).click()
  await expect(page.getByRole('status').filter({ hasText: /未审批、未执行/ })).toBeVisible()
  await expect(page.locator('.bottom table')).toContainText('真实事件不被演示替换')
  await expect(page.locator('.bottom table')).not.toContainText('虚构风险乙')
  const beforeHandover = page.url()
  await page.getByRole('button', { name: '查看交接' }).click()
  expect(page.url()).toBe(beforeHandover)
  await expect(page.locator('.review-dashboard .hero h1')).toBeVisible()
  await expect(page.locator('dialog[open]')).toContainText('真实交接记录')
  await expect(page.locator('dialog[open]')).toContainText('交班人 → 接班人 · 2026-10-14 09:15')
  await page.getByRole('button', { name: '取消', exact: true }).click()
  await expect(page.getByRole('button', { name: '查看交接' })).toBeFocused()
  expect(page.url()).toBe(beforeHandover)
  await page.getByLabel('开启独立演示').uncheck()
  await expect(page.getByLabel('选择业务系统')).toHaveValue('全局业务系统')
  await expect(page.locator('.risk-row')).toHaveCount(0)
  await expect(page.locator('.bottom table')).toContainText('真实事件不被演示替换')
})

test('custom date dialog validates range and restores focus on cancel or apply', async ({ page }) => {
  await setup(page)
  await page.getByLabel('开启独立演示').check()
  const custom = page.getByRole('button', { name: '自定义', exact: true })
  await custom.click()
  const dialog = page.locator('dialog[open]')
  await dialog.getByLabel('开始日期').fill('2026-10-14')
  await dialog.getByLabel('结束日期').fill('2026-10-01')
  await expect(dialog.getByRole('button', { name: '应用' })).toBeDisabled()
  await page.keyboard.press('Escape')
  await expect(custom).toBeFocused()
  await custom.click()
  await dialog.getByLabel('开始日期').fill('2026-10-01')
  await dialog.getByLabel('结束日期').fill('2026-10-14')
  const request = page.waitForRequest(request => request.url().includes('/dashboard/preview?period=custom'))
  await dialog.getByRole('button', { name: '应用' }).click()
  const customRequest = await request
  expect(customRequest.url()).toContain('from=2026-10-01')
  expect(customRequest.url()).toContain('to=2026-10-14')
  await expect(dialog).toBeHidden()
  await expect(custom).toBeFocused()
})
