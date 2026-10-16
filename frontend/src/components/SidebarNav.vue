<script setup>
import { nextTick, ref, watch } from 'vue'
import { menu } from '../navigation'
import SvgIcon from './SvgIcon.vue'

const props = defineProps({
  activeView: { type: String, required: true },
  permissionTab: { type: String, required: true },
  mobileOpen: { type: Boolean, default: false }
})
const emit = defineEmits(['navigate', 'open-copilot', 'close'])
const sidebar = ref(null)
let returnFocus
const expanded = ref(Object.fromEntries(menu.filter(item => item.children).map(item => [item.label, true])))
function isActive(item) {
  return item.id === props.activeView && (!item.permissionTab || item.permissionTab === props.permissionTab)
}
function groupActive(group) { return group.children?.some(isActive) }
function toggleGroup(group) { expanded.value[group.label] = !expanded.value[group.label] }
watch(() => [props.activeView, props.permissionTab], () => {
  const group = menu.find(item => item.children?.some(isActive))
  if (group) expanded.value[group.label] = true
})
function activate(item) {
  if (item.action === 'copilot') emit('open-copilot')
  else emit('navigate', item.id, item.permissionTab)
  emit('close')
}
function trapFocus(event) {
  if (!props.mobileOpen) return
  if (event.key === 'Escape') emit('close')
  if (event.key !== 'Tab') return
  const buttons = [...sidebar.value.querySelectorAll('button:not(:disabled)')].filter(button => button.getClientRects().length)
  const first = buttons[0], last = buttons.at(-1)
  if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus() }
  else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus() }
}
watch(() => props.mobileOpen, async (open) => {
  if (open) {
    returnFocus = document.activeElement
    await nextTick()
    sidebar.value.querySelector('button').focus()
  } else if (returnFocus?.isConnected) returnFocus.focus()
})
</script>

<template>
  <aside id="workspace-navigation" ref="sidebar" class="sidebar review-sidebar" :class="{ 'mobile-open': mobileOpen }" aria-label="主导航" @keydown="trapFocus">
    <button class="drawer-close" type="button" aria-label="关闭导航" @click="emit('close')">×</button>
    <nav>
      <template v-for="(item, index) in menu" :key="item.label">
        <template v-if="item.children">
          <button class="nav nav-group-toggle" type="button" :aria-expanded="expanded[item.label]" :aria-controls="`navigation-group-${index}`" :class="{ 'contains-active': groupActive(item) }" @click="toggleGroup(item)">
            <SvgIcon class="icon" :name="item.icon" /><span>{{ item.label }}</span><span class="nav-chevron" :class="{ expanded: expanded[item.label] }" aria-hidden="true">›</span>
          </button>
          <div :id="`navigation-group-${index}`" v-show="expanded[item.label]" class="nav-submenu">
            <button v-for="child in item.children" :key="child.label" class="nav child nav-child" :class="{ active: isActive(child) }" type="button" :disabled="child.enabled === false" :aria-current="isActive(child) ? 'page' : undefined" @click="activate(child)">
              <SvgIcon class="icon micro-icon" :name="child.icon" /><span>{{ child.label }}</span><small v-if="child.enabled === false">待接入</small>
            </button>
          </div>
        </template>
        <button v-else class="nav" :class="{ active: isActive(item), 'nav-row': item.id === 'dashboard' }" type="button" :title="item.label" :disabled="item.enabled === false" :aria-current="isActive(item) ? 'page' : undefined" @click="activate(item)">
          <SvgIcon class="icon" :name="item.icon" /><span>{{ item.label }}</span><small v-if="item.enabled === false">待接入</small>
        </button>
      </template>
    </nav>
    <div class="nav-note">统一运维工作台<br>受控操作 · 权限与审计</div>
  </aside>
</template>
