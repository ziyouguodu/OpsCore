import assert from 'node:assert/strict'
import test from 'node:test'

const memory = new Map()
const sessionMemory = new Map()
globalThis.localStorage = {
  getItem: key => memory.get(key) || null,
  setItem: (key, value) => memory.set(key, value),
  removeItem: key => memory.delete(key)
}
globalThis.sessionStorage = {
  getItem: key => sessionMemory.get(key) || null,
  setItem: (key, value) => sessionMemory.set(key, value),
  removeItem: key => sessionMemory.delete(key)
}

const client = await import('../src/api.js')

test('clears the token and notifies the application when an authenticated request returns 401', async () => {
  let expired = false
  assert.equal(typeof client.setSessionExpiredHandler, 'function')
  client.setSessionExpiredHandler(() => { expired = true })
  client.setToken('expired-token')
  globalThis.fetch = async () => ({
    ok: false,
    status: 401,
    json: async () => ({ error: 'token expired' })
  })

  await assert.rejects(() => client.api('/assets'), /token expired/)
  assert.equal(client.getToken(), null)
  assert.equal(expired, true)
})

test('exposes the HTTP status on API failures', async () => {
  client.setToken('valid-token')
  globalThis.fetch = async () => ({
    ok: false,
    status: 409,
    json: async () => ({ error: 'conflict' })
  })

  await assert.rejects(async () => {
    await client.api('/tasks/1', { method: 'PATCH' })
  }, error => error.status === 409 && error.message === 'conflict')
})

test('keeps bearer tokens in tab-scoped session storage instead of persistent local storage', () => {
  client.setToken('tab-token')
  assert.equal(sessionMemory.get('opscore.token'), 'tab-token')
  assert.equal(memory.has('opscore.token'), false)
})
