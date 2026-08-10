import assert from 'node:assert/strict'
import test from 'node:test'

import { buildListPath } from '../src/list-query.js'

test('builds encoded server-side pagination and filter parameters', () => {
  const path = buildListPath('assets', { page: 2, pageSize: 25 }, {
    keyword: '支付 服务', type: '物理机', environment: '生产', business: '', advanced: true
  })
  const url = new URL(path, 'http://opscore.local')

  assert.equal(url.pathname, '/assets')
  assert.equal(url.searchParams.get('page'), '2')
  assert.equal(url.searchParams.get('pageSize'), '25')
  assert.equal(url.searchParams.get('keyword'), '支付 服务')
  assert.equal(url.searchParams.get('type'), '物理机')
  assert.equal(url.searchParams.get('environment'), '生产')
  assert.equal(url.searchParams.has('business'), false)
  assert.equal(url.searchParams.has('advanced'), false)
  assert.equal(url.searchParams.get('sort'), 'updatedAt')
  assert.equal(url.searchParams.get('order'), 'desc')
})
