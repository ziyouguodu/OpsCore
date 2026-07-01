<script setup>
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'

const props = defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, default: '确认操作' },
  message: { type: String, default: '' },
  target: { type: String, default: '' },
  confirmLabel: { type: String, default: '确认删除' },
  busy: { type: Boolean, default: false }
})

const emit = defineEmits(['confirm', 'cancel'])
const confirmButton = ref(null)

function cancel() {
  if (props.busy) return
  emit('cancel')
}

function confirm() {
  if (props.busy) return
  emit('confirm')
}

function onKeydown(event) {
  if (event.key === 'Escape' && props.open && !props.busy) {
    event.preventDefault()
    cancel()
  }
}

watch(
  () => props.open,
  async isOpen => {
    if (isOpen) {
      window.addEventListener('keydown', onKeydown)
      await nextTick()
      confirmButton.value?.focus()
    } else {
      window.removeEventListener('keydown', onKeydown)
    }
  }
)

onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="confirm-backdrop" @click.self="cancel">
      <section
        class="confirm-dialog"
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="confirm-title"
        aria-describedby="confirm-description"
        data-testid="confirm-dialog"
      >
        <header class="confirm-head">
          <span class="confirm-icon" aria-hidden="true">!</span>
          <div>
            <p class="confirm-kicker">危险操作</p>
            <h2 id="confirm-title">{{ title }}</h2>
          </div>
          <button
            class="icon-button confirm-close"
            type="button"
            aria-label="关闭确认弹窗"
            :disabled="busy"
            @click="cancel"
          >
            ×
          </button>
        </header>

        <div class="confirm-body">
          <p id="confirm-description">{{ message }}</p>
          <div v-if="target" class="confirm-target">
            <span>操作对象</span>
            <strong>{{ target }}</strong>
          </div>
        </div>

        <footer class="confirm-actions">
          <button type="button" class="ghost-button" :disabled="busy" data-testid="confirm-cancel" @click="cancel">取消</button>
          <button
            ref="confirmButton"
            type="button"
            class="danger-button"
            :disabled="busy"
            data-testid="confirm-accept"
            @click="confirm"
          >
            {{ busy ? '处理中...' : confirmLabel }}
          </button>
        </footer>
      </section>
    </div>
  </Teleport>
</template>
