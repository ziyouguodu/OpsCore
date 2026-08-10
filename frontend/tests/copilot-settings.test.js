import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const source = await readFile(new URL('../src/components/CopilotSettingsView.vue', import.meta.url), 'utf8')

test('renders saved model profiles before the editor', () => {
  assert.match(source, /已保存模型/)
  assert.match(source, /copilot-profile-table/)
  assert.match(source, /当前启用/)
  assert.match(source, /新增配置/)
  assert.match(source, /编辑/)
  assert.match(source, /设为当前/)
})

test('shows only fields relevant to the selected provider', () => {
  assert.match(source, /v-if="config\.provider === 'local'"/)
  assert.match(source, /本地模型地址/)
  assert.match(source, /v-else/)
  assert.match(source, /API Endpoint/)
  assert.match(source, /API Key/)
})

test('keeps the profile editor closed until create or edit is requested', () => {
  assert.match(source, /v-if="editorOpen"/)
  assert.match(source, /取消/)
})
