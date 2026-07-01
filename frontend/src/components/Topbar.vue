<script setup>
import SvgIcon from './SvgIcon.vue'

defineProps({
  breadcrumb: { type: String, default: '工作台' },
  title: { type: String, required: true },
  subtitle: { type: String, default: '' },
  error: { type: String, default: '' },
  loading: { type: Boolean, default: false },
  username: { type: String, default: 'admin' },
  authenticated: { type: Boolean, default: false }
})

const emit = defineEmits(['logout', 'open-copilot', 'open-navigation'])
</script>

<template>
  <header class="topbar">
    <div>
      <span class="breadcrumb">{{ breadcrumb }}</span>
      <h1>{{ title }}</h1>
      <p>{{ subtitle }}</p>
    </div>
    <div class="top-actions">
      <span v-if="error" class="error">{{ error }}</span>
      <span v-if="loading" class="muted">加载中...</span>
      <template v-if="authenticated">
        <button class="mobile-nav-toggle" type="button" title="打开导航" aria-label="打开导航" @click="emit('open-navigation')">
          <SvgIcon name="menu" />
        </button>
        <button class="topbar-copilot" title="打开 AI Copilot" aria-label="打开 AI Copilot" @click="emit('open-copilot')">
          <img src="/copilot-root-cause-icon.png" alt="" />
        </button>
        <span class="user-pill">{{ username }}</span>
        <button @click="emit('logout')">退出</button>
      </template>
    </div>
  </header>
</template>
