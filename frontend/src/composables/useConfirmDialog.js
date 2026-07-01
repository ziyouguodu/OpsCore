import { nextTick, reactive } from 'vue'

export function useConfirmDialog(onError) {
  const confirmState = reactive({
    open: false,
    title: '',
    message: '',
    target: '',
    confirmLabel: '确认删除',
    busy: false,
    action: null,
    resolver: null,
    trigger: null
  })

  function resetConfirmState() {
    const trigger = confirmState.trigger
    confirmState.open = false
    confirmState.title = ''
    confirmState.message = ''
    confirmState.target = ''
    confirmState.confirmLabel = '确认删除'
    confirmState.busy = false
    confirmState.action = null
    confirmState.resolver = null
    confirmState.trigger = null
    nextTick(() => trigger?.focus?.())
  }

  function cancelConfirm() {
    if (confirmState.busy) return
    confirmState.resolver?.(false)
    resetConfirmState()
  }

  function requestConfirm(options, action) {
    if (confirmState.open) {
      cancelConfirm()
    }
    confirmState.title = options.title
    confirmState.message = options.message
    confirmState.target = options.target || ''
    confirmState.confirmLabel = options.confirmLabel || '确认删除'
    confirmState.action = action
    confirmState.trigger = document.activeElement
    confirmState.open = true
    return new Promise((resolve) => {
      confirmState.resolver = resolve
    })
  }

  async function acceptConfirm() {
    if (confirmState.busy) return
    confirmState.busy = true
    try {
      await confirmState.action?.()
      confirmState.resolver?.(true)
    } catch (err) {
      onError?.(err)
      confirmState.resolver?.(false)
    } finally {
      resetConfirmState()
    }
  }

  return {
    confirmState,
    requestConfirm,
    cancelConfirm,
    acceptConfirm
  }
}
