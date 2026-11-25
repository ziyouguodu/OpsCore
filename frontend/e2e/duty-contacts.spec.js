import { expect, test } from '@playwright/test'

const password = process.env.ADMIN_PASSWORD || 'OpsCore2026'

test('administrator phone maintenance persists, duty contacts use current user data and clear without snapshot fallback', async ({ page, request }) => {
  const login = await request.post('/api/auth/login', { data: { username: 'admin', password } })
  expect(login.ok()).toBeTruthy()
  const { token } = await login.json()
  const headers = { Authorization: `Bearer ${token}` }
  const json = async (method, path, data) => {
    const response = await request[method](`/api${path}`, { headers, data })
    expect(response.ok(), `${method} ${path}`).toBeTruthy()
    return response.status() === 204 ? null : response.json()
  }
  const original = await json('get', '/duty-center')
  const seed = Date.now()
  const created = []
  try {
    for (let i = 0; i < 2; i++) {
      created.push(await json('post', '/users', { username: `contact-${seed}-${i}`, displayName: `联系测试${i}`, password: 'ContactTest2026!', mustChangePassword: false, roles: ['ops_engineer'], phone: i ? '' : '+86 138-0013-8000' }))
    }
    const originalPhone = '+86 138-0013-8000'
    const legacyPayload = { username: created[0].username, displayName: created[0].displayName, mustChangePassword: false, roles: ['ops_engineer'] }
    expect((await json('put', `/users/${created[0].id}`, legacyPayload)).phone).toBe(originalPhone)
    const people = created.map((user, i) => ({ id: `contact-${user.id}`, userId: user.id, username: user.username, name: user.displayName, role: '运维工程师', team: '联系测试组', count: 0, next: '待安排', status: '空闲' }))
    const today = new Intl.DateTimeFormat('sv-SE', { timeZone: 'Asia/Shanghai' }).format(new Date())
    const data = { teams: [{ name: '联系测试组' }], members: people, schedules: [], assignments: { [today]: { primary: people[0].name, backup: people[1].name } }, currentPeople: [{ id: people[0].id, name: people[0].name, role: '主值班', team: '联系测试组', since: '08:00', until: '20:00', phone: '19999999999', status: '在线' }], handovers: [], escalation: { name: '', team: '', severity: '', levels: [] } }
    let saved = await json('put', '/duty-center', { revision: original.revision, data })
    expect(saved.data.currentPeople[0].phone).toBe('+86 138-0013-8000')
    const directory = await json('get', '/user-directory')
    expect(directory.every(user => !Object.hasOwn(user, 'phone'))).toBe(true)
    await page.addInitScript(token => sessionStorage.setItem('opscore.token', token), token)
    await page.goto('/#dashboard')
    await page.getByRole('button', { name: '联系值班', exact: true }).click()
    const dialog = page.getByRole('dialog', { name: '联系值班 · 真实业务数据' })
    await expect(dialog).toContainText('+86 138-0013-8000')
    await expect(dialog).toContainText('未配置联系方式')
    await expect(dialog.getByRole('link', { name: `拨打${people[0].name}联系电话` })).toHaveAttribute('href', 'tel:+8613800138000')
    await page.keyboard.press('Escape')
    await expect(page.getByRole('button', { name: '联系值班', exact: true })).toBeFocused()
    await page.goto('/#permissions/users')
    const row = page.getByRole('row').filter({ hasText: created[0].username })
    await row.getByRole('button', { name: '编辑', exact: true }).click()
    const editor = page.locator('.editor-panel')
    await expect(editor.getByLabel('联系电话', { exact: true })).toHaveValue('+86 138-0013-8000')
    await editor.getByLabel('联系电话', { exact: true }).fill('13900139000')
    await editor.getByRole('button', { name: '保存用户', exact: true }).click()
    await expect(editor).toBeHidden()
    await expect(row).toContainText('13900139000')
    await page.goto('/#oncall')
    const primary = page.locator('.duty-role-card.primary')
    await expect(primary).toContainText('13900139000')
    await expect(primary.getByRole('link', { name: `拨打${people[0].name}联系电话` })).toHaveAttribute('href', 'tel:13900139000')
    await page.reload()
    await expect(primary).toContainText('13900139000')
    await page.screenshot({ path: '../output/playwright/phone-duty-desktop.png', fullPage: true })
    await page.setViewportSize({ width: 390, height: 844 })
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390)
    await page.screenshot({ path: '../output/playwright/phone-duty-mobile.png', fullPage: true })
    await page.setViewportSize({ width: 1280, height: 900 })
    await page.goto('/#permissions/users')
    await row.getByRole('button', { name: '编辑', exact: true }).click()
    await editor.getByLabel('联系电话', { exact: true }).fill('')
    await editor.getByRole('button', { name: '保存用户', exact: true }).click()
    await expect(editor).toBeHidden()
    await page.goto('/#oncall')
    await expect(primary).toContainText('未配置联系方式')
    await expect(primary.getByRole('link')).toHaveCount(0)
    saved = await json('get', '/duty-center')
    expect(saved.data.currentPeople[0].phone).toBe('')
    expect((await json('put', `/users/${created[0].id}`, legacyPayload)).phone).toBe('')
    const workspace = await json('get', '/dashboard/workspace')
    expect(workspace.duty.primary.phone).toBe('')
    const audit = await json('get', '/audit-events?limit=100')
    expect(JSON.stringify(audit)).not.toContain('13900139000')
    expect(JSON.stringify(audit)).not.toContain(originalPhone)
    const operator = await request.post('/api/auth/login', { data: { username: created[0].username, password: 'ContactTest2026!' } })
    const operatorData = await operator.json()
    expect(Object.hasOwn(operatorData.user, 'phone')).toBe(false)
    const opsToken = operatorData.token
    const opsHeaders = { Authorization: `Bearer ${opsToken}` }
    expect((await request.put(`/api/users/${created[1].id}`, { headers: opsHeaders, data: { ...created[1], phone: '13800138000' } })).status()).toBe(403)
    const opsDuty = await request.get('/api/duty-center', { headers: opsHeaders })
    expect(opsDuty.ok()).toBeTruthy()
    expect((await opsDuty.json()).data.members.every(member => Object.hasOwn(member, 'phone'))).toBe(true)
  } finally {
    const current = await json('get', '/duty-center')
    await json('put', '/duty-center', { revision: current.revision, data: original.data })
    for (const user of created.reverse()) await json('delete', `/users/${user.id}`)
  }
})
