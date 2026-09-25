import { expect, test } from '@playwright/test'

const adminPassword = process.env.ADMIN_PASSWORD || 'OpsCore2026'

async function login(page) {
  await page.goto('/')
  await page.getByLabel('账号').fill('admin')
  await page.getByLabel('密码').fill(adminPassword)
  await page.getByRole('button', { name: '进入控制台' }).click()
  await expect(page.locator('.topbar h1')).toHaveText('首页健康总览')
}

test('collapsed category icons open navigation to implemented workspaces', async ({ page }) => {
  await login(page)
  const brand = page.locator('.sidebar .brand-lockup')
  const brandLogo = brand.locator('img')
  await expect(brandLogo).toBeVisible()
  await expect(brandLogo).toHaveAttribute('alt', 'OpsCore logo')
  await expect(brand.getByText('OpsCore', { exact: true })).toBeVisible()
  const brandBox = await brand.boundingBox()
  const toggleBox = await page.getByRole('button', { name: '展开菜单' }).boundingBox()
  expect(toggleBox.y - (brandBox.y + brandBox.height)).toBeGreaterThanOrEqual(18)

  const destinations = [
    ['资产管理', '资产台账（CMDB）', '资产台账（CMDB）'],
    ['协同与事件响应', '值班管理', '值班管理'],
    ['权限管理', '菜单与资源权限', '权限管理'],
    ['系统配置', 'AI Copilot 配置', 'AI Copilot 配置']
  ]

  for (const [group, destination, title] of destinations) {
    await page.getByRole('button', { name: `展开${group}菜单` }).click()
    await expect(page.getByRole('button', { name: new RegExp(destination) })).toBeVisible()
    await page.getByRole('button', { name: new RegExp(destination) }).click()
    await expect(page.locator('.topbar h1')).toHaveText(title)
    if (destination === '菜单与资源权限') {
      await expect(page.locator('.nav-child.active')).toContainText('菜单与资源权限')
    }
    await page.getByLabel('收起菜单').click()
  }
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
