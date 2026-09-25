<script setup>
import { nextTick, ref, watch } from 'vue'
import SvgIcon from './SvgIcon.vue'
import CopilotMessage from './CopilotMessage.vue'

const props = defineProps({
  open: { type: Boolean, default: false },
  expanded: { type: Boolean, default: false },
  busy: { type: Boolean, default: false },
  messages: { type: Array, default: () => [] },
  question: { type: String, default: '' }
})

const emit = defineEmits(['hide', 'toggle-size', 'send', 'update:question'])
const questionInput = ref(null)
const messagesBody = ref(null)
let opener = null

watch(() => props.open, async (open) => {
  if (!open) {
    await nextTick()
    opener?.focus?.()
    opener = null
    return
  }
  opener = document.activeElement
  await nextTick()
  questionInput.value?.focus()
})

watch(() => props.messages, async () => {
  await nextTick()
  const body = messagesBody.value
  if (body) body.scrollTop = body.scrollHeight
}, { deep: true, flush: 'post' })
</script>

<template>
  <section v-if="open" class="copilot" :class="{ expanded }" role="dialog" aria-label="OpsCore AI Copilot" @keydown.esc="emit('hide')">
    <header>
      <div class="copilot-title">
        <span class="copilot-logo"><img class="copilot-icon-img" src="/copilot-root-cause-icon.png" alt="" /></span>
        <div>
          <strong>OpsCore AI Copilot</strong>
          <small>资产、事件、值班与任务助手</small>
        </div>
      </div>
      <div class="copilot-actions">
        <button :title="expanded ? '还原窗口' : '放大窗口'" :aria-label="expanded ? '还原 Copilot 窗口' : '放大 Copilot 窗口'" @click="emit('toggle-size')">
          <SvgIcon :name="expanded ? 'restore' : 'maximize'" />
        </button>
        <button title="隐藏 Copilot" aria-label="隐藏 Copilot" @click="emit('hide')">
          <SvgIcon name="minimize" />
        </button>
      </div>
    </header>
    <div ref="messagesBody" class="copilot-body" aria-live="polite">
      <div v-for="(message, index) in messages" :key="index" :class="['chat', message.role, { pending: message.pending }]">
        <CopilotMessage v-if="message.role === 'ai' && !message.pending" :text="message.text" />
        <span v-else>{{ message.text }}</span>
      </div>
    </div>
    <div class="copilot-input">
      <input
        ref="questionInput"
        :value="question"
        :disabled="busy"
        aria-label="向 AI Copilot 提问"
        placeholder="输入问题，例如：查询支付服务关联资产"
        @input="emit('update:question', $event.target.value)"
        @keyup.enter="emit('send')"
      />
      <button class="primary" :disabled="busy || !question.trim()" @click="emit('send')">{{ busy ? '分析中' : '发送' }}</button>
    </div>
  </section>
</template>
