import { onUnmounted, ref } from 'vue'

export function useToast() {
  const messages = ref([])
  const timers = new Map()
  let nextID = 1

  function dismiss(id) {
    const timer = timers.get(id)
    if (timer) window.clearTimeout(timer)
    timers.delete(id)
    messages.value = messages.value.filter((item) => item.id !== id)
  }

  function notify(message, type = 'success', duration = 3600) {
    const id = nextID++
    messages.value.push({ id, message, type })
    if (duration > 0) timers.set(id, window.setTimeout(() => dismiss(id), duration))
    return id
  }

  onUnmounted(() => {
    for (const timer of timers.values()) window.clearTimeout(timer)
    timers.clear()
  })

  return { messages, notify, dismiss }
}
