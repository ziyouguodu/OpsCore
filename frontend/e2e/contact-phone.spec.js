import { expect, test } from '@playwright/test'

test('contact phone supports copy, safe dialing, empty state and Escape focus return', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write'])
  await page.addInitScript(() => sessionStorage.setItem('opscore.token', 'contact-test-token'))
  const primary = { id: '1', name: '联系测试主值', role: '主值班', since: '08:00', until: '20:00', phone: '+86 138-0013-8000' }
  await page.route('**/api/**', async route => {
    const url = new URL(route.request().url())
    let body = []
    if (url.pathname === '/api/auth/me') body = { id: 1, username: 'contact', displayName: '联系测试', roles: ['ops_engineer'], mustChangePassword: false }
    else if (url.pathname === '/api/dashboard/workspace') body = { source: 'real', statistics: {}, incidents: [], tasks: [], businessSystems: [], duty: { source: 'real', primary, backup: { id: '2', name: '未配号备班', role: '备份值班', phone: '' }, handovers: [], pendingCount: 0 }, trend: { source: 'real', status: 'unavailable', points: [] } }
    else if (/\/api\/(assets|middleware|tasks|incidents)$/.test(url.pathname)) body = { items: [], total: 0, pageCount: 1, options: { business: [], networkZone: [] }, counts: {} }
    else if (url.pathname === '/api/duty-center') body = { revision: 0, data: {} }
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(body) })
  })
  await page.goto('/#dashboard')
  const trigger = page.getByRole('button', { name: '联系值班', exact: true })
  await expect(page.locator('.kpi-duty')).not.toContainText(primary.phone)
  await trigger.click()
  const dialog = page.getByRole('dialog', { name: '联系值班 · 真实业务数据' })
  await expect(dialog.getByRole('link', { name: `拨打${primary.name}联系电话` })).toHaveAttribute('href', 'tel:+8613800138000')
  await expect(dialog).toContainText('未配置联系方式')
  await expect(dialog.getByRole('link')).toHaveCount(1)
  await dialog.getByRole('button', { name: `复制${primary.name}联系电话` }).click()
  await expect(dialog.getByRole('status')).toHaveText('号码已复制')
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(primary.phone)
  await page.keyboard.press('Escape')
  await expect(trigger).toBeFocused()
  await page.setViewportSize({ width: 390, height: 844 })
  await trigger.click()
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390)
  await page.screenshot({ path: '../output/playwright/phone-contact-mobile.png' })
  await page.keyboard.press('Escape')
  await context.clearPermissions()
  await page.evaluate(() => { navigator.clipboard.writeText = async () => { throw new Error('denied') } })
  await trigger.click()
  await dialog.getByRole('button', { name: `复制${primary.name}联系电话` }).click()
  await expect(dialog.getByRole('status')).toHaveText('复制失败，请手动选择号码复制')
})
