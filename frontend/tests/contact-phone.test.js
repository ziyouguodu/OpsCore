import assert from 'node:assert/strict'
import test from 'node:test'
import { phoneDialValue } from '../src/contact-phone.js'

test('normalizes supported telephone values for a safe tel link', () => {
  assert.equal(phoneDialValue('13800138000'), '13800138000')
  assert.equal(phoneDialValue(' +86 138-0013-8000 '), '+8613800138000')
  assert.equal(phoneDialValue('010-12345678'), '01012345678')
  for (const value of ['', '未配置', '12', 'javascript:alert(1)', 'tel:123456', '123;456', '123\n456', '+123456789012345678901']) assert.equal(phoneDialValue(value), '')
})
