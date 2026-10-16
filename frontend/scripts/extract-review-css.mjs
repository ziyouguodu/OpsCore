import { readFile, writeFile } from 'node:fs/promises'
import postcss from 'postcss'

const prototype = await readFile(new URL('../../deliverables/home-glass-framework-review.html', import.meta.url), 'utf8')
const css = prototype.match(/<style>([\s\S]*?)<\/style>/)[1]
const sheet = postcss.parse(css)
const scopedSurfaces = `.review-shell :where(.review-dashboard,.review-topbar,.review-sidebar,.review-dialog)`
sheet.walkRules((rule) => {
  rule.selectors = rule.selectors.map((selector) => {
    if (selector.startsWith(':root')) return selector
    if (selector === 'body') return '.review-shell'
    if (selector === '*') return `${scopedSurfaces},${scopedSurfaces} *`
    if (selector.startsWith('.layout')) return selector.replace('.layout', '.review-shell')
    if (selector.startsWith('.topbar')) return selector.replace('.topbar', '.review-topbar')
    if (selector.startsWith('.sidebar')) return selector.replace('.sidebar', '.review-sidebar')
    if (selector.startsWith('.main')) return selector.replace('.main', '.review-main')
    if (selector.startsWith('dialog')) return selector.replace('dialog', '.review-dialog')
    if (selector.startsWith('.drawer-open')) return selector.replace('.drawer-open', '.review-shell.drawer-open').replace('.sidebar', '.review-sidebar')
    if (selector.startsWith('.brand') && !selector.startsWith('.brand-line')) return `.review-topbar ${selector}`
    if (/^\.(nav|drawer-close)/.test(selector)) return `.review-sidebar ${selector}`
    if (/^\.(search|top-actions|account|avatar|mobile-toggle|theme)/.test(selector)) return `.review-topbar ${selector}`
    if (/^\.(dialog-body|dialog-actions|date-fields|date-field|date-help)/.test(selector)) return `.review-dialog ${selector}`
    if (/^(button|input|h[123]|p|small|svg|table|th|td|\*|\.icon|\.muted|\.primary|\.warning)/.test(selector)) return `${scopedSurfaces} ${selector}`
    return `.review-dashboard ${selector}`
  })
})
await writeFile(new URL('../src/review-framework.css', import.meta.url), sheet.toString() + '\n')
