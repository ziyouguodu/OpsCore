import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import vm from 'node:vm'

const source = await readFile(new URL('../src/App.vue', import.meta.url), 'utf8')
function harness(api) {
  const context = vm.createContext({ api, userSaving: { value: false }, error: { value: '' }, canManageUsers: { value: true }, newUser: { id: null, username: 'contact-a', displayName: 'A', phone: '13800138000', password: 'Password123!', mustChangePassword: false, role: 'ops_engineer' }, userFormOpen: { value: true }, state: { users: [], userDirectory: [] }, currentUserID: () => 99, auth: { user: {} }, notify: () => {}, loadDashboard: async () => {} })
  for (const name of ['saveUser', 'openUserForm', 'closeUserForm', 'editUser', 'resetUserForm']) {
    const match = source.match(new RegExp(`(?:async )?function ${name}\\([^]*?\\n\\}`))
    assert.ok(match, name)
    vm.runInContext(match[0], context)
  }
  return context
}

test('user save stays locked through associated refresh and cannot close a newer editor', async () => {
  let release
  const calls = []
  const context = harness(async (path, options) => {
    calls.push(path)
    if (path === '/user-directory') return new Promise(resolve => { release = resolve })
    return { id: 1, ...JSON.parse(options.body) }
  })
  const first = context.saveUser()
  await new Promise(resolve => setImmediate(resolve))
  assert.equal(context.userSaving.value, true)
  context.openUserForm()
  context.editUser({ id: 2, username: 'b', displayName: 'B', phone: '', roles: ['ops_engineer'] })
  await context.saveUser()
  assert.deepEqual(calls, ['/users', '/user-directory'])
  release([])
  await first
  assert.equal(context.userSaving.value, false)
  context.editUser({ id: 2, username: 'b', displayName: 'B', phone: '13900139000', roles: ['ops_engineer'] })
  assert.equal(context.newUser.id, 2)
  assert.equal(context.newUser.phone, '13900139000')
  assert.equal(context.userFormOpen.value, true)
})

test('failed user phone save preserves input and permits retry', async () => {
  const context = harness(async () => { throw new Error('validation failed') })
  await context.saveUser()
  assert.equal(context.newUser.phone, '13800138000')
  assert.equal(context.newUser.username, 'contact-a')
  assert.equal(context.userFormOpen.value, true)
  assert.equal(context.userSaving.value, false)
  assert.match(context.error.value, /validation failed/)
})
