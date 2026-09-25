import test from 'node:test'
import assert from 'node:assert/strict'
import { loadCopilotMessages, saveCopilotMessages } from '../src/copilot-chat-storage.js'

function createStorage() {
  const entries = new Map()
  return {
    getItem: (key) => entries.get(key) ?? null,
    setItem: (key, value) => entries.set(key, value)
  }
}

test('restores a user conversation from session storage and keeps users isolated', () => {
  const storage = createStorage()
  const messages = [
    { role: 'user', text: '资产情况怎么样' },
    { role: 'ai', text: '共有 3 项资产。' }
  ]

  saveCopilotMessages(7, messages, storage)

  assert.deepEqual(loadCopilotMessages(7, storage), messages)
  assert.equal(loadCopilotMessages(8, storage)[0].role, 'ai')
  assert.match(loadCopilotMessages(8, storage)[0].text, /请描述需要分析/)
  assert.deepEqual(loadCopilotMessages('', storage)[0], {
    role: 'ai',
    text: '请描述需要分析的运维问题，我会基于当前账号可访问的数据给出证据和建议。'
  })
})

test('does not persist pending messages or restore malformed entries', () => {
  const storage = createStorage()
  saveCopilotMessages('7', [
    { role: 'user', text: '继续分析' },
    { role: 'ai', text: '正在分析…', pending: true },
    { role: 'unexpected', text: '忽略' }
  ], storage)

  assert.deepEqual(loadCopilotMessages('7', storage), [{ role: 'user', text: '继续分析' }])
  storage.setItem('opscore.copilot.messages.7', '{bad json')
  assert.match(loadCopilotMessages('7', storage)[0].text, /请描述需要分析/)
})

test('limits restored history to the latest 60 messages', () => {
  const storage = createStorage()
  const messages = Array.from({ length: 70 }, (_, index) => ({ role: 'user', text: `问题 ${index + 1}` }))

  saveCopilotMessages('7', messages, storage)

  const restored = loadCopilotMessages('7', storage)
  assert.equal(restored.length, 60)
  assert.equal(restored[0].text, '问题 11')
  assert.equal(restored.at(-1).text, '问题 70')
})
