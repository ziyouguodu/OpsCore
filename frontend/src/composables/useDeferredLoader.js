import { onUnmounted } from 'vue'

export function useDeferredLoader(isEnabled, onError) {
  const timers = new Map()

  function schedule(loader, label, delay = 0) {
    if (!isEnabled()) return
    const existing = timers.get(loader)
    if (existing) window.clearTimeout(existing)
    const timer = window.setTimeout(async () => {
      timers.delete(loader)
      try {
        await loader()
      } catch (err) {
        onError(`${label}加载失败：${err.message}`)
      }
    }, delay)
    timers.set(loader, timer)
  }

  onUnmounted(() => {
    for (const timer of timers.values()) window.clearTimeout(timer)
    timers.clear()
  })

  return { schedule }
}
