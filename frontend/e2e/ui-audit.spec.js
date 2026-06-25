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

  const assets = await apiJSON(request, 'get', '/assets', token)
  if (!assets.length) {
    const asset = await apiJSON(request, 'post', '/assets', token, {
      type: '虚拟机',
      vendor: 'Audit',
      cpuArch: 'x86_64',
      sn: `AUDIT-SN-${seed}`,
      physicalLocation: 'A1',
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
      connectedStatus: '已并网',
      hostMachine: ''
    })
    created.push({ path: '/assets', id: asset.id })
  }

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
      scope: '巡检覆盖'
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
  for (const item of [...created].reverse()) {
    await request.delete(`/api${item.path}/${item.id}`, {
      headers: { Authorization: `Bearer ${token}` }
    }).catch(() => {})
  }
}

async function loginViaUI(page) {
  await page.goto('/')
  await page.evaluate(() => localStorage.clear())
  await page.reload()
  await expect(page.getByRole('heading', { name: '智能运维中枢指挥平台' })).toBeVisible()
  await page.getByLabel('账号').fill('admin')
  await page.getByLabel('密码').fill(adminPassword)
  await page.getByRole('button', { name: '进入控制台' }).click()
  await expect(page.locator('.topbar h1')).toHaveText('首页健康总览')
}

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
    await page.getByRole('button', { name: '高级搜索' }).click()
    await expect(page.getByRole('button', { name: '收起高级搜索' })).toBeVisible()
    await page.getByRole('button', { name: '收起高级搜索' }).click()
    await openAndCancelEditor(page, /新增资产/, /新增资产|编辑资产/)
    await verifyListDetail(page, '隐藏资产详情')
    await verifyPager(page)

    await goNav(page, '中间件与数据库', '中间件与数据库')
    await page.getByRole('button', { name: '高级搜索' }).click()
    await expect(page.getByRole('button', { name: '收起高级搜索' })).toBeVisible()
    await page.getByRole('button', { name: '收起高级搜索' }).click()
    await openAndCancelEditor(page, /新增实例/, /新增实例|编辑实例/)
    await verifyListDetail(page, '隐藏实例详情')
    await verifyPager(page)

    await goNav(page, '值班管理', '值班管理')
    await expect(page.getByRole('button', { name: '概览' })).toHaveClass(/active/)
    await page.getByRole('button', { name: '排班日历' }).click()
    await openAndCancelDutyModal(page, '手动分配值班', '分配值班')
    await page.getByRole('button', { name: '排班配置' }).click()
    await openAndCancelDutyModal(page, '新建排班', /新建排班模板|编辑排班模板/)
    await page.getByRole('button', { name: '值班列表' }).click()
    await openAndCancelDutyModal(page, '添加人员', '添加值班人员')
    await openAndCancelDutyModal(page, '团队配置', '团队配置')
    await page.getByRole('button', { name: '交接班日志' }).click()
    await openAndCancelDutyModal(page, '提交交接', '提交交接班')
    await page.getByRole('button', { name: '升级策略' }).click()
    await openAndCancelDutyModal(page, '编辑策略', '编辑升级策略')

    await goNav(page, '任务跟踪', '任务跟踪')
    await openAndCancelEditor(page, /创建任务/, /创建任务|编辑任务/)
    await verifyListDetail(page, '隐藏任务详情')
    await verifyPager(page)

    await goNav(page, '事件管理', '事件管理')
    await openAndCancelEditor(page, /新建事件/, /新建事件|编辑事件/)
    await verifyListDetail(page, '隐藏事件详情')
    await verifyPager(page)

    await goNav(page, '用户与角色', '权限管理')
    await openAndCancelEditor(page, /新增用户/, /新增用户|编辑用户/)
    await page.locator('.segmented-tabs').getByRole('button', { name: '菜单与资源权限' }).click()
    await expect(page.getByRole('heading', { name: '菜单授权概览' })).toBeVisible()
    await expect(page.getByRole('heading', { name: '资源权限矩阵' })).toBeVisible()

    await goNav(page, 'AI Copilot 配置', 'AI Copilot 配置')
    await page.locator('.provider-card', { hasText: '本地模型' }).click()
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
