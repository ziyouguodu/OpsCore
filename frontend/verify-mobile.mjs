import { chromium } from 'playwright'

const browser = await chromium.launch()
const page = await browser.newPage({ viewport: { width: 390, height: 844 } })
await page.goto('http://localhost:5173/')
await page.getByLabel('账号').fill('admin')
await page.getByLabel('密码').fill('OpsCore2026')
await page.getByRole('button', { name: '进入控制台' }).click()
await page.waitForSelector('.duty-center, .review-dashboard', { state: 'attached', timeout: 15000 })
await page.goto('http://localhost:5173/#oncall')
await page.waitForSelector('.duty-tabbar', { timeout: 15000 })
await page.waitForTimeout(1000)

// 滚到最右，检查最后两个标签能否到达
const before = await page.evaluate(() => {
  const tb = document.querySelector('.duty-tabbar')
  return { scrollLeft: tb.scrollLeft, scrollW: tb.scrollWidth, clientW: tb.clientWidth }
})

const last = page.locator('.duty-tabbar button', { hasText: '升级策略' })
await last.scrollIntoViewIfNeeded()
await page.waitForTimeout(300)

const after = await page.evaluate(() => {
  const tb = document.querySelector('.duty-tabbar')
  const tbRect = tb.getBoundingClientRect()
  return {
    scrollLeft: tb.scrollLeft,
    scrollW: tb.scrollWidth,
    clientW: tb.clientWidth,
    docScrollW: document.documentElement.scrollWidth,
    tabsVisible: [...tb.querySelectorAll('button')].filter(el => {
      const r = el.getBoundingClientRect()
      return r.left >= tbRect.left - 1 && r.right <= tbRect.right + 1
    }).map(el => el.textContent.trim())
  }
})

console.log('滚动前:', JSON.stringify(before))
console.log('滚动后:', JSON.stringify(after))

// 点击最后两个标签，确认可切换
for (const label of ['交接班日志', '升级策略']) {
  await page.locator('.duty-tabbar button', { hasText: label }).click()
  await page.waitForTimeout(300)
  const active = await page.evaluate(() => document.querySelector('.duty-tabbar button.active')?.textContent.trim())
  const heading = await page.locator('.duty-view h2').first().textContent()
  console.log(`  点击「${label}」-> active=${active}  标题=${heading}`)
}

await page.screenshot({ path: '/Users/mac/Desktop/work/OpsCore/output/playwright/oncall-390.png', fullPage: true })
await browser.close()
