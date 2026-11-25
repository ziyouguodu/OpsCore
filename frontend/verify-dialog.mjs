import { chromium } from 'playwright'

const browser = await chromium.launch()
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
await page.goto('http://localhost:5173/')
await page.getByLabel('账号').fill('admin')
await page.getByLabel('密码').fill('OpsCore2026')
await page.getByRole('button', { name: '进入控制台' }).click()
await page.waitForSelector('.kpi-duty', { timeout: 15000 })
await page.waitForTimeout(800)

await page.locator('.kpi-duty .duty-handoff button').click()
await page.waitForTimeout(600)

const dialog = await page.evaluate(() => {
  const d = document.querySelector('dialog[open], .review-dialog[open], dialog')
  if (!d) return { found: false }
  const r = d.getBoundingClientRect()
  return {
    found: true,
    open: d.hasAttribute('open'),
    visible: r.width > 0 && r.height > 0,
    tag: d.tagName.toLowerCase(),
    classes: d.className,
    text: d.innerText.replace(/\n/g, ' | ').slice(0, 400),
    h3: [...d.querySelectorAll('h2,h3')].map(n => n.textContent.trim()),
    ariaLabel: d.getAttribute('aria-label'),
    titleAttr: d.getAttribute('title')
  }
})
console.log('弹窗状态:', JSON.stringify(dialog, null, 1))

const anyOpen = await page.locator('dialog[open]').count()
console.log('dialog[open] 数量:', anyOpen)
const visibleText = await page.getByText('值班交接').count()
console.log('包含"值班交接"文案数量:', visibleText)
await page.screenshot({ path: '/Users/mac/Desktop/work/OpsCore/output/playwright/duty-handoff-dialog.png' })
await browser.close()
