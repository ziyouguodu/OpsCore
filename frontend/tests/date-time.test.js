import assert from 'node:assert/strict'
import test from 'node:test'

import { formatDateTime, toAPITimestamp, toDateTimeLocal } from '../src/date-time.js'

test('serializes datetime-local values with an explicit timezone', () => {
  const local = '2026-07-10T08:30'
  const serialized = toAPITimestamp(local)
  assert.equal(serialized, new Date(local).toISOString())
  assert.match(serialized, /Z$/)
})

test('round trips API timestamps into datetime-local controls', () => {
  const timestamp = '2026-07-10T00:30:00.000Z'
  assert.equal(toAPITimestamp(toDateTimeLocal(timestamp)), timestamp)
})

test('preserves empty optional timestamps', () => {
  assert.equal(toAPITimestamp(''), '')
  assert.equal(toDateTimeLocal(''), '')
})

test('formats API timestamps for dense list views', () => {
  assert.notEqual(formatDateTime('2026-07-10T00:30:00.000Z'), '2026-07-10T00:30:00.000Z')
  assert.equal(formatDateTime(''), '-')
})
