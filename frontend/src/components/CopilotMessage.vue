<script>
import { defineComponent, h } from 'vue'
import { parseCopilotMarkdown } from '../copilot-markdown.js'

function renderInline(source) {
  const nodes = []
  const pattern = /(\*\*[^*\n]+\*\*|__[^_\n]+__|`[^`\n]+`|\*[^*\n]+\*|_[^_\n]+_)/g
  let cursor = 0
  let match

  while ((match = pattern.exec(source))) {
    if (match.index > cursor) nodes.push(source.slice(cursor, match.index))
    const token = match[0]
    if (token.startsWith('**') || token.startsWith('__')) {
      nodes.push(h('strong', token.slice(2, -2)))
    } else if (token.startsWith('`')) {
      nodes.push(h('code', { class: 'copilot-inline-code' }, token.slice(1, -1)))
    } else {
      nodes.push(h('em', token.slice(1, -1)))
    }
    cursor = match.index + token.length
  }

  if (cursor < source.length) nodes.push(source.slice(cursor))
  return nodes
}

function renderLines(lines) {
  return lines.flatMap((line, index) => index === 0
    ? renderInline(line)
    : [h('br'), ...renderInline(line)])
}

function renderBlock(block, index) {
  if (block.type === 'heading') return h(`h${Math.min(block.level + 1, 5)}`, { key: index }, renderInline(block.text))
  if (block.type === 'paragraph') return h('p', { key: index }, renderLines(block.lines))
  if (block.type === 'code') {
    return h('pre', { key: index, class: 'copilot-code-block' }, [
      h('code', block.text)
    ])
  }
  if (block.type === 'list') {
    const tag = block.ordered ? 'ol' : 'ul'
    return h(tag, { key: index }, block.items.map((item) => h('li', renderInline(item))))
  }
  if (block.type === 'quote') return h('blockquote', { key: index }, renderLines(block.lines))
  if (block.type === 'rule') return h('hr', { key: index })
  if (block.type === 'table') {
    return h('div', { key: index, class: 'copilot-table-wrap' }, [
      h('table', { class: 'copilot-markdown-table' }, [
        h('thead', [h('tr', block.headers.map((cell) => h('th', renderInline(cell))))]),
        h('tbody', block.rows.map((row) => h('tr', row.map((cell) => h('td', renderInline(cell))))))
      ])
    ])
  }
  return null
}

export default defineComponent({
  name: 'CopilotMessage',
  props: {
    text: { type: String, default: '' }
  },
  setup(props) {
    return () => h('div', { class: 'copilot-message-content' }, parseCopilotMarkdown(props.text).map(renderBlock))
  }
})
</script>
