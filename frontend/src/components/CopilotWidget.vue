<script setup>
import SvgIcon from './SvgIcon.vue'

defineProps({
  open: { type: Boolean, default: false },
  expanded: { type: Boolean, default: false },
  messages: { type: Array, default: () => [] },
  question: { type: String, default: '' }
})

const emit = defineEmits(['hide', 'toggle-size', 'send', 'update:question'])
</script>

<template>
  <section v-if="open" class="copilot" :class="{ expanded }">
    <header>
      <div class="copilot-title">
        <span class="copilot-logo"><img class="copilot-icon-img" src="/copilot-root-cause-icon.png" alt="" /></span>
        <div>
          <strong>OpsCore AI Copilot</strong>
          <small>资产、事件、值班与任务助手</small>
        </div>
      </div>
      <div class="copilot-actions">
        <button :title="expanded ? '还原窗口' : '放大窗口'" @click="emit('toggle-size')">
          <SvgIcon :name="expanded ? 'restore' : 'maximize'" />
        </button>
        <button title="隐藏 Copilot" @click="emit('hide')">
          <SvgIcon name="minimize" />
        </button>
      </div>
    </header>
    <div class="copilot-body">
      <p>我可以帮你查询 CMDB 资产、定位活跃事件、汇总今日值班和待处理任务。</p>
      <div v-for="(message, index) in messages" :key="index" :class="['chat', message.role]">{{ message.text }}</div>
    </div>
    <div class="copilot-input">
      <input
        :value="question"
        placeholder="输入问题，例如：查询支付服务关联资产"
        @input="emit('update:question', $event.target.value)"
        @keyup.enter="emit('send')"
      />
      <button class="primary" @click="emit('send')">发送</button>
    </div>
  </section>
</template>
