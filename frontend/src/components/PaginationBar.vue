<script setup>
import { computed } from 'vue'

const props = defineProps({
  total: { type: Number, default: 0 },
  page: { type: Number, default: 1 },
  pageSize: { type: Number, default: 10 },
  pageCount: { type: Number, default: 1 }
})

const emit = defineEmits(['update:page', 'update:page-size'])

const normalizedPageCount = computed(() => Math.max(1, props.pageCount))
const visiblePages = computed(() => {
  const total = normalizedPageCount.value
  if (total <= 7) return Array.from({ length: total }, (_, index) => index + 1)

  const pages = new Set([1, total])
  for (let page = Math.max(2, props.page - 1); page <= Math.min(total - 1, props.page + 1); page += 1) {
    pages.add(page)
  }
  const sorted = [...pages].sort((a, b) => a - b)
  const items = []
  sorted.forEach((page, index) => {
    if (index > 0 && page - sorted[index - 1] > 1) items.push(`gap-${page}`)
    items.push(page)
  })
  return items
})

function goToPage(page) {
  const next = Math.min(Math.max(1, page), normalizedPageCount.value)
  if (next !== props.page) emit('update:page', next)
}

function changePageSize(event) {
  emit('update:page-size', Number(event.target.value))
  emit('update:page', 1)
}
</script>

<template>
  <nav class="pager" aria-label="分页">
    <span>共 {{ total }} 条，每页显示：</span>
    <select :value="pageSize" aria-label="每页显示条数" @change="changePageSize">
      <option :value="10">10 条/页</option>
      <option :value="20">20 条/页</option>
      <option :value="50">50 条/页</option>
    </select>
    <button type="button" aria-label="上一页" :disabled="page <= 1" @click="goToPage(page - 1)">‹</button>
    <template v-for="item in visiblePages" :key="item">
      <span v-if="typeof item === 'string'" class="pager-gap" aria-hidden="true">…</span>
      <button
        v-else
        type="button"
        :aria-label="`第 ${item} 页`"
        :aria-current="item === page ? 'page' : undefined"
        :class="{ active: item === page }"
        @click="goToPage(item)"
      >{{ item }}</button>
    </template>
    <button type="button" aria-label="下一页" :disabled="page >= normalizedPageCount" @click="goToPage(page + 1)">›</button>
  </nav>
</template>
