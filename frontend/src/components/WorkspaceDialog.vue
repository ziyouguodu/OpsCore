<script setup>
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'

const props = defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, required: true }
})
const emit = defineEmits(['update:open'])
const element = ref(null)
let returnFocus

function close() {
  element.value?.close()
}

function onClose() {
  emit('update:open', false)
  if (returnFocus?.isConnected) returnFocus.focus()
}

watch(() => props.open, async (open) => {
  if (!open) {
    close()
    return
  }
  returnFocus = document.activeElement
  await nextTick()
  element.value?.showModal()
})
onBeforeUnmount(close)
</script>

<template>
  <dialog ref="element" class="review-dialog" :aria-label="title" @close="onClose" @click="($event.target === element) && close()">
    <div class="head"><h2>{{ title }}</h2><button type="button" aria-label="关闭弹窗" @click="close">×</button></div>
    <div class="dialog-body"><slot /></div>
    <div class="dialog-actions"><button type="button" @click="close">取消</button><slot name="actions" :close="close" /></div>
  </dialog>
</template>
