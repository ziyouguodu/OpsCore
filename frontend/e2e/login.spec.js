import { expect, test } from '@playwright/test'

test('renders the OpsCore login screen', async ({ page }) => {
  await page.goto('/')

  await expect(page.getByRole('heading', { name: '智能运维中枢指挥平台' })).toBeVisible()
  await expect(page.getByLabel('账号')).toBeVisible()
  await expect(page.getByLabel('密码')).toBeVisible()
  await expect(page.getByRole('button', { name: '进入控制台' })).toBeVisible()
})
