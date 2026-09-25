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

test('accepts field names as CSV headers and rejects mismatched templates', () => {
  assert.deepEqual(parseCsv('name,endpoint\nredis-main,192.0.2.1:6379', bulkSchemas.middleware), [
    { __row: 2, name: 'redis-main', endpoint: '192.0.2.1:6379' }
  ])
  assert.throws(() => parseCsv('other,value\na,b', bulkSchemas.assets), /表头与当前导入模板不匹配/)
})
