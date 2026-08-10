import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { permissionTabs, routeViews } from '../navigation'

function parseRouteHash() {
  const raw = window.location.hash.replace(/^#/, '')
  const [view, tab] = raw.split('/')
  return {
    view: routeViews.includes(view) ? view : 'dashboard',
    permissionTab: permissionTabs.includes(tab) ? tab : 'users'
  }
}

export function useWorkspaceRoute({ beforeNavigate = () => {}, afterNavigate = () => {} } = {}) {
  const initial = parseRouteHash()
  const activeView = ref(initial.view)
  const permissionTab = ref(initial.permissionTab)

  function routeHash(view = activeView.value, tab = permissionTab.value) {
    return view === 'permissions' ? `#${view}/${tab}` : `#${view}`
  }

  function resetScroll() {
    nextTick(() => window.scrollTo({ top: 0, left: 0, behavior: 'auto' }))
  }

  function applyRoute(view, tab, clearFeedback = false, syncTab = false) {
    const nextTab = permissionTabs.includes(tab) ? tab : permissionTab.value
    const changed = activeView.value !== view || (view === 'permissions' && permissionTab.value !== nextTab)
    if (changed) beforeNavigate()
    activeView.value = routeViews.includes(view) ? view : 'dashboard'
    if (activeView.value === 'permissions' || syncTab) permissionTab.value = nextTab
    afterNavigate({ clearFeedback })
    resetScroll()
  }

  function syncFromHash() {
    const next = parseRouteHash()
    applyRoute(next.view, next.permissionTab, false, true)
  }

  function goToView(view, tab) {
    applyRoute(view, tab, true)
  }

  watch([activeView, permissionTab], () => {
    const nextHash = routeHash()
    if (window.location.hash !== nextHash) window.history.replaceState(null, '', nextHash)
  })

  onMounted(() => window.addEventListener('hashchange', syncFromHash))
  onUnmounted(() => window.removeEventListener('hashchange', syncFromHash))

  return { activeView, permissionTab, goToView }
}
