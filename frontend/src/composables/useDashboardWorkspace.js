import { computed, ref } from 'vue'
import { api } from '../api'
import { buildPreviewPath, emptyWorkspace, normalizePreview, normalizeWorkspace } from '../dashboard-workspace'

export function useDashboardWorkspace() {
  const workspace = ref(emptyWorkspace())
  const preview = ref(null)
  const previewEnabled = ref(false)
  const previewLoading = ref(false)
  const previewError = ref('')
  let sequence = 0
  let query = { period: 'today', system: 'all' }

  async function loadWorkspace() {
    workspace.value = normalizeWorkspace(await api('/dashboard/workspace'))
  }
  async function loadPreview(nextQuery = query) {
    query = nextQuery
    const request = ++sequence
    previewError.value = ''
    if (!previewEnabled.value) { preview.value = null; previewLoading.value = false; return }
    previewLoading.value = true
    try {
      const result = normalizePreview(await api(buildPreviewPath(query)))
      if (request === sequence) preview.value = result
    } catch (error) {
      if (request === sequence) previewError.value = error.message
    } finally { if (request === sequence) previewLoading.value = false }
  }
  function resetWorkspace() {
    sequence += 1
    query = { period: 'today', system: 'all' }
    workspace.value = emptyWorkspace()
    preview.value = null
    previewEnabled.value = false
    previewLoading.value = false
    previewError.value = ''
  }
  return { workspace, preview, previewEnabled, previewLoading, previewError, businessSystems: computed(() => workspace.value.businessSystems), loadWorkspace, loadPreview, resetWorkspace }
}
