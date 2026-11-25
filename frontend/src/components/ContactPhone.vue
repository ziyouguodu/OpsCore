<script setup>
import { computed, ref, watch } from 'vue'
import { phoneDialValue } from '../contact-phone'

const props = defineProps({ phone: { type: String, default: '' }, name: { type: String, default: '' } })
const dial = computed(() => phoneDialValue(props.phone))
const message = ref('')
watch(() => props.phone, () => { message.value = '' })

async function copy() {
  message.value = ''
  const phone = props.phone
  try {
    await navigator.clipboard.writeText(phone)
    if (props.phone === phone) message.value = '号码已复制'
  } catch {
    if (props.phone === phone) message.value = '复制失败，请手动选择号码复制'
  }
}
</script>

<template>
  <div class="contact-phone">
    <template v-if="dial">
      <span class="contact-number">{{ phone }}</span>
      <span class="contact-actions"><button type="button" :aria-label="`复制${name}联系电话`" @click="copy">复制</button><a :href="`tel:${dial}`" :aria-label="`拨打${name}联系电话`">拨打</a></span>
    </template>
    <span v-else class="muted">未配置联系方式</span>
    <span v-if="message" class="contact-feedback" role="status" aria-live="polite">{{ message }}</span>
  </div>
</template>
