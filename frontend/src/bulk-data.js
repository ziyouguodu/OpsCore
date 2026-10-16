export const bulkSchemas = {
  assets: [
    ['assetNo', '资产编号'], ['type', '类型', true], ['vendor', '厂商'], ['cpuArch', 'CPU架构', true], ['sn', 'SN'],
    ['location', '物理位置'], ['business', '所属业务', true], ['ipv4', 'IPv4', true], ['ipv6', 'IPv6'],
    ['environment', '环境', true], ['os', '操作系统', true], ['hostname', '主机名'], ['networkZone', '网络区域', true],
    ['cpu', 'CPU规格', true], ['memory', '内存规格', true], ['disk', '磁盘规格', true], ['deploymentInfo', '部署信息', true],
    ['owner', '负责人', true], ['status', '状态', true], ['hostMachine', '所在宿主机']
  ],
  middleware: [
    ['name', '实例名称', true], ['kind', '类型', true], ['version', '版本'], ['environment', '环境', true],
    ['networkZone', '网络区域', true], ['endpoint', '访问地址', true], ['business', '所属业务', true], ['owner', '负责人', true],
    ['status', '状态'], ['assetId', '关联资产ID']
  ]
}

function csvCell(value) {
  const text = String(value ?? '')
  return `"${text.replaceAll('"', '""')}"`
}

export function toCsv(items, schema, { template = false } = {}) {
  const rows = [schema.map(([, label, required]) => template && required ? `${label}（必填）` : label)]
  for (const item of items) rows.push(schema.map(([key]) => item[key] ?? ''))
  return `\uFEFF${rows.map((row) => row.map(csvCell).join(',')).join('\r\n')}`
}

export function parseCsv(text, schema) {
  const source = String(text || '').replace(/^\uFEFF/, '')
  const rows = []
  let row = []
  let cell = ''
  let quoted = false
  for (let index = 0; index < source.length; index += 1) {
    const char = source[index]
    if (quoted) {
      if (char === '"' && source[index + 1] === '"') {
        cell += '"'
        index += 1
      } else if (char === '"') quoted = false
      else cell += char
    } else if (char === '"' && cell === '') quoted = true
    else if (char === ',') {
      row.push(cell)
      cell = ''
    } else if (char === '\n' || char === '\r') {
      if (char === '\r' && source[index + 1] === '\n') index += 1
      row.push(cell)
      if (row.some((value) => value.trim())) rows.push(row)
      row = []
      cell = ''
    } else cell += char
  }
  row.push(cell)
  if (row.some((value) => value.trim())) rows.push(row)
  if (quoted) throw new Error('CSV 文件包含未闭合的引号')
  if (rows.length < 2) throw new Error('CSV 文件中没有可导入的数据行')

  const headers = rows[0].map((header) => header.trim())
  const columns = schema.map(([key, label]) => headers.findIndex((header) => header === label || header === key || header === `${label}（必填）`))
  if (columns.every((index) => index < 0)) throw new Error('CSV 表头与当前导入模板不匹配')
  return rows.slice(1).map((values, index) => {
    const record = { __row: index + 2 }
    schema.forEach(([key], schemaIndex) => {
      const column = columns[schemaIndex]
      if (column >= 0) record[key] = (values[column] || '').trim()
    })
    return record
  })
}

export function downloadText(filename, content, type = 'text/csv;charset=utf-8') {
  const url = URL.createObjectURL(new Blob([content], { type }))
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}
