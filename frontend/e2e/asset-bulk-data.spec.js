import { expect, test } from '@playwright/test'

const adminPassword = process.env.ADMIN_PASSWORD || 'OpsCore2026'

async function apiJSON(request, method, path, token, data) {
  const response = await request[method](`/api${path}`, {
    headers: { Authorization: `Bearer ${token}` },
    data
  })
  if (!response.ok()) throw new Error(`${method.toUpperCase()} ${path} failed: ${response.status()} ${await response.text()}`)
  if (response.status() === 204) return null
  return response.json()
}

async function login(page) {
  await page.goto('/')
  await page.getByLabel('账号').fill('admin')
  await page.getByLabel('密码').fill(adminPassword)
  await page.getByRole('button', { name: '进入控制台' }).click()
  await expect(page.locator('.topbar h1')).toHaveText('首页健康总览')
}

async function goNav(page, label, title) {
  const expand = page.getByLabel('展开菜单')
  if (await expand.isVisible().catch(() => false)) await expand.click()
  await page.locator('button.nav-child', { hasText: label }).first().click()
  await expect(page.locator('.topbar h1')).toHaveText(title)
}

async function readDownload(download) {
  const stream = await download.createReadStream()
  const chunks = []
  for await (const chunk of stream) chunks.push(chunk)
  return Buffer.concat(chunks).toString('utf8')
}

test('searches multiple IPs and imports/exports CSV in both asset workspaces', async ({ page, request }) => {
  const loginResponse = await request.post('/api/auth/login', { data: { username: 'admin', password: adminPassword } })
  expect(loginResponse.ok()).toBeTruthy()
  const { token } = await loginResponse.json()
  const seed = Date.now()
  const assetNos = [`bulk-a-${seed}`, `bulk-b-${seed}`, `bulk-import-${seed}`]
  const middlewareNames = [`bulk-mw-a-${seed}`, `bulk-mw-b-${seed}`, `bulk-mw-import-${seed}`]

  const assetPayload = (assetNo, ipv4) => ({
    assetNo, type: '虚拟机', vendor: 'E2E', cpuArch: 'x86_64', sn: `SN-${assetNo}`, location: 'E2E',
    business: 'E2E业务', ipv4, ipv6: '', environment: '研发', os: 'Linux', hostname: assetNo,
    networkZone: 'E2E区', cpu: '2C', memory: '4GB', disk: '50GB', deploymentInfo: 'E2E测试',
    owner: 'E2E', status: '运行中', hostMachine: ''
  })
  const middlewarePayload = (name, endpoint) => ({
    name, kind: 'Redis', version: '7', environment: '研发', networkZone: 'E2E区', endpoint,
    business: 'E2E业务', owner: 'E2E', status: '运行中'
  })

  try {
    await apiJSON(request, 'post', '/assets', token, assetPayload(assetNos[0], '192.0.2.101'))
    await apiJSON(request, 'post', '/assets', token, assetPayload(assetNos[1], '192.0.2.102'))
    await apiJSON(request, 'post', '/assets', token, assetPayload(`bulk-near-${seed}`, '192.0.2.1019'))
    await apiJSON(request, 'post', '/middleware', token, middlewarePayload(middlewareNames[0], '192.0.2.101:6379'))
    await apiJSON(request, 'post', '/middleware', token, middlewarePayload(middlewareNames[1], 'redis://192.0.2.102:6379/0'))
    await apiJSON(request, 'post', '/middleware', token, middlewarePayload(`bulk-mw-near-${seed}`, '192.0.2.1019:6379'))

    await login(page)
    await goNav(page, '资产台账', '资产台账（CMDB）')
    await page.getByRole('button', { name: '高级搜索' }).click()
    await page.getByRole('textbox', { name: '多个 IP 地址' }).fill('192.0.2.101, 192.0.2.102')
    const assetRows = page.locator('tbody tr.clickable-row')
    await expect(assetRows).toHaveCount(2)
    await expect(page.locator('tbody')).toContainText(assetNos[0])
    await expect(page.locator('tbody')).toContainText(assetNos[1])
    await expect(page.getByRole('button', { name: '导出已选' })).toBeDisabled()
    await page.getByLabel(`选择资产 ${assetNos[0]}`).check()
    await expect(page.getByText('已选 1 项')).toBeVisible()
    await page.getByRole('button', { name: '批量删除' }).click()
    await expect(page.getByTestId('confirm-dialog')).toBeVisible()
    await page.getByTestId('confirm-cancel').click()
    const selectedAssetDownloadPromise = page.waitForEvent('download')
    await page.getByRole('button', { name: '导出已选' }).click()
    const selectedAssetCsv = await readDownload(await selectedAssetDownloadPromise)
    expect(selectedAssetCsv).toContain(assetNos[0])
    expect(selectedAssetCsv).not.toContain(assetNos[1])
    await page.getByRole('button', { name: '清除选择' }).click()

    const importedAssetCsv = [
      '资产编号,类型,厂商,CPU架构,SN,物理位置,所属业务,IPv4,IPv6,环境,操作系统,主机名,网络区域,CPU规格,内存规格,磁盘规格,部署信息,负责人,状态,所在宿主机',
      `${assetNos[2]},虚拟机,E2E,x86_64,SN-${assetNos[2]},E2E,E2E业务,198.51.100.101,,研发,Linux,${assetNos[2]},E2E区,2C,4GB,50GB,E2E测试,E2E,运行中,`
    ].join('\r\n')
    await page.getByLabel('导入资产 CSV 文件').setInputFiles({ name: 'assets.csv', mimeType: 'text/csv', buffer: Buffer.from(importedAssetCsv) })
    await page.getByTestId('confirm-accept').click()
    await expect(page.getByText(/批量导入资产成功，共 1 条/)).toBeVisible()
    await page.getByLabel('选择本页资产').check()
    await expect(page.getByText('已选 2 项')).toBeVisible()
    const assetDownloadPromise = page.waitForEvent('download')
    await page.getByRole('button', { name: '导出已选' }).click()
    const assetCsv = await readDownload(await assetDownloadPromise)
    expect(assetCsv).toContain(assetNos[0])
    expect(assetCsv).toContain(assetNos[1])
    expect(assetCsv).not.toContain('登录密码')
    expect(assetCsv).not.toContain('secret')

    await page.getByRole('button', { name: '重置' }).click()
    await page.getByRole('textbox', { name: '资产关键词' }).fill(assetNos[2])
    await expect(page.locator('tbody tr.clickable-row')).toHaveCount(1)
    await page.getByLabel(`选择资产 ${assetNos[2]}`).check()
    const deleteAssetResponse = page.waitForResponse((response) => response.url().includes('/api/assets/') && response.request().method() === 'DELETE')
    await page.getByRole('button', { name: '批量删除' }).click()
    await page.getByTestId('confirm-accept').click()
    expect((await deleteAssetResponse).ok()).toBeTruthy()
    await expect(page.locator('tbody tr.clickable-row')).toHaveCount(0)

    await goNav(page, '中间件与数据库', '中间件与数据库')
    await page.getByRole('button', { name: '高级搜索' }).click()
    await page.getByRole('textbox', { name: '多个 IP 地址' }).fill('192.0.2.101 192.0.2.102')
    const middlewareRows = page.locator('tbody tr.clickable-row')
    await expect(middlewareRows).toHaveCount(2)
    await expect(page.locator('tbody')).toContainText(middlewareNames[0])
    await expect(page.locator('tbody')).toContainText(middlewareNames[1])
    await page.getByLabel('选择本页实例').check()
    await expect(page.getByText('已选 2 项')).toBeVisible()
    const selectedMiddlewareDownloadPromise = page.waitForEvent('download')
    await page.getByRole('button', { name: '导出已选' }).click()
    const selectedMiddlewareCsv = await readDownload(await selectedMiddlewareDownloadPromise)
    expect(selectedMiddlewareCsv).toContain(middlewareNames[0])
    expect(selectedMiddlewareCsv).toContain(middlewareNames[1])
    await page.getByRole('button', { name: '清除选择' }).click()

    const importedMiddlewareCsv = [
      '实例名称,类型,版本,环境,网络区域,访问地址,所属业务,负责人,状态,关联资产ID',
      `${middlewareNames[2]},Redis,7,研发,E2E区,198.51.100.102:6379,E2E业务,E2E,运行中,`
    ].join('\r\n')
    await page.getByLabel('导入实例 CSV 文件').setInputFiles({ name: 'middleware.csv', mimeType: 'text/csv', buffer: Buffer.from(importedMiddlewareCsv) })
    await page.getByTestId('confirm-accept').click()
    await expect(page.getByText(/批量导入实例成功，共 1 条/)).toBeVisible()
    await page.getByLabel('选择本页实例').check()
    await expect(page.getByText('已选 2 项')).toBeVisible()
    const middlewareDownloadPromise = page.waitForEvent('download')
    await page.getByRole('button', { name: '导出已选' }).click()
    const middlewareCsv = await readDownload(await middlewareDownloadPromise)
    expect(middlewareCsv).toContain(middlewareNames[0])
    expect(middlewareCsv).toContain(middlewareNames[1])
    expect(middlewareCsv).not.toContain('secret')

    await page.getByRole('button', { name: '重置' }).click()
    await page.getByRole('textbox', { name: '实例关键词' }).fill(middlewareNames[2])
    await expect(page.locator('tbody tr.clickable-row')).toHaveCount(1)
    await page.getByLabel(`选择实例 ${middlewareNames[2]}`).check()
    const deleteMiddlewareResponse = page.waitForResponse((response) => response.url().includes('/api/middleware/') && response.request().method() === 'DELETE')
    await page.getByRole('button', { name: '批量删除' }).click()
    await page.getByTestId('confirm-accept').click()
    expect((await deleteMiddlewareResponse).ok()).toBeTruthy()
    await expect(page.locator('tbody tr.clickable-row')).toHaveCount(0)
  } finally {
    for (const name of middlewareNames.concat(`bulk-mw-near-${seed}`)) {
      const found = await apiJSON(request, 'get', `/middleware?keyword=${encodeURIComponent(name)}`, token)
      for (const item of found.items || []) {
        if (item.name === name) await apiJSON(request, 'delete', `/middleware/${item.id}`, token)
      }
    }
    for (const assetNo of assetNos.concat(`bulk-near-${seed}`)) {
      const found = await apiJSON(request, 'get', `/assets?keyword=${encodeURIComponent(assetNo)}`, token)
      for (const item of found.items || []) {
        if (item.assetNo === assetNo) await apiJSON(request, 'delete', `/assets/${item.id}`, token)
      }
    }
  }
})
