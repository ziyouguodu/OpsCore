<script setup>
defineProps({ messages: { type: Array, default: () => [] } })
const emit = defineEmits(['dismiss'])
</script>

<template>
  <div class="toast-stack" aria-live="polite" aria-atomic="true">
    <div v-for="item in messages" :key="item.id" :class="['toast-message', item.type]" role="status">
      <span>{{ item.message }}</span>
      <button type="button" aria-label="关闭提示" @click="emit('dismiss', item.id)">×</button>
    </div>
  </div>
</template>
