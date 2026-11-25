import { expect, test } from '@playwright/test'

const adminPassword = process.env.ADMIN_PASSWORD || 'OpsCore2026'

async function login(page) {
  await page.goto('/')
  await page.getByLabel('账号').fill('admin')
  await page.getByLabel('密码').fill(adminPassword)
  await page.getByRole('button', { name: '进入控制台' }).click()
  await expect(page.locator('.review-dashboard .hero h1')).toBeVisible()
}

test('shows the expanded OpsCore category menu and navigates to implemented workspaces', async ({ page }) => {
  await login(page)
  const brand = page.locator('.review-topbar .brand')
  const brandLogo = brand.locator('img')
  await expect(brandLogo).toBeVisible()
  await expect(brandLogo).toHaveAttribute('alt', 'OpsCore logo')
  await expect(brand.getByText('OpsCore', { exact: true })).toBeVisible()
  await expect(page.locator('.review-sidebar .brand')).toHaveCount(0)
  await expect(page.locator('.review-sidebar .nav-title')).toHaveCount(0)
  const assets = page.getByRole('button', { name: '资产管理', exact: true })
  await expect(assets).toHaveAttribute('aria-expanded', 'true')
  const beforeCollapse = page.url()
  await assets.click()
  await expect(assets).toHaveAttribute('aria-expanded', 'false')
  await expect(page.getByRole('button', { name: '资产台账（CMDB）', exact: true })).toBeHidden()
  expect(page.url()).toBe(beforeCollapse)
  await assets.press('Enter')
  await expect(assets).toHaveAttribute('aria-expanded', 'true')
  await expect(page.locator('.review-sidebar button > svg')).toHaveCount(await page.locator('.review-sidebar button:not(.drawer-close)').count())

  const destinations = [
    ['资产管理', '资产台账（CMDB）', '资产台账（CMDB）'],
    ['协同与事件响应', '值班管理', '值班管理'],
    ['权限管理', '用户与角色', '权限管理'],
    ['系统配置', 'AI Copilot 配置', 'AI Copilot 配置']
  ]

  for (const [group, destination, title] of destinations) {
    const toggle = page.getByRole('button', { name: group, exact: true })
    if (await toggle.getAttribute('aria-expanded') === 'false') await toggle.click()
    await expect(page.getByRole('button', { name: destination, exact: true })).toBeVisible()
    await page.getByRole('button', { name: destination, exact: true }).click()
    await expect(page.locator('.workspace-page-header h1')).toHaveText(title)
    if (destination === '菜单与资源权限') {
      await expect(page.locator('.nav-child.active')).toContainText('菜单与资源权限')
    }
    await page.getByRole('button', { name: '首页仪表盘' }).click()
  }
  await page.getByRole('button', { name: '权限管理', exact: true }).click()
  await expect(page.locator('.review-dashboard .hero h1')).toBeVisible()
  await page.getByRole('button', { name: '权限管理', exact: true }).press('Enter')
  await page.getByRole('button', { name: '用户与角色', exact: true }).click()
  await expect(page.locator('.review-sidebar .nav.active')).toHaveCount(1)
  await expect(page.locator('.workspace-page-header h1')).toHaveText('权限管理')
  const settings = page.getByRole('button', { name: '系统配置', exact: true })
  await settings.click()
  await expect(page.locator('.workspace-page-header h1')).toHaveText('权限管理')
  await settings.press('Enter')
  await page.getByRole('button', { name: 'AI Copilot 配置', exact: true }).click()
  await expect(page.locator('.workspace-page-header h1')).toHaveText('AI Copilot 配置')
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(await page.evaluate(() => innerWidth))
})

test('permissions subpages navigate only through the sidebar and keep deep links on desktop and mobile', async ({ page }) => {
  await login(page)
  const region = page.getByRole('region', { name: '权限管理工作区' })
  const sidebar = page.getByRole('complementary', { name: '主导航', includeHidden: true })
  for (const width of [1280, 390]) {
    await page.setViewportSize({ width, height: 844 })
    for (const [tab, label, heading] of [['users', '用户与角色', '账号列表'], ['resources', '菜单与资源权限', '菜单授权概览']]) {
      if (width === 390) await page.getByRole('button', { name: '打开导航', exact: true }).click()
      const group = sidebar.getByRole('button', { name: '权限管理', exact: true })
      if (await group.getAttribute('aria-expanded') === 'false') await group.click()
      await sidebar.getByRole('button', { name: label, exact: true }).click()
      await expect(page).toHaveURL(new RegExp(`#permissions/${tab}$`))
      await expect(region.getByRole('heading', { name: heading, exact: true })).toBeVisible()
      await expect(region.locator('.segmented-tabs')).toHaveCount(0)
      await expect(region.getByRole('button', { name: /^(用户与角色|菜单与资源权限)$/ })).toHaveCount(0)
      await expect(sidebar.locator('.nav-child[aria-current="page"]')).toHaveText(label)
      if (width === 390) await expect(sidebar).toBeHidden()
      await page.reload()
      await expect(region.getByRole('heading', { name: heading, exact: true })).toBeVisible()
      await expect(page.locator('h1')).toHaveCount(1)
      expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(width)
    }
  }
  await page.goto('/#permissions/resources')
  await expect(region.getByRole('heading', { name: '资源权限矩阵', exact: true })).toBeVisible()
})

test('renders Copilot Markdown as readable structured content and keeps HTML inert', async ({ page }) => {
  const answer = [
    '## 当前概况',
    '',
    '**影响**：共有 `3` 项资产。',
    '',
    '| 类别 | 数量 |',
    '| --- | ---: |',
    '| 资产 | 3 |',
    '',
    '**下一步建议**：',
    '1. 核实 `<script>alert(1)</script>`',
    '',
    '```json',
    '{"healthy": true}',
    '```'
  ].join('\n')
  await page.route('**/api/copilot/chat', (route) => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ answer, provider: 'compatible', model: 'test-model' })
  }))
  await login(page)
  await page.getByRole('button', { name: '打开 AI Copilot' }).click()
  await page.getByRole('textbox', { name: '向 AI Copilot 提问' }).fill('请总结当前运维情况')
  await page.getByRole('button', { name: '发送' }).click()

  const response = page.locator('.chat.ai').last()
  await expect(response.locator('h3')).toHaveText('当前概况')
  await expect(response.locator('strong')).toContainText(['影响', '下一步建议'])
  await expect(response.locator('table')).toContainText('资产')
  await expect(response.locator('code.copilot-inline-code').first()).toHaveText('3')
  await expect(response.locator('pre.copilot-code-block')).toContainText('{"healthy": true}')
  await expect(response.locator('script')).toHaveCount(0)
  await expect(response).toContainText('<script>alert(1)</script>')
})
