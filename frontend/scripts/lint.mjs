import { readdir, readFile } from 'node:fs/promises'
import { extname, join, relative } from 'node:path'

const root = new URL('../src/', import.meta.url)
const allowedExtensions = new Set(['.js', '.vue', '.css'])
const rules = [
  { name: '原生浏览器弹窗', pattern: /window\.(?:alert|confirm|prompt)\s*\(/g },
  { name: '持久化认证数据', pattern: /localStorage\.(?:setItem|getItem)\s*\(/g },
  { name: '调试日志', pattern: /console\.(?:log|debug)\s*\(/g },
  { name: '未审计 HTML 注入', pattern: /\bv-html\s*=/g }
]

async function walk(directory) {
  const entries = await readdir(directory, { withFileTypes: true })
  const files = []
  for (const entry of entries) {
    const path = join(directory, entry.name)
    if (entry.isDirectory()) files.push(...await walk(path))
    else if (allowedExtensions.has(extname(entry.name))) files.push(path)
  }
  return files
}

const issues = []
for (const file of await walk(root.pathname)) {
  const source = await readFile(file, 'utf8')
  const label = relative(root.pathname, file)
  source.split('\n').forEach((line, index) => {
    if (/\s+$/.test(line)) issues.push(`${label}:${index + 1} 行尾空白`)
  })
  for (const rule of rules) {
    rule.pattern.lastIndex = 0
    for (const match of source.matchAll(rule.pattern)) {
      const line = source.slice(0, match.index).split('\n').length
      issues.push(`${label}:${line} ${rule.name}`)
    }
  }
}

if (issues.length) {
  console.error(`前端静态检查发现 ${issues.length} 个问题：`)
  for (const issue of issues) console.error(`- ${issue}`)
  process.exit(1)
}

console.info('前端静态检查通过')
