import { expect, test } from '@playwright/test'

test('renders the OpsCore login screen', async ({ page }) => {
  await page.goto('/')

  const title = page.getByRole('heading', { name: '智能运维中枢指挥平台' })
  await expect(title).toBeVisible()
  const titleBox = await title.boundingBox()
  expect(titleBox?.height).toBeLessThan(80)
  await expect(page.getByLabel('账号')).toBeVisible()
  await expect(page.getByLabel('密码')).toBeVisible()
  await expect(page.getByRole('button', { name: '进入控制台' })).toBeVisible()
})
