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

async function expectWorkspaceHeader(page, title) {
  const header = page.locator('.workspace-page-header')
  await expect(header).toHaveCount(1)
  await expect(page.locator('.panel > header.workspace-page-header, .duty-center > header.workspace-page-header')).toHaveCount(1)
  await expect(header.locator('h1')).toHaveText(title)
  await expect(page.locator('h1')).toHaveCount(1)
  await expect(page.locator('.topbar h1')).toHaveCount(0)
  await expect(header.locator('.breadcrumb')).toBeVisible()
  await expect(header.locator('p')).toHaveCount(1)
  const description = await header.locator('p').innerText()
  expect(description.trim()).not.toBe('')
  await expect(page.getByText(description, { exact: true })).toHaveCount(1)
}

async function login(page) {
  await page.getByLabel('账号').fill('admin')
  await page.getByLabel('密码').fill(adminPassword)
  await page.getByRole('button', { name: '进入控制台' }).click()
  await expect(page.locator('.review-dashboard .hero h1')).toBeVisible()
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
    ['dashboard', null, '02-dashboard-desktop.png'],
    ['cmdb', '资产台账（CMDB）', '03-cmdb-desktop.png'],
    ['middleware', '中间件与数据库', '04-middleware-desktop.png'],
    ['oncall', '值班管理', '05-oncall-desktop.png'],
    ['tasks', '任务跟踪', '06-tasks-desktop.png'],
    ['incidents', '事件管理', '07-incidents-desktop.png'],
    ['permissions/users', '权限管理', '08-permissions-desktop.png'],
    ['audit', '操作审计', '08b-audit-desktop.png'],
    ['copilot-settings', 'AI Copilot 配置', '09-copilot-settings-desktop.png']
  ]
  for (const [route, title, filename] of routes) {
    await page.goto(`/#${route}`)
    if (route === 'dashboard') {
      await expect(page.locator('.review-dashboard .hero h1')).toBeVisible()
      await expect(page.locator('h1')).toHaveCount(1)
      await expect(page.locator('.workspace-page-header, .topbar h1')).toHaveCount(0)
    } else {
      await expectWorkspaceHeader(page, title)
    }
    await expect(page.getByRole('search')).toBeVisible()
    await expect(page.getByRole('textbox', { name: '全局搜索', exact: true })).toBeVisible()
    await expectNoPageOverflow(page)
    await page.screenshot({ path: path.join(screenshotDir, filename), fullPage: true })
    if (route === 'oncall') {
      for (const label of ['排班日历', '排班配置', '值班列表', '交接班日志', '升级策略']) {
        await page.getByRole('tab', { name: label, exact: true }).click()
        await expectNoPageOverflow(page)
        await page.screenshot({ path: path.join(screenshotDir, `duty-desktop-${label}.png`), fullPage: true })
      }
    }
  }
  await page.locator('.provider-card').filter({ hasText: '本地模型' }).click()
  await expect(page.getByLabel('本地模型地址')).toBeVisible()
  await expect(page.getByLabel('API Endpoint')).toBeHidden()
  await expectNoPageOverflow(page)
  await page.screenshot({ path: path.join(screenshotDir, '09b-copilot-editor-desktop.png'), fullPage: true })

  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/#dashboard')
  await expect(page.locator('.review-dashboard .hero h1')).toBeVisible()
  await expectNoPageOverflow(page)
  await page.screenshot({ path: path.join(screenshotDir, '10-dashboard-mobile.png'), fullPage: true })
  await page.getByRole('button', { name: '打开导航' }).click()
  await expect(page.locator('.sidebar')).toBeVisible()
  await expectNoPageOverflow(page)
  await page.screenshot({ path: path.join(screenshotDir, '11-navigation-mobile.png'), fullPage: true })
  await page.keyboard.press('Escape')
  await expect(page.locator('.sidebar')).toBeHidden()

  for (const [route, title] of routes.filter(([route]) => route !== 'dashboard')) {
    await page.goto(`/#${route}`)
    await expectWorkspaceHeader(page, title)
    const search = page.getByRole('button', { name: '打开全局搜索', exact: true })
    await expect(search).toBeVisible()
    await search.click()
    await expect(page.getByRole('dialog', { name: '全局搜索 · 真实业务数据' }).getByRole('textbox', { name: '搜索关键词' })).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(search).toBeFocused()
    await expectNoPageOverflow(page)
    await page.screenshot({ path: path.join(screenshotDir, `workspace-mobile-${route.replaceAll('/', '-')}.png`), fullPage: true })
  }

  await page.goto('/#oncall')
  await expect(page.getByRole('tablist', { name: '值班工作区' })).toBeVisible()
  for (const label of ['概览', '排班日历', '排班配置', '值班列表', '交接班日志', '升级策略']) {
    await page.getByRole('tab', { name: label, exact: true }).click()
    await expectNoPageOverflow(page)
    await page.screenshot({ path: path.join(screenshotDir, `duty-mobile-${label}.png`), fullPage: true })
  }
  await page.goto('/#copilot-settings')
  await expectWorkspaceHeader(page, 'AI Copilot 配置')
  await expect(page.getByRole('heading', { name: /已保存模型/ })).toBeVisible()
  await expectNoPageOverflow(page)
  await page.screenshot({ path: path.join(screenshotDir, '12-copilot-settings-mobile.png'), fullPage: true })

  expect(consoleErrors).toEqual([])
})
