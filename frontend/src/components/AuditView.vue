<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  canManage: { type: Boolean, default: false },
  events: { type: Array, default: () => [] }
})

defineEmits(['refresh'])

const keyword = ref('')
const outcome = ref('')

const filteredEvents = computed(() => {
  const query = keyword.value.trim().toLowerCase()
  return props.events.filter(event => {
    const matchesOutcome = !outcome.value || event.outcome === outcome.value
    const haystack = [event.actorUsername, event.action, event.resourceType, event.resourceId, event.ipAddress]
      .join(' ')
      .toLowerCase()
    return matchesOutcome && (!query || haystack.includes(query))
  })
})

function formatTime(value) {
  if (!value) return '-'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false
  }).format(new Date(value))
}
</script>

<template>
  <section class="panel audit-workspace" role="region" aria-label="操作审计工作区">
    <div v-if="!canManage" class="empty-state">
      <strong>当前角色无权查看操作审计</strong>
      <span>操作审计仅对超级管理员开放。</span>
    </div>
    <template v-else>
      <div class="query-bar audit-query">
        <input v-model="keyword" placeholder="搜索账号、动作、资源或 IP" aria-label="搜索审计记录" />
        <select v-model="outcome" aria-label="筛选操作结果">
          <option value="">全部结果</option>
          <option value="success">成功</option>
          <option value="failure">失败</option>
        </select>
        <button type="button" @click="$emit('refresh')">刷新</button>
        <span class="muted">最近 {{ filteredEvents.length }} 条</span>
      </div>

      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>时间</th>
              <th>操作人</th>
              <th>动作</th>
              <th>资源</th>
              <th>结果</th>
              <th>来源 IP</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="event in filteredEvents" :key="event.id">
              <td>{{ formatTime(event.createdAt) }}</td>
              <td>{{ event.actorUsername || '匿名请求' }}</td>
              <td><code>{{ event.action }}</code></td>
              <td>{{ event.resourceType }}<template v-if="event.resourceId"> / {{ event.resourceId }}</template></td>
              <td><span :class="['pill', event.outcome === 'success' ? 'success' : 'danger']">{{ event.outcome === 'success' ? '成功' : '失败' }}</span></td>
              <td>{{ event.ipAddress || '-' }}</td>
            </tr>
            <tr v-if="!filteredEvents.length">
              <td colspan="6"><div class="empty-state compact">暂无符合条件的审计记录</div></td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </section>
</template>
