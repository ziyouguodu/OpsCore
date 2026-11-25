<script setup>
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'

const props = defineProps({ open: Boolean, busy: Boolean })
const emit = defineEmits(['close'])
const dialog = ref(null)
let returnFocus

function close(event) {
  event?.preventDefault()
  if (!props.busy) emit('close')
}

watch(() => props.open, async open => {
  if (open) {
    returnFocus = document.activeElement
    await nextTick()
    dialog.value?.showModal()
  } else {
    dialog.value?.close()
    if (returnFocus?.isConnected) returnFocus.focus()
  }
})
onBeforeUnmount(() => dialog.value?.close())
</script>

<template>
  <dialog ref="dialog" class="duty-dialog" @cancel="close" @click="($event.target === dialog) && close($event)">
    <slot />
  </dialog>
</template>
