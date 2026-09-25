import { expect, test } from '@playwright/test'

const adminPassword = process.env.ADMIN_PASSWORD || 'OpsCore2026'

async function login(page) {
  await page.goto('/')
  await page.getByLabel('账号').fill('admin')
  await page.getByLabel('密码').fill(adminPassword)
  await page.getByRole('button', { name: '进入控制台' }).click()
  await expect(page.locator('.topbar h1')).toHaveText('首页健康总览')
}

test('long Copilot replies scroll inside a fixed window and persist after reload', async ({ page }) => {
  const lastLine = '明细 120：建议核对对应资产的负责人和最近变更记录。'
  const answer = Array.from({ length: 120 }, (_, index) => `- 明细 ${index + 1}：建议核对对应资产的负责人和最近变更记录。`).join('\n')
  await page.route('**/api/copilot/chat', (route) => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ answer, provider: 'compatible', model: 'test-model' })
  }))

  await login(page)
  await page.getByRole('button', { name: '打开 AI Copilot' }).click()
  await page.getByRole('textbox', { name: '向 AI Copilot 提问' }).fill('列出较长的资产检查清单')
  await page.getByRole('button', { name: '发送' }).click()

  const copilot = page.getByRole('dialog', { name: 'OpsCore AI Copilot' })
  const body = copilot.locator('.copilot-body')
  await expect(body).toContainText(lastLine)
  const beforeReload = await body.evaluate((element) => ({
    panelHeight: element.closest('.copilot').getBoundingClientRect().height,
    messageHeight: element.scrollHeight,
    viewportHeight: window.innerHeight,
    clientHeight: element.clientHeight,
    overflowY: getComputedStyle(element).overflowY
  }))
  expect(beforeReload.panelHeight).toBeLessThanOrEqual(beforeReload.viewportHeight - 47)
  expect(beforeReload.messageHeight).toBeGreaterThan(beforeReload.clientHeight)
  expect(beforeReload.overflowY).toBe('auto')
  await expect(copilot.locator('.copilot-input')).toBeVisible()

  await page.reload()
  await expect(page.locator('.topbar h1')).toHaveText('首页健康总览')
  await page.getByRole('button', { name: '打开 AI Copilot' }).click()
  await expect(page.locator('.chat.ai').last()).toContainText(lastLine)
  await expect(page.getByRole('textbox', { name: '向 AI Copilot 提问' })).toBeVisible()
})
