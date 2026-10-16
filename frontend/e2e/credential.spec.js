import { expect, test } from '@playwright/test'

const assets = [1, 2].map(id => ({ id, assetNo: `credential-asset-${id}`, type: '虚拟机', cpuArch: 'x86_64', business: '凭据测试', ipv4: `192.0.2.${id}`, environment: '研发', os: 'Linux', networkZone: '测试区', cpu: '2C', memory: '4GB', disk: '50GB', deploymentInfo: '测试', owner: '测试负责人', status: '运行中' }))
const middleware = [1, 2].map(id => ({ id, name: `credential-instance-${id}`, kind: 'Redis', environment: '研发', networkZone: '测试区', endpoint: `192.0.2.${id}:6379`, business: '凭据测试', owner: '测试负责人', status: '运行中' }))

async function setup(page, kind, role = 'super_admin') {
  const saved = []
  const metadata = { loginUrl: 'https://console.example', username: 'existing-user', secret: '', notes: 'existing notes', hasSecret: true }
  let storedSecret = 'stored-secret'
  await page.addInitScript(() => sessionStorage.setItem('opscore.token', 'credential-test-token'))
  await page.route('**/api/**', async route => {
    const request = route.request()
    const url = new URL(request.url())
    let status = 200
    let body = []
    if (url.pathname === '/api/auth/me') body = { id: 1, username: 'credential-user', displayName: '凭据测试用户', roles: [role], mustChangePassword: false }
    else if (url.pathname === '/api/dashboard/workspace') body = { source: 'real', statistics: {}, incidents: [], tasks: [], duty: { source: 'real' }, trend: { source: 'real', status: 'unavailable', points: [] } }
    else if (url.pathname === '/api/security/credential-verification') {
      expect(request.method()).toBe('GET')
      body = { hasPassword: true }
    } else if (/\/credential(?:\/reveal)?$/.test(url.pathname)) {
      expect(role).toBe('super_admin')
      const id = Number(url.pathname.split('/')[3])
      if (url.pathname.endsWith('/reveal')) {
        const { password } = request.postDataJSON()
        if (password !== 'unified-test-password') {
          status = password === 'limited' ? 429 : 403
          body = { error: '统一二次校验失败' }
        } else body = { ...metadata, id: 99, middlewareId: id, secret: storedSecret }
      } else if (request.method() === 'PUT') {
        const payload = request.postDataJSON()
        expect(Object.keys(payload).sort()).toEqual(['loginUrl', 'notes', 'secret', 'username'])
        saved.push(payload)
        if (payload.secret) storedSecret = payload.secret
        Object.assign(metadata, payload, { secret: '' })
        body = { ...metadata, id: 99, middlewareId: id }
      } else if (id === 2) {
        status = 404
        body = { error: '未保存凭据' }
      } else body = { ...metadata, id: 99, middlewareId: id }
    } else if (/\/api\/(assets|middleware)\/\d+$/.test(url.pathname) && request.method() === 'PUT') {
      body = request.postDataJSON()
      const records = url.pathname.includes('/assets/') ? assets : middleware
      Object.assign(records.find(item => item.id === body.id), body)
    } else if (/\/api\/(assets|middleware|tasks|incidents)$/.test(url.pathname)) {
      const items = url.pathname.endsWith('/assets') ? assets : url.pathname.endsWith('/middleware') ? middleware : []
      body = { items, total: items.length, page: 1, pageCount: 1, options: { business: [], networkZone: [] }, counts: {} }
    } else if (url.pathname === '/api/duty-center') body = { revision: 1, data: {} }
    await route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) })
  })
  await page.goto(`/#${kind === 'assets' ? 'cmdb' : 'middleware'}`)
  await expect(page.locator('tbody tr.clickable-row')).toHaveCount(2)
  return saved
}

for (const kind of ['assets', 'middleware']) {
  const label = kind === 'assets' ? '资产' : '实例'
  const loadLabel = kind === 'assets' ? '加载登录信息' : '加载账号密码'
  const saveLabel = kind === 'assets' ? '保存登录信息' : '保存账号密码'

  test(`${label}凭据统一校验、显隐、编辑和记录隔离`, async ({ page }) => {
    const saved = await setup(page, kind)
    const rows = page.locator('tbody tr.clickable-row')
    await rows.first().getByRole('button', { name: '详情', exact: true }).click()
    const detail = page.locator('aside').filter({ has: page.locator('.credential-box') })
    const secret = detail.getByLabel('密码或密钥', { exact: true })
    await detail.getByRole('button', { name: loadLabel }).click()
    await expect(detail.getByLabel('登录账号')).toHaveValue('existing-user')
    await expect(secret).toHaveAttribute('type', 'password')
    await expect(detail.getByRole('button', { name: '显示密码/密钥' })).toHaveCount(0)
    for (const wrong of ['wrong', 'limited']) {
      await detail.getByLabel('统一二次校验密码').fill(wrong)
      await detail.getByRole('button', { name: '二次校验查看' }).click()
      await expect(detail.getByText(/无法查看密码\/密钥/)).toBeVisible()
      await expect(detail.getByLabel('统一二次校验密码')).toHaveValue('')
      await expect(secret).toHaveValue('')
      await expect(page.getByRole('button', { name: '进入控制台' })).toHaveCount(0)
      expect(await page.evaluate(() => sessionStorage.getItem('opscore.token'))).toBe('credential-test-token')
    }
    await detail.getByLabel('统一二次校验密码').fill('unified-test-password')
    await detail.getByRole('button', { name: '二次校验查看' }).click()
    await expect(secret).toHaveAttribute('type', 'text')
    await expect(secret).toHaveValue('stored-secret')
    await detail.getByRole('button', { name: '隐藏密码/密钥' }).click()
    await expect(secret).toHaveAttribute('type', 'password')
    await detail.getByRole('button', { name: '显示密码/密钥' }).click()
    await expect(secret).toHaveAttribute('type', 'text')
    await detail.getByRole('button', { name: loadLabel }).click()
    await expect(secret).toHaveValue('')
    await expect(secret).toHaveAttribute('type', 'password')
    await secret.fill('replacement-secret')
    await detail.getByRole('button', { name: saveLabel }).click()
    await expect(detail.getByText(/已保存$/, { exact: false })).toBeVisible()
    await expect(secret).toHaveValue('')
    await expect(secret).toHaveAttribute('type', 'password')
    expect(saved.at(-1).secret).toBe('replacement-secret')

    await rows.first().getByRole('button', { name: '编辑', exact: true }).click()
    const editor = page.locator('.editor-panel')
    await expect(editor.getByLabel('登录用户名')).toHaveValue('existing-user')
    await expect(editor.getByLabel('登录密码或密钥')).toHaveValue('')
    await expect(editor.getByLabel('登录信息备注')).toHaveValue('existing notes')
    const address = editor.getByLabel(kind === 'assets' ? '登录地址' : '管理地址或连接入口', { exact: true })
    await expect(address).toHaveValue('https://console.example')
    await editor.getByLabel('登录密码或密钥').fill('edit-only-secret')
    await editor.getByRole('button', { name: '保存修改' }).click()
    await expect(editor).toBeHidden()
    expect(saved.at(-1)).toEqual({ loginUrl: 'https://console.example', username: 'existing-user', notes: 'existing notes', secret: 'edit-only-secret' })
    await expect(secret).toHaveValue('')

    await detail.getByLabel('统一二次校验密码').fill('unified-test-password')
    await detail.getByRole('button', { name: '二次校验查看' }).click()
    await expect(secret).toHaveValue('edit-only-secret')
    await rows.nth(1).getByRole('button', { name: '详情', exact: true }).click()
    await expect(secret).toHaveValue('')
    await expect(secret).toHaveAttribute('type', 'password')
    await expect(detail.getByLabel('统一二次校验密码')).toHaveValue('')
    await detail.getByRole('button', { name: loadLabel }).click()
    await expect(detail.getByText(kind === 'assets' ? '当前未保存登录信息' : '当前未保存实例账号密码', { exact: true })).toBeVisible()
    await detail.getByRole('button', { name: `隐藏${label}详情` }).click()
    await expect(detail).toBeHidden()
  })

  test(`${label}编辑表单明确必填且工程师没有凭据入口`, async ({ page }) => {
    await setup(page, kind, 'ops_engineer')
    await page.getByRole('button', { name: `新增${label}`, exact: false }).click()
    const editor = page.locator('.editor-panel')
    await expect(editor.locator('[required]')).toHaveCount(kind === 'assets' ? 13 : 7)
    await expect(editor.getByText(/（必填）/)).toHaveCount(kind === 'assets' ? 13 : 7)
    if (kind === 'assets') {
      await expect(editor.getByLabel('资产编号')).toHaveAttribute('aria-required', 'false')
      await expect(editor.getByText('资产编号（可选）')).toBeVisible()
    }
    await expect(editor.getByLabel('登录密码或密钥')).toHaveCount(0)
    await editor.getByRole('button', { name: '取消', exact: true }).click()
    const rows = page.locator('tbody tr.clickable-row')
    await rows.first().getByRole('button', { name: '编辑', exact: true }).click()
    await expect(editor.locator('[required]')).toHaveCount(kind === 'assets' ? 14 : 7)
    if (kind === 'assets') await expect(editor.getByText('资产编号（必填）')).toBeVisible()
    await editor.getByRole('button', { name: '取消', exact: true }).click()
    await rows.first().getByRole('button', { name: '详情', exact: true }).click()
    await expect(page.getByText(kind === 'assets' ? '当前角色无权查看登录信息。运维工程师默认不可见。' : '当前角色无权查看实例账号密码。运维工程师默认不可见。')).toBeVisible()
    await expect(page.getByLabel('统一二次校验密码')).toHaveCount(0)
  })
}
