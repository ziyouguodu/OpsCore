import test from 'node:test'
import assert from 'node:assert/strict'

import { incidentStatusOptions, taskStatusOptions } from '../src/workflow-status.js'

test('offers only the current task status and legal next transitions', () => {
  assert.deepEqual(taskStatusOptions('待确认'), ['待确认', '已完成', '处理中', '已关闭'])
  assert.deepEqual(taskStatusOptions('已关闭'), ['已关闭'])
})

test('offers only the current incident status and legal next transitions', () => {
  assert.deepEqual(incidentStatusOptions('已恢复'), ['已恢复', '已关闭', '处理中'])
  assert.deepEqual(incidentStatusOptions('已关闭'), ['已关闭'])
})
