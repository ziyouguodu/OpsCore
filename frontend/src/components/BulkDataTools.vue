<script setup>
import { ref } from 'vue'

defineProps({
  canWrite: { type: Boolean, default: false },
  busy: { type: Boolean, default: false },
  canBulkDelete: { type: Boolean, default: false },
  label: { type: String, required: true },
  selectedCount: { type: Number, default: 0 }
})

const emit = defineEmits(['import-file', 'export-selected', 'delete-selected', 'clear-selection', 'download-template'])
const fileInput = ref(null)

function onFileChange(event) {
  const file = event.target.files?.[0]
  if (file) emit('import-file', file)
  event.target.value = ''
}
</script>

<template>
  <div class="page-actions bulk-data-tools">
    <input ref="fileInput" class="visually-hidden" type="file" accept=".csv,text/csv" :aria-label="`导入${label} CSV 文件`" @change="onFileChange" />
    <span class="selection-summary">已选 {{ selectedCount }} 项</span>
    <button :disabled="busy || selectedCount === 0" @click="$emit('export-selected')"><span class="btn-icon">↧</span>导出已选</button>
    <button v-if="selectedCount" class="danger-action" :disabled="busy || !canBulkDelete" :title="canBulkDelete ? '删除所有已选择项' : '所选记录包含无删除权限的项'" @click="$emit('delete-selected')">批量删除</button>
    <button v-if="selectedCount" class="link" :disabled="busy" @click="$emit('clear-selection')">清除选择</button>
    <button :disabled="!canWrite || busy" @click="fileInput?.click()"><span class="btn-icon">↥</span>{{ busy ? '处理中…' : '批量导入' }}</button>
    <button :disabled="busy" @click="$emit('download-template')">下载导入模板</button>
  </div>
</template>
