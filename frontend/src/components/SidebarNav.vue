<script setup>
import { menu } from '../navigation'
import SvgIcon from './SvgIcon.vue'

defineProps({
  activeView: { type: String, required: true },
  permissionTab: { type: String, required: true },
  collapsed: { type: Boolean, default: false },
  mobileOpen: { type: Boolean, default: false }
})

const emit = defineEmits(['toggle', 'navigate'])
</script>

<template>
  <aside class="sidebar" :class="{ 'mobile-open': mobileOpen }">
    <div class="brand">
      <img src="/opscore-ai-ops-logo.png" alt="OpsCore logo" />
      <span>OpsCore</span>
      <button class="sidebar-toggle" type="button" :aria-label="collapsed ? '展开菜单' : '收起菜单'" @click="emit('toggle')">
        {{ collapsed ? '>' : '<' }}
      </button>
    </div>
    <div class="side-section">工作台</div>
    <button class="nav-row" :class="{ active: activeView === 'dashboard' }" title="首页仪表盘" @click="emit('navigate', 'dashboard')">
      <span class="nav-icon"><SvgIcon name="dashboard" /></span>
      <span class="nav-label">首页仪表盘</span>
    </button>

    <div class="side-section">功能模块</div>
    <div v-for="group in menu.slice(1)" :key="group.label" class="nav-group" :class="{ disabled: group.enabled === false }">
      <div class="nav-parent" :title="group.label">
        <span class="nav-icon"><SvgIcon :name="group.icon" /></span>
        <strong>{{ group.label }}</strong>
        <span class="nav-state">{{ group.enabled === false ? '灰度' : '展开' }}</span>
      </div>
      <div v-if="group.children" class="nav-children">
        <button
          v-for="child in group.children"
          :key="`${group.label}-${child.label}`"
          class="nav-child"
          :class="{ active: activeView === child.id && (!child.permissionTab || permissionTab === child.permissionTab), disabled: !child.enabled }"
          :disabled="!child.enabled"
          :title="child.label"
          @click="emit('navigate', child.id, child.permissionTab)"
        >
          <span class="child-icon">
            <img v-if="child.icon === 'copilot'" class="copilot-icon-img" src="/copilot-root-cause-icon.png" alt="" />
            <SvgIcon v-else :name="child.icon" />
          </span>
          <span>{{ child.label }}</span>
          <em>{{ child.enabled ? '一期' : '灰度' }}</em>
        </button>
      </div>
    </div>
  </aside>
</template>
