const STORAGE_KEY_PREFIX = 'opscore.copilot.messages.'
const MAX_MESSAGES = 60
const WELCOME_MESSAGE = {
  role: 'ai',
  text: '请描述需要分析的运维问题，我会基于当前账号可访问的数据给出证据和建议。'
}

export function loadCopilotMessages(userID, storage = sessionStorage) {
  if (!userID) return [{ ...WELCOME_MESSAGE }]

  try {
    const saved = JSON.parse(storage.getItem(`${STORAGE_KEY_PREFIX}${userID}`) || 'null')
    if (!Array.isArray(saved)) return [{ ...WELCOME_MESSAGE }]

    const messages = saved
      .filter((message) => message && ['ai', 'user', 'error'].includes(message.role) && typeof message.text === 'string' && !message.pending)
      .slice(-MAX_MESSAGES)

    return messages.length ? messages : [{ ...WELCOME_MESSAGE }]
  } catch {
    return [{ ...WELCOME_MESSAGE }]
  }
}

export function saveCopilotMessages(userID, messages, storage = sessionStorage) {
  if (!userID) return

  const saved = messages
    .filter((message) => message && ['ai', 'user', 'error'].includes(message.role) && typeof message.text === 'string' && !message.pending)
    .slice(-MAX_MESSAGES)
    .map(({ role, text }) => ({ role, text }))

  try {
    storage.setItem(`${STORAGE_KEY_PREFIX}${userID}`, JSON.stringify(saved))
  } catch {
    // Storage can be unavailable or full; chat must remain usable in memory.
  }
}
