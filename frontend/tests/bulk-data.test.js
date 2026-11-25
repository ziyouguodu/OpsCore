import assert from 'node:assert/strict'
import test from 'node:test'
import { bulkSchemas, parseCsv, toCsv } from '../src/bulk-data.js'

test('round trips CSV values containing commas, quotes and line breaks', () => {
  const records = [{ assetNo: 'asset-1', business: '支付,结算', deploymentInfo: '第一行\n含"引号"' }]
  const csv = toCsv(records, bulkSchemas.assets)
  const parsed = parseCsv(csv, bulkSchemas.assets)

  assert.equal(parsed.length, 1)
  assert.equal(parsed[0].assetNo, records[0].assetNo)
  assert.equal(parsed[0].business, records[0].business)
  assert.equal(parsed[0].deploymentInfo, records[0].deploymentInfo)
})

test('neutralizes spreadsheet formulas while preserving ordinary negative numbers', () => {
  const record = {
    assetNo: '=HYPERLINK("https://example.invalid","open")',
    type: ' +SUM(1,2)',
    vendor: '-1+HYPERLINK("https://example.invalid","open")',
    cpuArch: '\t@SUM(1,2)',
    sn: '-12.5',
    location: '+12.5'
  }
  const exported = parseCsv(toCsv([record], bulkSchemas.assets), bulkSchemas.assets)[0]
  assert.equal(exported.assetNo, "'=HYPERLINK(\"https://example.invalid\",\"open\")")
  assert.equal(exported.type, "' +SUM(1,2)")
  assert.equal(exported.vendor, "'-1+HYPERLINK(\"https://example.invalid\",\"open\")")
  assert.equal(exported.cpuArch, "'\t@SUM(1,2)")
  assert.equal(exported.sn, '-12.5')
  assert.equal(exported.location, '+12.5')
})

test('accepts field names as CSV headers and rejects mismatched templates', () => {
  assert.deepEqual(parseCsv('name,endpoint\nredis-main,192.0.2.1:6379', bulkSchemas.middleware), [
    { __row: 2, name: 'redis-main', endpoint: '192.0.2.1:6379' }
  ])
  assert.throws(() => parseCsv('other,value\na,b', bulkSchemas.assets), /表头与当前导入模板不匹配/)
})

test('marks required template columns without changing export headers', () => {
  const required = {
    assets: ['type', 'cpuArch', 'business', 'ipv4', 'environment', 'os', 'networkZone', 'cpu', 'memory', 'disk', 'deploymentInfo', 'owner', 'status'],
    middleware: ['name', 'kind', 'environment', 'networkZone', 'endpoint', 'business', 'owner']
  }
  for (const [resource, schema] of Object.entries(bulkSchemas)) {
    assert.deepEqual(schema.filter(([, , mandatory]) => mandatory).map(([key]) => key), required[resource])
    const template = toCsv([], schema, { template: true })
    for (const [key, label] of schema) {
      assert.ok(template.includes(`"${label}${required[resource].includes(key) ? '（必填）' : ''}"`))
    }
    assert.ok(!toCsv([], schema).includes('（必填）'))
    const item = Object.fromEntries(schema.map(([key]) => [key, `value-${key}`]))
    const csv = `${template}\r\n${schema.map(([key]) => item[key]).join(',')}`
    assert.deepEqual(parseCsv(csv, schema), [{ __row: 2, ...item }])
  }
})
