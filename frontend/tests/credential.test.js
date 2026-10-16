import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import vm from 'node:vm'
import { bulkSchemas } from '../src/bulk-data.js'

const source = await readFile(new URL('../src/App.vue', import.meta.url), 'utf8')
const functions = ['credentialPayload', 'resetAssetFormCredential', 'resetMiddlewareFormCredential', 'resetAssetCredentialState', 'resetMiddlewareCredentialState', 'editAsset', 'editMiddleware', 'loadCredential', 'loadMiddlewareCredential', 'revealCredential', 'revealMiddlewareCredential', 'saveCredential', 'saveMiddlewareCredential', 'saveAsset', 'saveMiddleware', 'hasAssetFormCredential', 'hasMiddlewareFormCredential', 'closeAssetForm', 'closeMiddlewareForm', 'resetAssetForm', 'resetMiddlewareForm']

function harness(api, admin = true) {
  const ref = value => ({ value })
  const credential = () => ({ loginUrl: '', username: '', secret: '', notes: '', hasSecret: false })
  const context = vm.createContext({
    api, bulkSchemas, isSampleRecord: () => false, canManageCredentials: ref(admin), error: ref(''),
    state: { assets: [], middleware: [] }, notify: () => {},
    loadAssetPage: async () => {}, loadMiddlewarePage: async () => {}, loadDashboard: async () => {},
    selectedAsset: ref({ id: 1 }), selectedMiddleware: ref({ id: 1 }),
    credential: credential(), middlewareCredential: credential(),
    credentialReveal: { password: '', revealed: false, visible: false },
    middlewareCredentialReveal: { password: '', revealed: false, visible: false },
    credentialMessage: ref(''), middlewareCredentialMessage: ref(''),
    newAsset: {}, newMiddleware: {}, assetFormCredential: credential(), middlewareFormCredential: credential(),
    assetFormOpen: ref(false), middlewareFormOpen: ref(false),
    assetFormCredentialReady: ref(true), middlewareFormCredentialReady: ref(true),
    assetFormCredentialExists: ref(false), middlewareFormCredentialExists: ref(false),
    assetFormCredentialRequest: 0, middlewareFormCredentialRequest: 0,
    assetCredentialRequest: 0, middlewareCredentialRequest: 0
  })
  for (const name of functions) {
    const match = source.match(new RegExp(`(?:async )?function ${name}\\([^]*?\\n\\}`))
    assert.ok(match, name)
    vm.runInContext(match[0], context)
  }
  return context
}

for (const [kind, edit, reset, form, data, ready, selected, reveal, detail, load, show, save, message] of [
  ['assets', 'editAsset', 'resetAssetFormCredential', 'newAsset', 'assetFormCredential', 'assetFormCredentialReady', 'selectedAsset', 'credentialReveal', 'credential', 'loadCredential', 'revealCredential', 'saveCredential', 'credentialMessage'],
  ['middleware', 'editMiddleware', 'resetMiddlewareFormCredential', 'newMiddleware', 'middlewareFormCredential', 'middlewareFormCredentialReady', 'selectedMiddleware', 'middlewareCredentialReveal', 'middlewareCredential', 'loadMiddlewareCredential', 'revealMiddlewareCredential', 'saveMiddlewareCredential', 'middlewareCredentialMessage']
]) {
  const metadata = { id: 99, middlewareId: 1, loginUrl: 'https://console.example', username: 'operator', notes: 'existing note', hasSecret: true }

  test(`${kind}: editing prefills only redacted business metadata and accepts missing credentials`, async () => {
    const calls = []
    const c = harness(async path => { calls.push(path); return metadata })
    await c[edit]({ id: 1 })
    assert.deepEqual(JSON.parse(JSON.stringify(c[data])), { loginUrl: metadata.loginUrl, username: 'operator', notes: 'existing note', secret: '', hasSecret: false })
    assert.equal(c[ready].value, true)
    assert.deepEqual(calls, [`/${kind}/1/credential`])
    c.api = async () => { throw Object.assign(new Error('missing'), { status: 404 }) }
    await c[edit]({ id: 2 })
    assert.equal(c[ready].value, true)
    assert.equal(c[data].username, '')
    assert.equal(c.error.value, '')
    c.api = async () => { throw Object.assign(new Error('failed'), { status: 500 }) }
    await c[edit]({ id: 3 })
    assert.equal(c[ready].value, false)
    assert.match(c.error.value, /failed/)
  })

  test(`${kind}: switching or closing the editor discards late metadata`, async () => {
    const pending = []
    const c = harness(() => new Promise(resolve => pending.push(resolve)))
    const first = c[edit]({ id: 1 })
    const second = c[edit]({ id: 2 })
    pending[1]({ ...metadata, username: 'second' })
    await second
    pending[0](metadata)
    await first
    assert.equal(c[form].id, 2)
    assert.equal(c[data].username, 'second')
    const third = c[edit]({ id: 3 })
    c[reset]()
    pending[2](metadata)
    await third
    assert.equal(c[data].username, '')
  })

  test(`${kind}: failed verification clears passwords; successful verification, load and save control visibility`, async () => {
    const puts = []
    const c = harness(async (path, options) => {
      if (path.endsWith('/reveal')) {
        const { password } = JSON.parse(options.body)
        if (password !== 'secondary-password') throw Object.assign(new Error('wrong secondary password'), { status: 403 })
        return { ...metadata, secret: 'stored-secret' }
      }
      if (options?.method === 'PUT') puts.push(JSON.parse(options.body))
      return metadata
    })
    c[reveal].password = 'wrong'
    c[detail].secret = 'old-secret'
    await c[show]()
    assert.equal(c[reveal].password, '')
    assert.equal(c[reveal].revealed, false)
    assert.equal(c[detail].secret, '')
    assert.match(c[message].value, /wrong secondary password/)
    c[reveal].password = 'secondary-password'
    await c[show]()
    assert.equal(c[detail].secret, 'stored-secret')
    assert.equal(c[reveal].visible, true)
    assert.equal(c[reveal].revealed, true)
    assert.equal(c[reveal].password, '')
    await c[load]()
    assert.equal(c[detail].secret, '')
    assert.equal(c[reveal].visible, false)
    c[reveal].password = 'secondary-password'
    await c[show]()
    c[detail].username = 'edited-user'
    await c[save]()
    assert.deepEqual(puts[0], { loginUrl: metadata.loginUrl, username: 'edited-user', secret: 'stored-secret', notes: metadata.notes })
    assert.equal(c[detail].secret, '')
    assert.equal(c[reveal].revealed, false)
    assert.equal(c[reveal].visible, false)
  })

  test(`${kind}: late reveal cannot expose a previous record's secret`, async () => {
    let complete
    const c = harness(() => new Promise(resolve => { complete = resolve }))
    c[reveal].password = 'secondary-password'
    const request = c[show]()
    c[selected].value = { id: 2 }
    complete({ ...metadata, secret: 'old-record-secret' })
    await request
    assert.equal(c[detail].secret, '')
    assert.equal(c[reveal].revealed, false)
  })

  test(`${kind}: form save snapshots credentials and leaves a newer editor untouched`, async () => {
    const saveForm = kind === 'assets' ? 'saveAsset' : 'saveMiddleware'
    const open = kind === 'assets' ? 'assetFormOpen' : 'middlewareFormOpen'
    const calls = []
    let complete
    const c = harness(async (path, options) => {
      if (path === `/${kind}/1`) return new Promise(resolve => { complete = resolve })
      if (options?.method === 'PUT') calls.push({ path, body: JSON.parse(options.body) })
      return metadata
    })
    Object.assign(c[form], Object.fromEntries(bulkSchemas[kind].filter(([, , required]) => required).map(([key]) => [key, 'value'])), { id: 1, assetNo: 'asset-a' })
    Object.assign(c[data], { username: 'user-a', secret: 'secret-a' })
    c[open].value = true
    const pending = c[saveForm]()
    assert.equal(c[ready].value, false)
    await c[saveForm]()
    await c[edit]({ id: 2 })
    Object.assign(c[data], { username: 'user-b', secret: 'secret-b' })
    c[selected].value = { id: 2 }
    c[detail].username = 'detail-b'
    complete({ id: 1 })
    await pending
    assert.equal(calls.length, 1)
    assert.equal(calls[0].path, `/${kind}/1/credential`)
    assert.equal(calls[0].body.username, 'user-a')
    assert.equal(calls[0].body.secret, 'secret-a')
    assert.equal(c[form].id, 2)
    assert.equal(c[data].secret, 'secret-b')
    assert.equal(c[detail].username, 'detail-b')
    assert.equal(c[selected].value.id, 2)
    assert.equal(c[open].value, true)
    assert.equal(c[ready].value, true)
  })

  test(`${kind}: credential save failure keeps the editor available for retry`, async () => {
    const saveForm = kind === 'assets' ? 'saveAsset' : 'saveMiddleware'
    const open = kind === 'assets' ? 'assetFormOpen' : 'middlewareFormOpen'
    const c = harness(async path => {
      if (path.endsWith('/credential')) throw new Error('credential write failed')
      return { id: 1 }
    })
    Object.assign(c[form], Object.fromEntries(bulkSchemas[kind].filter(([, , required]) => required).map(([key]) => [key, 'value'])), { id: 1, assetNo: 'asset-a' })
    c[data].secret = 'retry-secret'
    c[open].value = true
    await c[saveForm]()
    assert.equal(c[form].id, 1)
    assert.equal(c[open].value, true)
    assert.equal(c[ready].value, true)
    assert.equal(c[data].secret, 'retry-secret')
    assert.match(c.error.value, /credential write failed/)
  })

  test(`${kind}: engineers cannot load, reveal or save credential data`, async () => {
    let calls = 0
    const c = harness(async () => { calls++; return metadata }, false)
    await c[edit]({ id: 1 })
    await c[load]()
    await c[show]()
    await c[save]()
    assert.equal(calls, 0)
  })
}

test('asset validation includes all thirteen create fields and the edit asset number', async () => {
  const c = harness(async () => { throw new Error('must not submit') })
  Object.assign(c.newAsset, Object.fromEntries(bulkSchemas.assets.filter(([, , required]) => required).map(([key]) => [key, 'value'])))
  c.newAsset.status = ''
  await c.saveAsset()
  assert.match(c.error.value, /状态/)
  c.newAsset.status = '运行中'
  c.newAsset.id = 1
  await c.saveAsset()
  assert.match(c.error.value, /资产编号/)
})

test('components use visible requirement labels and verification-gated secret controls', async () => {
  for (const component of ['AssetView', 'MiddlewareView']) {
    const text = await readFile(new URL(`../src/components/${component}.vue`, import.meta.url), 'utf8')
    assert.match(text, /bulkSchemas\./)
    assert.match(text, /'（必填）' : '（可选）'/)
    assert.match(text, /:required=/)
    assert.match(text, /:aria-required=/)
    assert.match(text, /credentialReveal\.revealed && credentialReveal\.visible \? 'text' : 'password'/)
    assert.match(text, /v-if="credentialReveal\.revealed"/)
    assert.match(text, /不是登录密码/)
  }
})
