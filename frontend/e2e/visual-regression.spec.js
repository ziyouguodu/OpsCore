import { expect, test } from '@playwright/test'
import { mkdir } from 'node:fs/promises'
import path from 'node:path'

const adminPassword = process.env.ADMIN_PASSWORD || 'OpsCore2026'
const screenshotDir = path.resolve(process.cwd(), '../output/playwright')

async function expectNoPageOverflow(page) {
  const dimensions = await page.evaluate(() => ({
    clientWidth: document.documentElement.clientWidth,
    scrollWidth: document.documentElement.scrollWidth
  }))
  expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth + 1)
}

async function login(page) {
  await page.getByLabel('账号').fill('admin')
  await page.getByLabel('密码').fill(adminPassword)
  await page.getByRole('button', { name: '进入控制台' }).click()
  await expect(page.locator('.topbar h1')).toHaveText('首页健康总览')
}

test('captures desktop and mobile OpsCore visual review surfaces', async ({ page }) => {
  await mkdir(screenshotDir, { recursive: true })
  const consoleErrors = []
  page.on('console', (message) => {
    if (message.type() === 'error' || message.type() === 'warning') consoleErrors.push(message.text())
  })

  await page.setViewportSize({ width: 1440, height: 900 })
  await page.goto('/')
  await page.evaluate(() => { localStorage.clear(); sessionStorage.clear() })
  await page.reload()
  await expect(page.getByRole('heading', { name: '智能运维中枢指挥平台' })).toBeVisible()
  await expectNoPageOverflow(page)
  await page.screenshot({ path: path.join(screenshotDir, '01-login-desktop.png'), fullPage: true })

  await login(page)
  const routes = [
    ['dashboard', '首页健康总览', '02-dashboard-desktop.png'],
    ['cmdb', '资产台账（CMDB）', '03-cmdb-desktop.png'],
    ['middleware', '中间件与数据库', '04-middleware-desktop.png'],
    ['oncall', '值班管理', '05-oncall-desktop.png'],
    ['tasks', '任务跟踪', '06-tasks-desktop.png'],
    ['incidents', '事件管理', '07-incidents-desktop.png'],
    ['permissions/users', '权限管理', '08-permissions-desktop.png'],
    ['copilot-settings', 'AI Copilot 配置', '09-copilot-settings-desktop.png']
  ]
  for (const [route, title, filename] of routes) {
    await page.goto(`/#${route}`)
    await expect(page.locator('.topbar h1')).toHaveText(title)
    await expectNoPageOverflow(page)
    await page.screenshot({ path: path.join(screenshotDir, filename), fullPage: true })
  }
  await page.locator('.provider-card').filter({ hasText: '本地模型' }).click()
  await expect(page.getByLabel('本地模型地址')).toBeVisible()
  await expect(page.getByLabel('API Endpoint')).toBeHidden()
  await expectNoPageOverflow(page)
  await page.screenshot({ path: path.join(screenshotDir, '09b-copilot-editor-desktop.png'), fullPage: true })

  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/#dashboard')
  await expect(page.locator('.topbar h1')).toHaveText('首页健康总览')
  await expectNoPageOverflow(page)
  await page.screenshot({ path: path.join(screenshotDir, '10-dashboard-mobile.png'), fullPage: true })
  await page.getByRole('button', { name: '打开导航' }).click()
  await expect(page.locator('.sidebar')).toBeVisible()
  await expectNoPageOverflow(page)
  await page.screenshot({ path: path.join(screenshotDir, '11-navigation-mobile.png'), fullPage: true })

  await page.goto('/#copilot-settings')
  await expect(page.locator('.topbar h1')).toHaveText('AI Copilot 配置')
  await expect(page.getByRole('heading', { name: /已保存模型/ })).toBeVisible()
  await expectNoPageOverflow(page)
  await page.screenshot({ path: path.join(screenshotDir, '12-copilot-settings-mobile.png'), fullPage: true })

  expect(consoleErrors).toEqual([])
})
