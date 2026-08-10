import { ref } from 'vue'
import { api } from '../api'

export function useCopilotChat() {
  const open = ref(false)
  const expanded = ref(false)
  const question = ref('')
  const busy = ref(false)
  const messages = ref([
    { role: 'ai', text: '请描述需要分析的运维问题，我会基于当前账号可访问的数据给出证据和建议。' }
  ])

  function hide() {
    open.value = false
    expanded.value = false
  }

  function toggleSize() {
    expanded.value = !expanded.value
  }

  async function send() {
    const text = question.value.trim()
    if (!text || busy.value) return
    messages.value.push({ role: 'user', text })
    question.value = ''
    const pending = { role: 'ai', text: '正在分析授权上下文…', pending: true }
    messages.value.push(pending)
    busy.value = true
    try {
      const result = await api('/copilot/chat', { method: 'POST', body: JSON.stringify({ question: text }) })
      Object.assign(pending, { text: result.answer, pending: false, provider: result.provider, model: result.model })
    } catch (err) {
      Object.assign(pending, { role: 'error', text: `暂时无法完成分析：${err.message}`, pending: false })
    } finally {
      busy.value = false
    }
  }

  return { open, expanded, question, messages, busy, hide, toggleSize, send }
}
