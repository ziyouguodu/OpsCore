<script setup>
import { ref } from 'vue'
import { api } from '../api'
import { searchWorkspace } from '../dashboard-workspace'
import SvgIcon from './SvgIcon.vue'
import WorkspaceDialog from './WorkspaceDialog.vue'

defineProps({
  breadcrumb: { type: String, default: '工作台' }, title: { type: String, required: true },
  subtitle: { type: String, default: '' }, error: { type: String, default: '' },
  loading: { type: Boolean, default: false }, username: { type: String, default: '' },
  authenticated: { type: Boolean, default: false }, theme: { type: String, default: 'deepsea' },
  mobileOpen: { type: Boolean, default: false }, preview: { type: Object, default: null }
})
const emit = defineEmits(['logout', 'open-copilot', 'open-navigation', 'change-theme', 'navigate'])
const keyword = ref('')
const searchOpen = ref(false)
const notificationsOpen = ref(false)
const searching = ref(false)
const results = ref([])
const searchError = ref('')
let searchSequence = 0
async function search() {
  if (!keyword.value.trim()) return
  const sequence = ++searchSequence
  searchOpen.value = true
  searching.value = true
  searchError.value = ''
  results.value = []
  try {
    const response = await searchWorkspace(api, keyword.value.trim())
    if (sequence === searchSequence) results.value = response
  } catch (error) {
    if (sequence === searchSequence) { searchError.value = error.message; if (error.status === 401) searchOpen.value = false }
  } finally { if (sequence === searchSequence) searching.value = false }
}
function openResult(group) {
  searchOpen.value = false
  emit('navigate', group.target)
}
</script>

<template>
  <header class="topbar review-topbar">
    <button class="mobile-toggle" type="button" aria-label="打开导航" :aria-expanded="mobileOpen" aria-controls="workspace-navigation" @click="emit('open-navigation')"><SvgIcon class="icon" name="menu" /></button>
    <div class="brand"><img src="/opscore-ai-ops-logo-transparent.png" alt="OpsCore logo" /><div><strong>OpsCore</strong><small>智能运维中枢指挥平台</small></div></div>
    <div class="topbar-context" :class="{ 'dashboard-context': title === '首页健康总览' }"><span class="breadcrumb">{{ breadcrumb }}</span><h1>{{ title }}</h1><p>{{ subtitle }}</p></div>
    <form class="search" role="search" @submit.prevent="search"><span class="muted">⌕</span><input v-model="keyword" aria-label="全局搜索" placeholder="搜索资产、事件、任务…" /><kbd>Enter</kbd></form>
    <div v-if="authenticated" class="top-actions">
      <span v-if="loading" class="muted">加载中...</span>
      <button id="notification" type="button" aria-label="通知" @click="notificationsOpen = true"><SvgIcon class="icon" name="bell" /></button>
      <select class="theme-select" aria-label="选择主题色调" :value="theme" @change="emit('change-theme', $event.target.value)"><option value="deepsea">深海蓝</option><option value="graphite">石墨紫</option><option value="aurora">极光钛</option></select>
      <button class="primary" type="button" title="打开 AI Copilot" aria-label="打开 AI Copilot" @click="emit('open-copilot')"><SvgIcon class="icon" name="copilot" /><span class="copilot-label">AI Copilot</span></button>
      <button class="account" type="button" @click="emit('logout')"><span class="avatar">{{ username.slice(0, 1) }}</span>{{ username }} · 退出</button>
      <button class="mobile-search" type="button" aria-label="打开全局搜索" @click="searchOpen = true">⌕</button>
    </div>
    <div v-if="error" class="topbar-error" role="alert">{{ error }}</div>
    <WorkspaceDialog v-model:open="searchOpen" title="全局搜索 · 真实业务数据">
      <form class="dialog-search" @submit.prevent="search"><input v-model="keyword" aria-label="搜索关键词" autofocus placeholder="资产、实例、任务、事件" /><button type="submit" :disabled="searching">搜索</button></form>
      <p v-if="searching">查询中…</p><p v-if="searchError" role="alert">{{ searchError }}</p>
      <section v-for="group in results" :key="group.path" class="search-group"><h3>{{ group.label }} <span class="tag">真实数据 · {{ group.total || 0 }}</span></h3><p v-if="group.error">{{ group.error }}</p><p v-else-if="!group.items.length">暂无匹配结果</p><button v-for="item in group.items" :key="item.id" class="search-result" type="button" @click="openResult(group)">{{ item.title || item.name || item.assetNo || item.hostname || item.id }}<small>{{ item.business || item.status }}</small></button></section>
    </WorkspaceDialog>
    <WorkspaceDialog v-model:open="notificationsOpen" title="通知 · 未接入模块">
      <span class="tag">{{ preview?.source === 'demo' ? '演示数据 · 虚构通知' : '待接入' }}</span>
      <p v-if="!preview?.notifications?.length">暂无通知数据；可在首页显式开启独立演示。</p>
      <p v-for="item in preview?.notifications || []" :key="item.riskId">{{ item.title }} · {{ item.time || item.createdAt }}</p>
    </WorkspaceDialog>
  </header>
</template>
