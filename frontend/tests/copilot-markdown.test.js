import assert from 'node:assert/strict'
import test from 'node:test'
import { parseCopilotMarkdown } from '../src/copilot-markdown.js'

test('parses headings, emphasis, inline code, lists, tables and fenced code', () => {
  const blocks = parseCopilotMarkdown([
    '## 当前概况',
    '',
    '**资产**数量为 `3`。',
    '',
    '- 服务器：2',
    '- 数据库：1',
    '',
    '| 类别 | 数量 |',
    '| --- | ---: |',
    '| 资产 | 3 |',
    '',
    '```json',
    '{"healthy":true}',
    '```'
  ].join('\n'))

  assert.deepEqual(blocks.map((block) => block.type), ['heading', 'paragraph', 'list', 'table', 'code'])
  assert.deepEqual(blocks[2].items, ['服务器：2', '数据库：1'])
  assert.deepEqual(blocks[3], { type: 'table', headers: ['类别', '数量'], rows: [['资产', '3']] })
  assert.equal(blocks[4].language, 'json')
  assert.equal(blocks[4].text, '{"healthy":true}')
})

test('keeps raw HTML as plain text content and handles empty input', () => {
  assert.deepEqual(parseCopilotMarkdown('<script>alert(1)</script>'), [
    { type: 'paragraph', lines: ['<script>alert(1)</script>'] }
  ])
  assert.deepEqual(parseCopilotMarkdown(''), [])
})
