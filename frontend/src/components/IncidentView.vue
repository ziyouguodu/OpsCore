<script setup>
import PaginationBar from './PaginationBar.vue'
import { formatDateTime } from '../date-time'
import { incidentStatusOptions } from '../workflow-status'

defineProps({
  activeIncidentCount: { type: Number, default: 0 },
  currentIncident: { type: Object, default: null },
  form: { type: Object, required: true },
  formOpen: { type: Boolean, default: false },
  isSample: { type: Function, required: true },
  levelCounts: { type: Object, default: () => ({}) },
  page: { type: Number, required: true },
  pageCount: { type: Number, required: true },
  pageSize: { type: Number, required: true },
  pagedIncidents: { type: Array, default: () => [] },
  total: { type: Number, default: 0 },
  users: { type: Array, default: () => [] }
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

function severityLabel(level) {
  return level === 'P1' ? '高危' : level === 'P2' ? '重要' : level === 'P3' ? '一般' : '观察'
}
</script>

<template>
  <section class="panel" role="region" aria-label="事件管理工作区">
    <div class="section-head actions-only">
      <div class="page-actions">
        <span class="pill danger">活跃事件 {{ activeIncidentCount }}</span>
        <button class="primary" @click="$emit('open-form')"><span class="btn-icon">＋</span>新建事件</button>
      </div>
    </div>

    <div class="status-grid severity-grid">
      <article v-for="level in ['P1','P2','P3','P4']" :key="level" :class="level.toLowerCase()">
        <small>{{ level }}</small>
        <strong>{{ levelCounts[level] || 0 }}</strong>
        <span>{{ severityLabel(level) }}</span>
      </article>
    </div>

    <section v-if="formOpen" class="editor-panel" tabindex="-1" @focusout="closeEditorOnFocusOut">
      <h3>{{ form.id ? '编辑事件' : '新建事件' }}</h3>
      <div class="form-grid">
        <input v-model="form.title" aria-label="事件标题" placeholder="事件标题" />
        <select v-model="form.level" aria-label="事件等级"><option>P1</option><option>P2</option><option>P3</option><option>P4</option></select>
        <select v-model="form.ownerUserId" aria-label="事件负责人">
          <option value="">未指定负责人</option>
          <option v-for="user in users" :key="user.id" :value="user.id">{{ user.displayName }}（{{ user.username }}）</option>
        </select>
        <input v-model="form.business" aria-label="事件所属业务" placeholder="所属业务" />
        <select v-model="form.status" aria-label="事件状态" disabled><option>{{ form.status }}</option></select>
        <input v-model="form.startedAt" type="datetime-local" aria-label="事件开始时间" />
        <input v-model="form.recoveredAt" type="datetime-local" aria-label="事件恢复时间" />
        <textarea v-model="form.summary" class="form-wide" rows="4" aria-label="事件摘要" placeholder="事件摘要：描述影响范围、初步原因、当前处置动作和下一步计划"></textarea>
        <button class="primary" @click="$emit('save')">{{ form.id ? '保存修改' : '创建事件' }}</button>
        <button @click="$emit('close-form')">取消</button>
      </div>
    </section>

    <div class="work-layout" :class="{ 'single-column': !currentIncident }">
      <section>
        <div class="table-wrap">
          <table>
            <thead><tr><th>事件</th><th>等级</th><th>状态</th><th>负责人</th><th>业务</th><th>操作</th></tr></thead>
            <tbody>
              <tr v-for="incident in pagedIncidents" :key="incident.id" :class="{ selected: currentIncident?.id === incident.id }" class="clickable-row" tabindex="0" @click="$emit('choose', incident)" @keyup.enter="$emit('choose', incident)" @keyup.space.prevent="$emit('choose', incident)">
                <td>{{ incident.title }}</td>
                <td><span class="pill danger">{{ incident.level }}</span></td>
                <td><span class="pill">{{ incident.status }}</span></td>
                <td>{{ incident.owner || '-' }}</td>
                <td>{{ incident.business || '-' }}</td>
                <td class="row-actions"><button class="link" @click.stop="$emit('choose', incident)">详情</button><button class="link" :disabled="isSample(incident)" @click.stop="$emit('edit', incident)">编辑</button><button class="link danger-text" :disabled="isSample(incident)" @click.stop="$emit('delete', incident)">删除</button></td>
              </tr>
              <tr v-if="!pagedIncidents.length"><td colspan="6" class="empty">暂无事件记录</td></tr>
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

      <aside v-if="currentIncident" class="detail-card">
        <div class="detail-title">
          <h3>{{ currentIncident.level }} · {{ currentIncident.title }}</h3>
          <button class="detail-close" aria-label="隐藏事件详情" @click="$emit('hide-detail')">隐藏</button>
        </div>
        <dl>
          <dt>当前状态</dt><dd>{{ currentIncident.status }}</dd>
          <dt>负责人</dt><dd>{{ currentIncident.owner || '-' }}</dd>
          <dt>所属业务</dt><dd>{{ currentIncident.business || '-' }}</dd>
          <dt>开始时间</dt><dd>{{ formatDateTime(currentIncident.startedAt) }}</dd>
          <dt>摘要</dt><dd>{{ currentIncident.summary || '暂无摘要' }}</dd>
        </dl>
        <label>事件状态
          <select :value="currentIncident.status" aria-label="事件状态" :disabled="isSample(currentIncident) || incidentStatusOptions(currentIncident.status).length === 1" @change="$emit('update-status', currentIncident, $event.target.value)">
            <option v-for="status in incidentStatusOptions(currentIncident.status)" :key="status">{{ status }}</option>
          </select>
        </label>
        <div class="detail-actions">
          <button class="primary" disabled>War Room（灰度）</button>
          <button disabled>复盘（灰度）</button>
          <button :disabled="isSample(currentIncident)" @click="$emit('edit', currentIncident)">编辑事件</button>
          <button class="danger-action" :disabled="isSample(currentIncident)" @click="$emit('delete', currentIncident)">删除事件</button>
        </div>
        <ol class="flow compact"><li>新建</li><li>处理中</li><li>已恢复</li><li>已关闭</li></ol>
      </aside>
    </div>
  </section>
</template>
