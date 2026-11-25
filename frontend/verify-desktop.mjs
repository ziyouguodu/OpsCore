import { chromium } from 'playwright'

const browser = await chromium.launch()
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
const errors = []
page.on('pageerror', e => errors.push(e.message.split('\n')[0]))

await page.goto('http://localhost:5173/')
await page.getByLabel('账号').fill('admin')
await page.getByLabel('密码').fill('OpsCore2026')
await page.getByRole('button', { name: '进入控制台' }).click()
await page.waitForSelector('.duty-center, .review-dashboard', { state: 'attached', timeout: 15000 })

console.log('=== 首页值班卡片（1440px）===')
const card = await page.evaluate(() => {
  const c = document.querySelector('.kpi-duty')
  const open = c.querySelector('.duty-open')
  const handoff = c.querySelector('.duty-handoff button')
  const r = e => { const b = e.getBoundingClientRect(); return { w: Math.round(b.width), h: Math.round(b.height) } }
  return {
    text: c.innerText.replace(/\n/g, ' | '),
    wholeCardClickable: !!open,
    openSize: r(open),
    cardSize: r(c),
    handoffSeparate: !!handoff,
    openCoversMostOfCard: r(open).h >= r(c).h * 0.5
  }
})
console.log(JSON.stringify(card, null, 1))

console.log('\n点击卡片主体 -> 应进入值班管理')
await page.locator('.kpi-duty .duty-open').click()
await page.waitForTimeout(500)
console.log('  当前标题:', await page.locator('.topbar h1').first().textContent(), '| hash:', await page.evaluate(() => location.hash))

console.log('\n点击「查看交接」-> 应打开弹窗而不跳转')
await page.goto('http://localhost:5173/#dashboard')
await page.waitForTimeout(800)
await page.locator('.kpi-duty .duty-handoff button').click()
await page.waitForTimeout(400)
console.log('  弹窗标题:', await page.locator('.review-dialog h3, dialog h3').first().textContent().catch(() => '(无)'))
console.log('  hash 仍为:', await page.evaluate(() => location.hash))
await page.keyboard.press('Escape')
await page.waitForTimeout(300)

console.log('\n=== 值班页标签栏（1440px）===')
await page.goto('http://localhost:5173/#oncall')
await page.waitForSelector('.duty-tabbar', { timeout: 15000 })
await page.waitForTimeout(800)
const tabMetrics = []
for (const label of ['概览', '排班日历', '排班配置', '值班列表', '交接班日志', '升级策略']) {
  await page.locator('.duty-tabbar button', { hasText: label }).click()
  await page.waitForTimeout(200)
  const m = await page.evaluate(() => {
    const tb = document.querySelector('.duty-tabbar')
    const center = document.querySelector('.duty-center')
    const view = document.querySelector('.duty-view')
    return {
      tabbarH: Math.round(tb.getBoundingClientRect().height),
      tabbarRight: Math.round(tb.getBoundingClientRect().right),
      centerRight: Math.round(center.getBoundingClientRect().right),
      viewW: Math.round(view.getBoundingClientRect().width),
      viewLeft: Math.round(view.getBoundingClientRect().left),
      docOverflow: document.documentElement.scrollWidth > window.innerWidth
    }
  })
  tabMetrics.push({ label, ...m })
}
for (const m of tabMetrics) {
  console.log(`  ${m.label.padEnd(6)} tabbarH=${String(m.tabbarH).padStart(3)} 右缘=${m.tabbarRight}/${m.centerRight} 内容区宽=${m.viewW} 起始=${m.viewLeft} 溢出=${m.docOverflow ? '是' : '否'}`)
}

await page.screenshot({ path: '/Users/mac/Desktop/work/OpsCore/output/playwright/oncall-1440-config.png', fullPage: true })
await page.locator('.duty-tabbar button', { hasText: '概览' }).click()
await page.waitForTimeout(300)
await page.screenshot({ path: '/Users/mac/Desktop/work/OpsCore/output/playwright/oncall-1440-overview.png', fullPage: true })
await page.goto('http://localhost:5173/#dashboard')
await page.waitForSelector('.kpi-duty', { timeout: 10000 })
await page.waitForTimeout(600)
await page.screenshot({ path: '/Users/mac/Desktop/work/OpsCore/output/playwright/dashboard-duty-card.png', fullPage: true })

console.log('\npageerrors:', errors.length ? errors.slice(0, 3) : '(无)')
await browser.close()
