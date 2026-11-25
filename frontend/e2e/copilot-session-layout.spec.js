import { expect, test } from '@playwright/test'

const adminPassword = process.env.ADMIN_PASSWORD || 'OpsCore2026'

async function login(page) {
  await page.goto('/')
  await page.getByLabel('账号').fill('admin')
  await page.getByLabel('密码').fill(adminPassword)
  await page.getByRole('button', { name: '进入控制台' }).click()
  await expect(page.locator('.review-dashboard .hero h1')).toBeVisible()
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
  await expect(page.locator('.review-dashboard .hero h1')).toBeVisible()
  await page.getByRole('button', { name: '打开 AI Copilot' }).click()
  await expect(page.locator('.chat.ai').last()).toContainText(lastLine)
  await expect(page.getByRole('textbox', { name: '向 AI Copilot 提问' })).toBeVisible()
})

test('logout clears Copilot profiles even when the previous session request finishes late', async ({ page }) => {
  let role = 'super_admin'
  let releaseProfiles
  let markProfilesRequested
  const profileGate = new Promise((resolve) => { releaseProfiles = resolve })
  const profileRequested = new Promise((resolve) => { markProfilesRequested = resolve })
  const respond = (route, body, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) })

  await page.route('**/api/**', async (route) => {
    const path = new URL(route.request().url()).pathname
    const user = { id: 1, username: role === 'super_admin' ? 'admin' : 'engineer', displayName: role, roles: [role], mustChangePassword: false }
    if (path === '/api/auth/login') return respond(route, { token: `test-${role}`, user })
    if (path === '/api/auth/me') return respond(route, user)
    if (path === '/api/copilot/configs') {
      if (role !== 'super_admin') return respond(route, { error: 'forbidden' }, 403)
      markProfilesRequested()
      await profileGate
      return respond(route, [{ id: 99, name: 'cached-profile-secret-name', provider: 'openai', endpoint: 'https://private.example/v1', model: 'private-model', hasApiKey: true, isActive: true }])
    }
    if (path === '/api/dashboard/workspace') return respond(route, { source: 'real', statistics: {} })
    if (path === '/api/oncall' || path === '/api/user-directory') return respond(route, [])
    if (path === '/api/users' || path === '/api/audit-events') return respond(route, [])
    if (path === '/api/security/credential-verification') return respond(route, { hasPassword: false })
    if (/^\/api\/(assets|middleware|tasks|incidents)$/.test(path)) return respond(route, { items: [], total: 0, page: 1, pageCount: 1, counts: {}, options: {} })
    return respond(route, {})
  })

  await page.goto('/#copilot-settings')
  await page.getByLabel('账号').fill('admin')
  await page.getByLabel('密码').fill('irrelevant-test-password')
  await page.getByRole('button', { name: '进入控制台' }).click()
  await profileRequested
  await expect(page.getByRole('button', { name: /退出/ })).toBeVisible()
  await page.getByRole('button', { name: /退出/ }).click()
  role = 'ops_engineer'
  releaseProfiles()

  await page.getByLabel('账号').fill('engineer')
  await page.getByLabel('密码').fill('irrelevant-test-password')
  await page.getByRole('button', { name: '进入控制台' }).click()
  await expect(page.locator('.copilot-profile-workspace')).toBeVisible()
  await expect(page.getByText('cached-profile-secret-name')).toHaveCount(0)
  await expect(page.getByText('private-model')).toHaveCount(0)
})
