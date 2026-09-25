import { ref } from 'vue'
import { api } from '../api'
import { loadCopilotMessages, saveCopilotMessages } from '../copilot-chat-storage'

export function useCopilotChat() {
  const open = ref(false)
  const expanded = ref(false)
  const question = ref('')
  const busy = ref(false)
  const messages = ref(loadCopilotMessages(''))
  let userID = ''

  function setUserID(id) {
    const nextUserID = id ? String(id) : ''
    if (nextUserID === userID) return
    userID = nextUserID
    question.value = ''
    busy.value = false
    messages.value = loadCopilotMessages(userID)
  }

  function persistMessages() {
    saveCopilotMessages(userID, messages.value)
  }

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
    const ownerID = userID
    messages.value.push({ role: 'user', text })
    question.value = ''
    const pending = { role: 'ai', text: '正在分析授权上下文…', pending: true }
    messages.value.push(pending)
    busy.value = true
    persistMessages()
    try {
      const result = await api('/copilot/chat', { method: 'POST', body: JSON.stringify({ question: text }) })
      if (ownerID !== userID) return
      Object.assign(pending, { text: result.answer, pending: false, provider: result.provider, model: result.model })
    } catch (err) {
      if (ownerID !== userID) return
      Object.assign(pending, { role: 'error', text: `暂时无法完成分析：${err.message}`, pending: false })
    } finally {
      if (ownerID === userID) {
        busy.value = false
        persistMessages()
      }
    }
  }

  return { open, expanded, question, messages, busy, hide, toggleSize, setUserID, send }
}
