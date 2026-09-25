const unorderedItem = /^\s*[-*+]\s+(.+)$/
const orderedItem = /^\s*\d+[.)]\s+(.+)$/
const heading = /^(#{1,4})\s+(.+)$/
const fence = /^\s*```\s*([\w+-]*)\s*$/

function cells(line) {
  return line.trim().replace(/^\|/, '').replace(/\|$/, '').split('|').map((cell) => cell.trim())
}

function isTableDelimiter(line) {
  const values = cells(line)
  return line.includes('|') && values.length > 0 && values.every((value) => /^:?-{3,}:?$/.test(value))
}

function startsBlock(lines, index) {
  const line = lines[index]
  return fence.test(line) || heading.test(line) || /^\s*>/.test(line) ||
    unorderedItem.test(line) || orderedItem.test(line) || /^\s*(?:---+|\*\*\*+|___+)\s*$/.test(line) ||
    (index + 1 < lines.length && line.includes('|') && isTableDelimiter(lines[index + 1]))
}

export function parseCopilotMarkdown(markdown) {
  const lines = String(markdown ?? '').replace(/\r\n?/g, '\n').split('\n')
  const blocks = []
  let index = 0

  while (index < lines.length) {
    const line = lines[index]
    if (!line.trim()) {
      index += 1
      continue
    }

    const fenceMatch = line.match(fence)
    if (fenceMatch) {
      const code = []
      index += 1
      while (index < lines.length && !/^\s*```\s*$/.test(lines[index])) code.push(lines[index++])
      if (index < lines.length) index += 1
      blocks.push({ type: 'code', language: fenceMatch[1], text: code.join('\n') })
      continue
    }

    const headingMatch = line.match(heading)
    if (headingMatch) {
      blocks.push({ type: 'heading', level: headingMatch[1].length, text: headingMatch[2] })
      index += 1
      continue
    }

    if (index + 1 < lines.length && line.includes('|') && isTableDelimiter(lines[index + 1])) {
      const headers = cells(line)
      const rows = []
      index += 2
      while (index < lines.length && lines[index].includes('|') && lines[index].trim()) {
        rows.push(cells(lines[index++]))
      }
      blocks.push({ type: 'table', headers, rows })
      continue
    }

    if (/^\s*(?:---+|\*\*\*+|___+)\s*$/.test(line)) {
      blocks.push({ type: 'rule' })
      index += 1
      continue
    }

    if (/^\s*>/.test(line)) {
      const quote = []
      while (index < lines.length && /^\s*>/.test(lines[index])) quote.push(lines[index++].replace(/^\s*>\s?/, ''))
      blocks.push({ type: 'quote', lines: quote })
      continue
    }

    const unorderedMatch = line.match(unorderedItem)
    const orderedMatch = line.match(orderedItem)
    if (unorderedMatch || orderedMatch) {
      const ordered = Boolean(orderedMatch)
      const pattern = ordered ? orderedItem : unorderedItem
      const items = []
      while (index < lines.length) {
        const item = lines[index].match(pattern)
        if (!item) break
        items.push(item[1])
        index += 1
      }
      blocks.push({ type: 'list', ordered, items })
      continue
    }

    const paragraph = [line]
    index += 1
    while (index < lines.length && lines[index].trim() && !startsBlock(lines, index)) paragraph.push(lines[index++])
    blocks.push({ type: 'paragraph', lines: paragraph })
  }

  return blocks
}
