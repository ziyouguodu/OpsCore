import { expect, test } from '@playwright/test'

const adminPassword = process.env.ADMIN_PASSWORD || 'OpsCore2026'

async function apiJSON(request, method, path, token, data) {
  const response = await request[method](`/api${path}`, {
    headers: { Authorization: `Bearer ${token}` },
    data
  })
  if (!response.ok()) {
    throw new Error(`${method.toUpperCase()} ${path} failed: ${response.status()} ${await response.text()}`)
  }
  if (response.status() === 204) return null
  return response.json()
}

async function ensureData(request, token) {
  const created = []
  const seed = Date.now().toString().slice(-6)

  const dutyOriginal = await apiJSON(request, 'get', '/duty-center', token)
  created.dutyOriginal = dutyOriginal
  const dutyUser = await apiJSON(request, 'post', '/users', token, {
    username: `duty-audit-${seed}`,
    displayName: '值班巡检用户',
    password: 'DutyAudit2026!',
    mustChangePassword: false,
    roles: ['ops_engineer']
  })
  created.push({ path: '/users', id: dutyUser.id })
  const adminUsers = await apiJSON(request, 'get', '/users', token)
  const admin = adminUsers.find(user => user.username === 'admin')
  const dutySeed = {
    teams: [{ name: '巡检值班组' }],
    members: [
      { id: `admin-${seed}`, userId: admin.id, username: admin.username, name: admin.displayName, team: '巡检值班组', role: 'SRE 负责人', count: 1, next: '待安排', status: '值班中' },
      { id: `ops-${seed}`, userId: dutyUser.id, username: dutyUser.username, name: dutyUser.displayName, team: '巡检值班组', role: '运维工程师', count: 0, next: '待安排', status: '空闲' }
    ],
    schedules: [{ id: `schedule-${seed}`, name: '巡检排班', team: '巡检值班组', rotation: 'daily', time: '08:00-20:00', members: [admin.displayName, dutyUser.displayName], active: true }],
    assignments: {},
    currentPeople: [{ id: `current-${seed}`, name: admin.displayName, role: '主值班', team: '巡检值班组', since: '08:00', until: '20:00', phone: '未配置', status: '在线' }],
    handovers: [],
    escalation: { name: 'P1 巡检升级策略', team: '巡检值班组', severity: 'P1', levels: [{ level: 1, target: '主值班', delay: '立即通知', channel: 'IM' }] }
  }
  const dutySaved = await apiJSON(request, 'put', '/duty-center', token, { revision: dutyOriginal.revision, data: dutySeed })
  created.dutySeedRevision = dutySaved.revision

  const asset = await apiJSON(request, 'post', '/assets', token, {
    type: '虚拟机',
    vendor: 'Audit',
    cpuArch: 'x86_64',
    sn: `AUDIT-SN-${seed}`,
    location: 'A1',
    business: '巡检业务',
    ipv4: `10.255.${seed.slice(0, 2)}.${seed.slice(2, 4)}`,
    ipv6: '',
    environment: '生产',
    os: 'Ubuntu',
    hostname: `audit-host-${seed}`,
    networkZone: 'audit-zone',
    cpu: '4C',
    memory: '8GB',
    disk: '100GB',
    deploymentInfo: 'audit-deploy',
    owner: 'UI Audit',
    status: '运行中',
    hostMachine: ''
  })
  created.push({ path: '/assets', id: asset.id, assetNo: asset.assetNo })

  const middleware = await apiJSON(request, 'get', '/middleware', token)
  if (!middleware.length) {
    const item = await apiJSON(request, 'post', '/middleware', token, {
      name: `audit-mysql-${seed}`,
      kind: 'MySQL',
      environment: '生产',
      networkZone: 'audit-db',
      endpoint: `10.255.${seed.slice(0, 2)}.${seed.slice(2, 4)}:3306`,
      business: '巡检业务',
      owner: 'UI Audit',
      status: '运行中'
    })
    created.push({ path: '/middleware', id: item.id })
  }

  const oncalls = await apiJSON(request, 'get', '/oncall', token)
  if (!oncalls.length) {
    const item = await apiJSON(request, 'post', '/oncall', token, {
      primary: 'UI Audit',
      backup: 'Smoke Ops',
      date: '2026-06-23',
      ruleType: 'daily',
      notes: '巡检覆盖'
    })
    created.push({ path: '/oncall', id: item.id })
  }

  const tasks = await apiJSON(request, 'get', '/tasks', token)
  if (!tasks.length) {
    const item = await apiJSON(request, 'post', '/tasks', token, {
      title: `UI 巡检任务 ${seed}`,
      assignee: 'UI Audit',
      status: '待处理',
      dueAt: '2026-06-24 18:00',
      description: '用于逐页真实点击巡检的数据'
    })
    created.push({ path: '/tasks', id: item.id })
  }

  const incidents = await apiJSON(request, 'get', '/incidents', token)
  if (!incidents.length) {
    const item = await apiJSON(request, 'post', '/incidents', token, {
      title: `UI 巡检事件 ${seed}`,
      level: 'P3',
      status: '新建',
      owner: 'UI Audit',
      business: '巡检业务',
      summary: '用于逐页真实点击巡检的数据'
    })
    created.push({ path: '/incidents', id: item.id })
  }

  return created
}

async function cleanupData(request, token, created) {
  if (created.dutyOriginal) {
    const current = await apiJSON(request, 'get', '/duty-center', token)
    await apiJSON(request, 'put', '/duty-center', token, { revision: current.revision, data: created.dutyOriginal.data })
  }
  for (const item of [...created].reverse()) {
    await request.delete(`/api${item.path}/${item.id}`, {
      headers: { Authorization: `Bearer ${token}` }
    }).catch(() => {})
  }
}

async function loginViaUI(page, username = 'admin', password = adminPassword) {
  await page.goto('/')
  await page.evaluate(() => { localStorage.clear(); sessionStorage.clear() })
  await page.reload()
  await expect(page.getByRole('heading', { name: '智能运维中枢指挥平台' })).toBeVisible()
  await page.getByLabel('账号').fill(username)
  await page.getByLabel('密码').fill(password)
  await page.getByRole('button', { name: '进入控制台' }).click()
  await expect(page.locator('.topbar h1')).toHaveText('首页健康总览')
}

test('keeps the collapsed dashboard readable on a mobile viewport', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await loginViaUI(page)

  const mainBox = await page.locator('.main').boundingBox()
  const firstKpiBox = await page.locator('.kpi-card').first().boundingBox()

  expect(mainBox?.width).toBeGreaterThan(340)
  expect(firstKpiBox?.width).toBeGreaterThan(300)
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390)

  const openNavigation = page.getByRole('button', { name: '打开导航' })
  await expect(openNavigation).toBeVisible()
  await openNavigation.click()
  await expect(page.locator('.sidebar')).toBeVisible()
  await page.locator('button.nav-child', { hasText: '任务跟踪' }).click()
  await expect(page.locator('.topbar h1')).toHaveText('任务跟踪')
  await expect(page.locator('.sidebar')).toBeHidden()
})

test('resets the page scroll position when switching workspaces', async ({ page }) => {
  await loginViaUI(page)
  await ensureSidebarExpanded(page)
  await goNav(page, '值班管理', '值班管理')
  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
  expect(await page.evaluate(() => window.scrollY)).toBeGreaterThan(0)

  await goNav(page, '任务跟踪', '任务跟踪')
  await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(0)
})

async function ensureSidebarExpanded(page) {
  const expand = page.getByLabel('展开菜单')
  if (await expand.isVisible().catch(() => false)) {
    await expand.click()
  }
}

async function goHome(page) {
  await page.locator('button.nav-row[title="首页仪表盘"]').click()
  await expect(page.locator('.topbar h1')).toHaveText('首页健康总览')
}

async function goNav(page, label, expectedTitle) {
  await ensureSidebarExpanded(page)
  await page.locator('button.nav-child', { hasText: label }).first().click()
  await expect(page.locator('.topbar h1')).toHaveText(expectedTitle)
}

async function openAndCancelEditor(page, buttonName, headingPattern = /新增|创建|新建|编辑/) {
  await page.getByRole('button', { name: buttonName }).first().click()
  const editor = page.locator('.editor-panel').first()
  await expect(editor).toBeVisible()
  await expect(editor.getByRole('heading', { name: headingPattern })).toBeVisible()
  await editor.getByRole('button', { name: '取消' }).click()
  await expect(editor).toBeHidden()
}

async function verifyListDetail(page, hiddenLabel) {
  const firstRow = page.locator('tbody tr.clickable-row').first()
  await expect(firstRow).toBeVisible()
  await firstRow.click()
  const hideButton = page.getByRole('button', { name: hiddenLabel })
  await expect(hideButton).toBeVisible()
  await hideButton.click()
  await expect(hideButton).toBeHidden()
}

async function verifyPager(page) {
  const pager = page.locator('.pager').first()
  await expect(pager).toBeVisible()
  await expect(pager.getByText(/每页显示/)).toBeVisible()
  await expect(pager.getByRole('button', { name: '上一页' })).toBeDisabled()
  if (await pager.getByRole('button', { name: '第 2 页' }).count() === 0) {
    await expect(pager.getByRole('button', { name: '下一页' })).toBeDisabled()
  }
  const pageSize = pager.locator('select')
  await pageSize.selectOption('20')
  await expect(pageSize).toHaveValue('20')
}

async function watchNativeDialogs(page) {
  let nativeDialogSeen = false
  page.on('dialog', async dialog => {
    nativeDialogSeen = true
    await dialog.dismiss().catch(() => {})
  })
  return () => nativeDialogSeen
}

async function openConfirmFromTrigger(page, trigger, expectedTitle) {
  await trigger.click()
  const dialog = page.getByTestId('confirm-dialog')
  await expect(dialog).toBeVisible()
  await expect(dialog.getByRole('heading', { name: expectedTitle })).toBeVisible()
  return dialog
}

async function openAndCancelDutyModal(page, buttonName, headingPattern) {
  await page.getByRole('button', { name: buttonName }).first().click()
  const modal = page.locator('.duty-modal').filter({ has: page.getByRole('heading', { name: headingPattern }) }).first()
  await expect(modal).toBeVisible()
  await modal.getByRole('button', { name: '取消' }).click()
  await expect(modal).toBeHidden()
}

test('clicks through OpsCore first-phase pages and core interactions', async ({ page, request }) => {
  const pageErrors = []
  page.on('pageerror', error => pageErrors.push(error.message))

  const loginResponse = await request.post('/api/auth/login', {
    data: { username: 'admin', password: adminPassword }
  })
  expect(loginResponse.ok()).toBeTruthy()
  const { token } = await loginResponse.json()
  const created = await ensureData(request, token)

  try {
    await loginViaUI(page)
    await ensureSidebarExpanded(page)
    await expect(page.locator('.topbar').getByTitle('打开 AI Copilot')).toBeVisible()

    const dashboardCards = [
      { label: '纳管资产', title: '资产台账（CMDB）' },
      { label: '今日值班', title: '值班管理' },
      { label: '进行中任务', title: '任务跟踪' },
      { label: '活跃事件', title: '事件管理' }
    ]
    for (const card of dashboardCards) {
      await goHome(page)
      await page.locator('.kpi-card', { hasText: card.label }).click()
      await expect(page.locator('.topbar h1')).toHaveText(card.title)
    }

    await goNav(page, '资产台账', '资产台账（CMDB）')
    await expect(page.getByRole('region', { name: '资产台账工作区' })).toBeVisible()
    await page.getByRole('button', { name: '高级搜索' }).click()
    await expect(page.getByRole('button', { name: '收起高级搜索' })).toBeVisible()
    await page.getByRole('button', { name: '收起高级搜索' }).click()
    await openAndCancelEditor(page, /新增资产/, /新增资产|编辑资产/)
    await verifyListDetail(page, '隐藏资产详情')
    const auditAsset = created.find(item => item.path === '/assets')
    const nativeDialogSeen = await watchNativeDialogs(page)
    const auditRow = page.locator('tbody tr.clickable-row', { hasText: auditAsset.assetNo }).first()
    await expect(auditRow).toBeVisible()
    const assetRowsBefore = await page.locator('tbody tr.clickable-row').count()

    await openConfirmFromTrigger(page, auditRow.getByRole('button', { name: '删除' }), '删除资产')
    await page.getByTestId('confirm-cancel').click()
    await expect(page.getByTestId('confirm-dialog')).toBeHidden()
    expect(nativeDialogSeen()).toBe(false)

    await openConfirmFromTrigger(page, auditRow.getByRole('button', { name: '删除' }), '删除资产')
    await page.keyboard.press('Escape')
    await expect(page.getByTestId('confirm-dialog')).toBeHidden()
    expect(nativeDialogSeen()).toBe(false)

    await openConfirmFromTrigger(page, auditRow.getByRole('button', { name: '删除' }), '删除资产')
    await page.getByTestId('confirm-accept').click()
    await expect(page.getByTestId('confirm-dialog')).toBeHidden()
    await expect.poll(async () => page.locator('tbody tr.clickable-row').count()).toBeLessThan(assetRowsBefore)
    expect(nativeDialogSeen()).toBe(false)
    await verifyPager(page)

    await goNav(page, '中间件与数据库', '中间件与数据库')
    await expect(page.getByRole('region', { name: '实例管理工作区' })).toBeVisible()
    await page.getByRole('button', { name: '高级搜索' }).click()
    await expect(page.getByRole('button', { name: '收起高级搜索' })).toBeVisible()
    await page.getByRole('button', { name: '收起高级搜索' }).click()
    await openAndCancelEditor(page, /新增实例/, /新增实例|编辑实例/)
    await verifyListDetail(page, '隐藏实例详情')
    await verifyPager(page)

    await goNav(page, '值班管理', '值班管理')
    await expect(page.getByRole('region', { name: '值班管理工作区' })).toBeVisible()
    await expect(page.getByRole('button', { name: '概览' })).toHaveClass(/active/)
    await page.getByRole('button', { name: '排班日历' }).click()
    await openAndCancelDutyModal(page, '手动分配值班', '分配值班')
    await page.getByRole('button', { name: '手动分配值班' }).click()
    const assignmentModal = page.locator('.duty-modal').filter({ has: page.getByRole('heading', { name: '分配值班' }) }).first()
    await assignmentModal.getByRole('button', { name: '确认分配' }).click()
    await expect(assignmentModal).toBeHidden()
    await expect.poll(async () => (await apiJSON(request, 'get', '/duty-center', token)).revision).toBeGreaterThan(created.dutySeedRevision)
    await page.reload()
    await expect(page.locator('.topbar h1')).toHaveText('值班管理')
    await expect(page.getByText(/数据版本/)).toBeVisible()
    await page.getByRole('button', { name: '排班日历' }).click()
    await page.getByRole('button', { name: '排班配置' }).click()
    await openAndCancelDutyModal(page, '新建排班', /新建排班模板|编辑排班模板/)
    await page.getByRole('button', { name: '值班列表' }).click()
    await openAndCancelDutyModal(page, '添加人员', '添加值班人员')
    await openAndCancelDutyModal(page, '团队配置', '团队配置')
    await goHome(page)
    await goNav(page, '值班管理', '值班管理')
    await expect(page.getByRole('button', { name: '添加人员' })).toBeVisible()
    await page.getByRole('button', { name: '交接班日志' }).click()
    await openAndCancelDutyModal(page, '提交交接', '提交交接班')
    await page.getByRole('button', { name: '升级策略' }).click()
    await openAndCancelDutyModal(page, '编辑策略', '编辑升级策略')

    await goNav(page, '任务跟踪', '任务跟踪')
    await expect(page.getByRole('region', { name: '任务跟踪工作区' })).toBeVisible()
    await openAndCancelEditor(page, /创建任务/, /创建任务|编辑任务/)
    await verifyListDetail(page, '隐藏任务详情')
    await verifyPager(page)

    await goNav(page, '事件管理', '事件管理')
    await expect(page.getByRole('region', { name: '事件管理工作区' })).toBeVisible()
    await openAndCancelEditor(page, /新建事件/, /新建事件|编辑事件/)
    await verifyListDetail(page, '隐藏事件详情')
    await verifyPager(page)

    await goNav(page, '用户与角色', '权限管理')
    await expect(page.getByRole('region', { name: '权限管理工作区' })).toBeVisible()
    await openAndCancelEditor(page, /新增用户/, /新增用户|编辑用户/)
    await page.locator('.segmented-tabs').getByRole('button', { name: '菜单与资源权限' }).click()
    await expect(page.getByRole('heading', { name: '菜单授权概览' })).toBeVisible()
    await expect(page.getByRole('heading', { name: '资源权限矩阵' })).toBeVisible()

    await goNav(page, 'AI Copilot 配置', 'AI Copilot 配置')
    await expect(page.getByRole('region', { name: 'AI Copilot 配置工作区' })).toBeVisible()
    await page.getByRole('button', { name: /本地模型/ }).click()
    await expect(page.getByLabel('本地模型地址')).toBeVisible()
    await page.getByRole('button', { name: '测试连接' }).click()
    await expect(page.locator('.top-actions .error, .connection-result').first()).toContainText(/AI Copilot|连接/)

    await page.getByTitle('打开 AI Copilot').click()
    await expect(page.locator('.copilot')).toBeVisible()
    await page.getByTitle('隐藏 Copilot').click()
    await expect(page.getByTitle('打开 AI Copilot')).toBeVisible()

    expect(pageErrors).toEqual([])
  } finally {
    await cleanupData(request, token, created)
  }
})

test('enforces ops engineer UI permissions for users and Copilot settings', async ({ page, request }) => {
  const seed = Date.now().toString().slice(-6)
  const username = `ui-ops-${seed}`
  const password = 'AuditOps2026!'
  const adminLogin = await request.post('/api/auth/login', {
    data: { username: 'admin', password: adminPassword }
  })
  expect(adminLogin.ok()).toBeTruthy()
  const { token } = await adminLogin.json()
  const user = await apiJSON(request, 'post', '/users', token, {
    username,
    displayName: 'UI 权限巡检',
    password,
    mustChangePassword: false,
    roles: ['ops_engineer']
  })

  try {
    await loginViaUI(page, username, password)
    await goNav(page, 'AI Copilot 配置', 'AI Copilot 配置')
    const copilotRegion = page.getByRole('region', { name: 'AI Copilot 配置工作区' })
    await expect(copilotRegion).toBeVisible()
    await expect(copilotRegion.getByRole('button', { name: '测试连接' })).toBeDisabled()
    await expect(copilotRegion.getByRole('button', { name: '保存配置' })).toBeDisabled()
    await expect(copilotRegion.getByLabel('API Endpoint')).toBeDisabled()
    await expect(copilotRegion.locator('.provider-card')).toHaveCount(5)
    for (const provider of await copilotRegion.locator('.provider-card').all()) {
      await expect(provider).toBeDisabled()
    }

    await goNav(page, '用户与角色', '权限管理')
    const permissionsRegion = page.getByRole('region', { name: '权限管理工作区' })
    await expect(permissionsRegion).toBeVisible()
    await expect(permissionsRegion.getByRole('button', { name: /新增用户/ })).toHaveCount(0)
    await expect(permissionsRegion.getByRole('heading', { name: '当前权限' })).toBeVisible()
  } finally {
    await request.delete(`/api/users/${user.id}`, {
      headers: { Authorization: `Bearer ${token}` }
    })
  }
})
