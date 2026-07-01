<script setup>
import PaginationBar from './PaginationBar.vue'

defineProps({
  activeTaskCount: { type: Number, default: 0 },
  currentTask: { type: Object, default: null },
  form: { type: Object, required: true },
  formOpen: { type: Boolean, default: false },
  isSample: { type: Function, required: true },
  page: { type: Number, required: true },
  pageCount: { type: Number, required: true },
  pageSize: { type: Number, required: true },
  pagedTasks: { type: Array, default: () => [] },
  statusCounts: { type: Object, default: () => ({}) },
  total: { type: Number, default: 0 }
})

const emit = defineEmits([
  'choose',
  'close-form',
  'delete',
  'edit',
  'hide-detail',
  'open-form',
  'save',
  'update-status',
  'update:page',
  'update:pageSize'
])

function closeEditorOnFocusOut(event) {
  const nextTarget = event.relatedTarget
  if (!nextTarget || !event.currentTarget.contains(nextTarget)) {
    emit('close-form')
  }
}
</script>

<template>
  <section class="panel" role="region" aria-label="任务跟踪工作区">
    <div class="section-head actions-only">
      <div class="page-actions">
        <span class="pill">未关闭 {{ activeTaskCount }}</span>
        <button class="primary" @click="$emit('open-form')"><span class="btn-icon">＋</span>创建任务</button>
      </div>
    </div>

    <div class="status-grid task-status">
      <article v-for="status in ['待处理','处理中','待确认','已完成','已关闭']" :key="status" :class="{ active: status === '处理中' }">
        <small>{{ status }}</small>
        <strong>{{ statusCounts[status] || 0 }}</strong>
        <span>{{ status === '待确认' ? '等待发起人确认' : status === '已关闭' ? '归档记录' : '任务状态' }}</span>
      </article>
    </div>

    <section v-if="formOpen" class="editor-panel" tabindex="-1" @focusout="closeEditorOnFocusOut">
      <h3>{{ form.id ? '编辑任务' : '创建任务' }}</h3>
      <div class="form-grid">
        <input v-model="form.title" placeholder="任务标题" />
        <input v-model="form.assignee" placeholder="负责人" />
        <input v-model="form.dueAt" placeholder="截止时间" />
        <select v-model="form.status"><option>待处理</option><option>处理中</option><option>待确认</option><option>已完成</option><option>已关闭</option></select>
        <textarea v-model="form.description" class="form-wide" rows="4" placeholder="任务说明：描述背景、处理要求、关联资产或验收标准"></textarea>
        <button class="primary" @click="$emit('save')">{{ form.id ? '保存修改' : '创建任务' }}</button>
        <button @click="$emit('close-form')">取消</button>
      </div>
    </section>

    <div class="work-layout" :class="{ 'single-column': !currentTask }">
      <section>
        <div class="table-wrap">
          <table>
            <thead><tr><th>标题</th><th>负责人</th><th>状态</th><th>截止时间</th><th>操作</th></tr></thead>
            <tbody>
              <tr v-for="task in pagedTasks" :key="task.id" :class="{ selected: currentTask?.id === task.id }" class="clickable-row" tabindex="0" @click="$emit('choose', task)" @keyup.enter="$emit('choose', task)">
                <td>{{ task.title }}</td>
                <td>{{ task.assignee || '-' }}</td>
                <td><span class="pill">{{ task.status }}</span></td>
                <td>{{ task.dueAt || '-' }}</td>
                <td class="row-actions"><button class="link" @click.stop="$emit('choose', task)">详情</button><button class="link" :disabled="isSample(task)" @click.stop="$emit('edit', task)">编辑</button><button class="link danger-text" :disabled="isSample(task)" @click.stop="$emit('delete', task)">删除</button></td>
              </tr>
              <tr v-if="!pagedTasks.length"><td colspan="5" class="empty">暂无任务记录</td></tr>
            </tbody>
          </table>
        </div>
        <PaginationBar
          :page="page"
          :page-size="pageSize"
          :total="total"
          :page-count="pageCount"
          @update:page="$emit('update:page', $event)"
          @update:page-size="$emit('update:pageSize', $event)"
        />
      </section>

      <aside v-if="currentTask" class="detail-card">
        <div class="detail-title">
          <h3>{{ currentTask.title }}</h3>
          <button class="detail-close" aria-label="隐藏任务详情" @click="$emit('hide-detail')">隐藏</button>
        </div>
        <dl>
          <dt>负责人</dt><dd>{{ currentTask.assignee || '-' }}</dd>
          <dt>当前状态</dt><dd>{{ currentTask.status }}</dd>
          <dt>截止时间</dt><dd>{{ currentTask.dueAt || '-' }}</dd>
          <dt>说明</dt><dd>{{ currentTask.description || '暂无说明' }}</dd>
        </dl>
        <label>状态流转
          <select :value="currentTask.status" :disabled="isSample(currentTask)" @change="$emit('update-status', currentTask, $event.target.value)">
            <option>待处理</option><option>处理中</option><option>待确认</option><option>已完成</option><option>已关闭</option>
          </select>
        </label>
        <div class="detail-actions">
          <button class="primary" :disabled="isSample(currentTask)" @click="$emit('edit', currentTask)">编辑任务</button>
          <button class="danger-action" :disabled="isSample(currentTask)" @click="$emit('delete', currentTask)">删除任务</button>
        </div>
        <ol class="flow compact"><li>待处理</li><li>处理中</li><li>待确认</li><li>已完成</li><li>已关闭</li></ol>
      </aside>
    </div>
  </section>
</template>
