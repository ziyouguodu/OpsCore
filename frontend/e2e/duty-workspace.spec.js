import { expect, test } from '@playwright/test'

test.use({ baseURL: 'http://localhost:5173', viewport: { width: 1280, height: 900 }, timezoneId: 'Asia/Shanghai' })

const today = '2026-10-16'
const future = '2026-10-18'
const team = '平台运维组'
const users = [
  { id: 1, username: 'duty-primary', displayName: '主班甲', phone: '13800138000', roles: ['super_admin'], mustChangePassword: false },
  { id: 2, username: 'duty-backup', displayName: '备班乙', phone: '', roles: ['ops_engineer'], mustChangePassword: false },
  { id: 3, username: 'duty-incoming', displayName: '接班丙', phone: '+86 139-0013-9000', roles: ['ops_engineer'], mustChangePassword: false }
]
const sections = [
  ['overview', '概览'], ['calendar', '排班日历'], ['schedules', '排班配置'],
  ['roster', '值班列表'], ['handover', '交接班日志'], ['escalation', '升级策略']
]

function createDutyState() {
  return {
    revision: 7,
    data: {
      teams: [{ name: team }],
      members: users.map((user, index) => ({ id: String(user.id), userId: user.id, username: user.username, name: user.displayName, phone: user.phone, team, role: '运维工程师', count: index + 1, next: today, status: index === 0 ? '值班中' : '空闲' })),
      currentPeople: [{ id: '1', name: users[0].displayName, role: '主值班', team, since: '08:00', until: '20:00', phone: users[0].phone, status: '在线' }],
      assignments: { [today]: { primary: users[0].displayName, backup: users[1].displayName } },
      schedules: [{ id: 'schedule-1', name: '平台周轮换', team, rotation: 'weekly', time: '08:00-20:00', members: users.map(user => user.displayName), active: true }],
      handovers: [{ id: 'handover-1', from: users[0].displayName, to: users[2].displayName, time: `${today} 08:00`, content: '关注待确认交接事项', complete: false }],
      escalation: { name: '平台 P1 升级策略', team, severity: 'P1', levels: [{ level: 1, target: '主值班', delay: '立即通知', channel: 'IM' }] }
    }
  }
}

function workspaceFrom(state) {
  const data = state.data
  const assignment = data.assignments[today] || { primary: '', backup: '' }
  const personFor = (type) => {
    const current = data.currentPeople.find(person => type === 'primary' ? ['主值班', 'primary'].includes(person.role) : person.role?.startsWith('备') || person.role === 'backup')
    if (current) return current
    const member = data.members.find(person => person.name === assignment[type])
    return member ? { id: member.id, name: member.name, phone: member.phone, team: member.team, role: type === 'primary' ? '主值班' : '备份值班', status: '已排班' } : null
  }
  const pendingHandovers = data.handovers.filter(item => !item.complete)
  return {
    source: 'real', generatedAt: `${today}T12:00:00+08:00`,
    statistics: { assetCount: 0, activeIncidentCount: 0, activeTaskCount: 0, assetTypeCounts: {}, incidentLevelCounts: {} },
    businessSystems: [], incidents: [], tasks: [],
    duty: { source: 'real', revision: state.revision, date: today, primary: personFor('primary'), backup: personFor('backup'), currentPeople: data.currentPeople, todayAssignment: assignment, nextAssignment: null, pendingCount: pendingHandovers.length, pendingHandovers, handovers: data.handovers },
    trend: { source: 'real', status: 'unavailable', points: [] }
  }
}

async function setup(page, { role = 'super_admin', view = 'oncall' } = {}) {
  const server = { state: createDutyState(), writes: [], failNextPut: false, nextGet: null, unexpected: [] }
  await page.clock.setFixedTime(new Date(`${today}T12:00:00+08:00`))
  await page.addInitScript(() => sessionStorage.setItem('opscore.token', 'duty-workspace-test-token'))
  await page.route('**/api/**', async route => {
    const request = route.request()
    const url = new URL(request.url())
    let status = 200
    let body
    if (url.pathname === '/api/duty-center' && request.method() === 'PUT') {
      const payload = request.postDataJSON()
      server.writes.push(payload)
      if (role !== 'super_admin') {
        status = 403
        body = { error: '当前角色仅可查看排班' }
      } else if (server.failNextPut) {
        server.failNextPut = false
        status = 500
        body = { error: '专项测试保存失败，请重试' }
      } else if (payload.revision !== server.state.revision) {
        status = 409
        body = { error: '值班数据版本冲突' }
      } else {
        server.state = { revision: server.state.revision + 1, data: structuredClone(payload.data) }
        for (const member of server.state.data.members) member.phone = users.find(user => user.id === member.userId)?.phone || ''
        for (const person of server.state.data.currentPeople) person.phone = server.state.data.members.find(member => member.name === person.name)?.phone || ''
        body = server.state
      }
    } else if (request.method() !== 'GET') {
      server.unexpected.push(`${request.method()} ${url.pathname}`)
      status = 405
      body = { error: '未配置的写请求' }
    } else if (url.pathname === '/api/auth/me') body = { ...users[0], roles: [role] }
    else if (url.pathname === '/api/dashboard/workspace') body = workspaceFrom(server.state)
    else if (url.pathname === '/api/duty-center') {
      const pending = server.nextGet
      server.nextGet = null
      if (pending) await pending
      body = server.state
    }
    else if (['/api/users', '/api/user-directory'].includes(url.pathname)) body = users
    else if (['/api/oncall', '/api/copilot/configs', '/api/audit-events'].includes(url.pathname)) body = []
    else if (url.pathname === '/api/security/credential-verification') body = { hasPassword: true }
    else if (/^\/api\/(assets|middleware|tasks|incidents)$/.test(url.pathname)) body = { items: [], total: 0, page: 1, pageSize: 10, pageCount: 1, options: { business: [], networkZone: [] }, counts: {} }
    else {
      server.unexpected.push(`${request.method()} ${url.pathname}`)
      status = 404
      body = { error: '未配置的读取请求' }
    }
    await route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) })
  })
  await page.goto(`/#${view}`)
  if (view === 'oncall') {
    await expect(page.getByRole('region', { name: '值班管理工作区' })).toBeVisible()
    await expect(page.locator('.duty-status-strip')).toContainText('数据版本 7')
  } else await expect(page.locator('.review-dashboard')).toBeVisible()
  return server
}

async function selectSection(page, label) {
  await page.getByRole('tab', { name: label, exact: true }).click()
  await expect(page.getByRole('tabpanel', { name: label, exact: true })).toBeVisible()
}

async function expectCompactLayout(page, width) {
  expect(await page.evaluate(() => Math.max(document.documentElement.scrollWidth, document.body.scrollWidth))).toBeLessThanOrEqual(width)
  const bounds = await page.getByRole('tablist', { name: '值班工作区' }).boundingBox()
  expect(bounds.height).toBeGreaterThan(0)
  expect(bounds.height).toBeLessThan(70)
}

test('首页主备岗位来源清晰，点击当前值班进入工作区且侧栏没有任务入口', async ({ page }) => {
  const server = await setup(page, { view: 'dashboard' })
  const duty = page.locator('.duty-open')
  await expect(duty.locator('.duty-person')).toHaveText(users[0].displayName)
  await expect(duty.locator('.duty-shift')).toHaveText('主值班 · 08:00–20:00')
  await expect(duty.locator('.duty-backup')).toHaveText(`备班 ${users[1].displayName}`)
  await expect(page.locator('.kpi-duty')).toContainText('待交接 1 项')
  const sidebar = page.getByRole('complementary', { name: '主导航' })
  await expect(sidebar.getByRole('button', { name: '任务跟踪', exact: true })).toHaveCount(0)
  await expect(sidebar.getByRole('button', { name: /任务与自动化/ })).toBeDisabled()
  await duty.click()
  await expect(page).toHaveURL(/#oncall$/)
  await expect(page.getByRole('region', { name: '值班管理工作区' })).toBeVisible()
  await expect(page.locator('.duty-role-card.primary')).toContainText(users[0].displayName)
  await expect(sidebar.getByRole('button', { name: '值班管理', exact: true })).toHaveAttribute('aria-current', 'page')
  expect(server.writes).toHaveLength(0)
  expect(server.unexpected).toEqual([])
})

test('六个工作区 Tab 支持 ARIA、方向键及首尾切换，桌面和移动布局紧凑无页面溢出', async ({ page }) => {
  const server = await setup(page)
  for (const width of [1280, 390]) {
    await page.setViewportSize({ width, height: width === 390 ? 844 : 900 })
    await expect(page.getByRole('tablist', { name: '值班工作区' }).getByRole('tab')).toHaveCount(6)
    for (const [id, label] of sections) {
      await selectSection(page, label)
      const tab = page.getByRole('tab', { name: label, exact: true })
      await expect(tab).toHaveAttribute('aria-selected', 'true')
      await expect(tab).toHaveAttribute('tabindex', '0')
      await expect(tab).toHaveAttribute('aria-controls', `duty-panel-${id}`)
      await expect(page.locator(`#duty-panel-${id}`)).toHaveAttribute('aria-labelledby', `duty-tab-${id}`)
      await expect(page.locator('.duty-tabbar [aria-selected="false"][tabindex="-1"]')).toHaveCount(5)
      await expect(page.getByRole('tabpanel')).toHaveCount(1)
      await expectCompactLayout(page, width)
    }
    await page.getByRole('tab', { name: '升级策略', exact: true }).focus()
    for (const [key, label] of [['Home', '概览'], ['ArrowRight', '排班日历'], ['End', '升级策略'], ['ArrowRight', '概览'], ['ArrowLeft', '升级策略'], ['Home', '概览']]) {
      await page.keyboard.press(key)
      const tab = page.getByRole('tab', { name: label, exact: true })
      await expect(tab).toBeFocused()
      await expect(tab).toHaveAttribute('aria-selected', 'true')
      await expect(page.getByRole('tabpanel', { name: label, exact: true })).toBeVisible()
      await expectCompactLayout(page, width)
    }
  }
  expect(server.writes).toHaveLength(0)
  expect(server.unexpected).toEqual([])
})

test('立即接班使用原生模态框，取消和 Escape 不写入，确认第三人仅保存一次并同步岗位', async ({ page }) => {
  const server = await setup(page)
  const initial = structuredClone(server.state)
  const trigger = page.getByRole('button', { name: '立即接班', exact: true })
  const dialog = page.locator('dialog.duty-dialog[open]')
  for (const action of ['cancel', 'escape']) {
    await trigger.click()
    await expect(dialog).toBeVisible()
    expect(await dialog.evaluate(element => element.matches(':modal'))).toBe(true)
    await expect(dialog.getByRole('dialog', { name: '确认接班' })).toHaveAttribute('aria-modal', 'true')
    await dialog.getByLabel('接班人员').selectOption(users[2].displayName)
    if (action === 'cancel') await dialog.getByRole('button', { name: '取消', exact: true }).click()
    else await page.keyboard.press('Escape')
    await expect(dialog).toHaveCount(0)
    await expect(trigger).toBeFocused()
    expect(server.writes).toHaveLength(0)
    expect(server.state).toEqual(initial)
  }
  await trigger.click()
  await dialog.getByLabel('接班岗位').selectOption('primary')
  await dialog.getByLabel('接班人员').selectOption(users[2].displayName)
  await dialog.getByRole('button', { name: '确认接班', exact: true }).click()
  await expect(dialog).toHaveCount(0)
  await expect(page.locator('.duty-status-strip')).toContainText('数据版本 8')
  expect(server.writes).toHaveLength(1)
  expect(server.writes[0].revision).toBe(7)
  expect(server.state.data.currentPeople).toHaveLength(1)
  expect(server.state.data.currentPeople[0]).toMatchObject({ id: '3', name: users[2].displayName, role: '主值班', team, until: '20:00', status: '在线' })
  expect(server.state.data.currentPeople[0].since).toMatch(/^\d{2}:\d{2}$/)
  expect(server.state.data.assignments[today]).toEqual({ primary: users[2].displayName, backup: users[1].displayName })
  await expect(page.locator('.duty-role-card.primary')).toContainText(users[2].displayName)
  await expect(page.locator('.duty-role-card.primary')).toContainText(users[2].phone)
  await expect(page.locator('.duty-role-card.primary').getByRole('link')).toHaveAttribute('href', 'tel:+8613900139000')
  await selectSection(page, '值班列表')
  await expect(page.getByRole('row').filter({ hasText: users[2].displayName })).toContainText('值班中')
  await expect(page.getByRole('row').filter({ hasText: users[0].displayName })).toContainText('空闲')
  await page.getByRole('button', { name: '刷新', exact: true }).click()
  await expect(page.locator('.duty-status-strip')).toContainText('数据版本 8')
  await selectSection(page, '概览')
  await expect(page.locator('.duty-role-card.primary')).toContainText(users[2].displayName)
  await expect(page.locator('.duty-role-card.primary')).toContainText(users[2].phone)
  await expect(page.locator('.duty-role-card.primary').getByRole('link')).toHaveAttribute('href', 'tel:+8613900139000')
  expect(server.writes).toHaveLength(1)
  expect(server.unexpected).toEqual([])
})

test('未来手动分配不影响当前岗位，保存失败保留输入并允许原版本重试成功', async ({ page }) => {
  const server = await setup(page)
  const initial = structuredClone(server.state)
  await selectSection(page, '排班日历')
  await page.getByRole('button', { name: '手动分配值班', exact: true }).click()
  const dialog = page.locator('dialog.duty-dialog[open]')
  await dialog.getByLabel('值班日期').fill(future)
  await dialog.getByLabel('备份值班', { exact: true }).check()
  await dialog.getByLabel('值班人员').selectOption(users[2].displayName)
  server.failNextPut = true
  await dialog.getByRole('button', { name: '确认分配', exact: true }).click()
  await expect(dialog.getByRole('alert')).toContainText('专项测试保存失败，请重试')
  await expect(dialog).toBeVisible()
  await expect(dialog.getByLabel('值班日期')).toHaveValue(future)
  await expect(dialog.getByLabel('备份值班', { exact: true })).toBeChecked()
  await expect(dialog.getByLabel('值班人员')).toHaveValue(users[2].displayName)
  expect(server.writes).toHaveLength(1)
  expect(server.state).toEqual(initial)
  expect(server.writes[0].data.currentPeople).toEqual(initial.data.currentPeople)
  await dialog.getByRole('button', { name: '确认分配', exact: true }).click()
  await expect(dialog).toHaveCount(0)
  await expect(page.locator('.duty-status-strip')).toContainText('数据版本 8')
  expect(server.writes).toHaveLength(2)
  expect(server.writes.map(write => write.revision)).toEqual([7, 7])
  expect(server.state.data.currentPeople).toEqual(initial.data.currentPeople)
  expect(server.state.data.assignments[today]).toEqual(initial.data.assignments[today])
  expect(server.state.data.assignments[future]).toEqual({ primary: '', backup: users[2].displayName })
  expect(server.state.data.members.find(member => member.id === '3')).toMatchObject({ next: future, count: 4, status: '空闲' })
  await expect(page.locator('.duty-day-detail')).toContainText(future)
  await expect(page.locator('.duty-day-detail')).toContainText(users[2].displayName)
  await expect(page.locator('.duty-center').getByRole('alert')).toHaveCount(0)
  await selectSection(page, '概览')
  await expect(page.locator('.duty-role-card.primary')).toContainText(users[0].displayName)
  await expect(page.locator('.duty-role-card.primary')).not.toContainText(users[2].displayName)
  expect(server.unexpected).toEqual([])
})

test('完成交接拒绝占用其他岗位的接班人，交给第三人时保留备班并完成日志', async ({ page }) => {
  const server = await setup(page)
  const backup = { id: '2', name: users[1].displayName, role: '备份值班', team, since: '08:00', until: '20:00', phone: '', status: '在线' }
  server.state.data.currentPeople.push(backup)
  await page.getByRole('button', { name: '刷新', exact: true }).click()
  await expect(page.locator('.duty-role-card.backup')).toContainText(users[1].displayName)
  const initial = structuredClone(server.state)
  await page.getByRole('button', { name: '提交交接', exact: true }).click()
  const dialog = page.locator('dialog.duty-dialog[open]')
  await dialog.getByRole('combobox', { name: '交接人', exact: true }).selectOption(users[0].displayName)
  await dialog.getByRole('combobox', { name: '接班人', exact: true }).selectOption(users[1].displayName)
  await dialog.getByLabel('交接内容').fill('本班检查完成，保留备班岗位')
  await dialog.getByLabel('交接已完成，系统运行正常').check()
  await dialog.getByRole('button', { name: '提交交接', exact: true }).click()
  await expect(page.locator('.duty-toast').filter({ hasText: '接班人已在当前其他岗位' })).toBeVisible()
  await expect(dialog).toBeVisible()
  expect(server.writes).toHaveLength(0)
  expect(server.state).toEqual(initial)
  await dialog.getByRole('combobox', { name: '接班人', exact: true }).selectOption(users[2].displayName)
  await dialog.getByRole('button', { name: '提交交接', exact: true }).click()
  await expect(dialog).toHaveCount(0)
  await expect(page.locator('.duty-status-strip')).toContainText('数据版本 8')
  expect(server.writes).toHaveLength(1)
  expect(server.state.data.currentPeople).toHaveLength(2)
  expect(server.state.data.currentPeople.find(person => person.role === '主值班')).toMatchObject({ id: '3', name: users[2].displayName })
  expect(server.state.data.currentPeople.find(person => person.id === '2')).toEqual(backup)
  expect(server.state.data.assignments[today]).toEqual({ primary: users[2].displayName, backup: users[1].displayName })
  expect(server.state.data.handovers[0]).toMatchObject({ from: users[0].displayName, to: users[2].displayName, content: '本班检查完成，保留备班岗位', complete: true })
  await expect(page.locator('.duty-role-card.primary')).toContainText(users[2].displayName)
  await expect(page.locator('.duty-role-card.primary')).toContainText(users[2].phone)
  await expect(page.locator('.duty-role-card.primary').getByRole('link')).toHaveAttribute('href', 'tel:+8613900139000')
  await expect(page.locator('.duty-role-card.backup')).toContainText(users[1].displayName)
  await selectSection(page, '交接班日志')
  const log = page.locator('.duty-handover-list article').filter({ hasText: '本班检查完成，保留备班岗位' })
  await expect(log).toContainText(`${users[0].displayName} → ${users[2].displayName}`)
  await expect(log).toContainText('交接完成')
  expect(server.unexpected).toEqual([])
})

test('升级策略 409 保留草稿，Escape 恢复远端策略且其他保存不带回旧草稿', async ({ page }) => {
  const server = await setup(page)
  await selectSection(page, '升级策略')
  await page.getByRole('button', { name: '编辑策略', exact: true }).click()
  const dialog = page.locator('dialog.duty-dialog[open]')
  await dialog.getByLabel('策略名称').fill('本地未提交策略')
  await dialog.getByLabel('升级时间').fill('5 分钟后')
  await dialog.getByLabel('通知渠道').fill('短信')
  server.state.revision++
  server.state.data.escalation.name = '远端新策略'
  const remote = structuredClone(server.state.data.escalation)
  const conflict = page.waitForResponse(response => new URL(response.url()).pathname === '/api/duty-center' && response.request().method() === 'PUT' && response.status() === 409)
  await dialog.getByRole('button', { name: '保存策略', exact: true }).click()
  await conflict
  await expect(dialog.getByRole('alert')).toContainText('已刷新最新版本')
  await expect(page.locator('.duty-status-strip')).toContainText('数据版本 8')
  await expect(dialog.getByLabel('策略名称')).toHaveValue('本地未提交策略')
  await expect(dialog.getByLabel('升级时间')).toHaveValue('5 分钟后')
  await expect(dialog.getByLabel('通知渠道')).toHaveValue('短信')
  await expect(dialog.getByRole('button', { name: '保存策略', exact: true })).toBeEnabled()
  expect(server.writes).toHaveLength(1)
  expect(server.writes[0].revision).toBe(7)
  expect(server.writes[0].data.escalation.name).toBe('本地未提交策略')
  expect(server.state.data.escalation).toEqual(remote)
  await page.keyboard.press('Escape')
  await expect(dialog).toHaveCount(0)
  await expect(page.getByRole('button', { name: '编辑策略', exact: true })).toBeFocused()
  await expect(page.locator('.duty-policy-meta')).toContainText('远端新策略')
  await expect(page.locator('.duty-policy-meta')).not.toContainText('本地未提交策略')
  await selectSection(page, '排班配置')
  await page.getByRole('button', { name: '停用', exact: true }).click()
  await expect(page.getByRole('button', { name: '启用', exact: true })).toBeVisible()
  await expect(page.locator('.duty-status-strip')).toContainText('数据版本 9')
  expect(server.writes).toHaveLength(2)
  expect(server.writes[1].revision).toBe(8)
  expect(server.writes[1].data.escalation).toEqual(remote)
  expect(server.state.data.escalation).toEqual(remote)
  expect(server.state.data.schedules[0].active).toBe(false)
  expect(server.unexpected).toEqual([])
})

test('刷新 GET 等待期间禁用各工作区写按钮，完成刷新后才能重新操作', async ({ page }) => {
  const server = await setup(page)
  let release
  server.nextGet = new Promise(resolve => { release = resolve })
  server.state.revision++
  const refresh = page.getByRole('button', { name: '刷新', exact: true })
  const loading = page.getByRole('status').filter({ hasText: '正在加载值班数据' })
  const started = page.waitForRequest(request => new URL(request.url()).pathname === '/api/duty-center' && request.method() === 'GET')
  try {
    await refresh.click()
    await started
    await expect(loading).toBeVisible()
    await expect(refresh).toBeDisabled()
    await expect(page.getByRole('button', { name: '团队配置', exact: true })).toBeDisabled()
    for (const [id, label] of sections) {
      await selectSection(page, label)
      const buttons = page.locator(`#duty-panel-${id}`).getByRole('button').filter({ hasText: /^(提交交接|立即接班|调整班次|安排值班|手动分配值班|调整主值班|调整备班|新建排班|编辑|停用|删除|添加人员|移除|确认交接|编辑策略)$/ })
      expect(await buttons.count()).toBeGreaterThan(0)
      for (const button of await buttons.all()) await expect(button).toBeDisabled()
    }
    await expect(page.locator('dialog.duty-dialog[open]')).toHaveCount(0)
    expect(server.writes).toHaveLength(0)
  } finally {
    release()
  }
  await expect(loading).toHaveCount(0)
  await expect(refresh).toBeEnabled()
  await expect(page.locator('.duty-status-strip')).toContainText('数据版本 8')
  await expect(page.getByRole('button', { name: '团队配置', exact: true })).toBeEnabled()
  await selectSection(page, '概览')
  const takeover = page.getByRole('button', { name: '立即接班', exact: true })
  await expect(takeover).toBeEnabled()
  await takeover.click()
  const dialog = page.locator('dialog.duty-dialog[open]')
  await expect(dialog).toBeVisible()
  await dialog.getByRole('button', { name: '取消', exact: true }).click()
  await expect(dialog).toHaveCount(0)
  expect(server.writes).toHaveLength(0)
  expect(server.unexpected).toEqual([])
})

test('运维工程师所有写按钮禁用但仍能切换日历并查看日期安排', async ({ page }) => {
  const server = await setup(page, { role: 'ops_engineer' })
  await expect(page.locator('.duty-status-strip')).toContainText('只读')
  await expect(page.getByRole('button', { name: '团队配置', exact: true })).toBeDisabled()
  for (const [id, label] of sections) {
    await selectSection(page, label)
    const panel = page.locator(`#duty-panel-${id}`)
    const buttons = panel.getByRole('button').filter({ hasText: /^(提交交接|立即接班|调整班次|安排值班|手动分配值班|调整主值班|调整备班|新建排班|编辑|停用|删除|添加人员|移除|确认交接|编辑策略)$/ })
    expect(await buttons.count()).toBeGreaterThan(0)
    for (const button of await buttons.all()) await expect(button).toBeDisabled()
  }
  await selectSection(page, '排班日历')
  const day = page.getByRole('button', { name: `${today} 主值班 ${users[0].displayName} 备班 ${users[1].displayName}`, exact: true })
  await expect(day).toBeEnabled()
  await day.click()
  await expect(day).toHaveAttribute('aria-pressed', 'true')
  await expect(page.locator('.duty-day-detail')).toContainText(users[0].displayName)
  await expect(page.locator('.duty-day-detail')).toContainText(users[1].displayName)
  const futureDay = page.getByRole('button', { name: `${future} 主值班 未安排 备班 未安排`, exact: true })
  await futureDay.click()
  await expect(futureDay).toHaveAttribute('aria-pressed', 'true')
  await expect(day).toHaveAttribute('aria-pressed', 'false')
  await expect(page.locator('.duty-day-detail')).toContainText(future)
  await expect(page.locator('.duty-day-detail')).toContainText('当前角色仅可查看排班')
  await expect(page.getByRole('button', { name: '下一月', exact: true })).toBeEnabled()
  await page.getByRole('button', { name: '下一月', exact: true }).click()
  await expect(page.locator('.duty-calendar-tools')).toContainText('2026年11月')
  await page.getByRole('button', { name: '重置视图', exact: true }).click()
  await expect(page.locator('.duty-calendar-tools')).toContainText('2026年10月')
  await expect(page.locator('dialog.duty-dialog[open]')).toHaveCount(0)
  expect(server.writes).toHaveLength(0)
  expect(server.unexpected).toEqual([])
})
