<script setup>
import { computed } from 'vue'
import SvgIcon from './SvgIcon.vue'

const props = defineProps({
  assetCount: { type: Number, default: 0 },
  oncallCount: { type: Number, default: 0 },
  taskCount: { type: Number, default: 0 },
  incidentCount: { type: Number, default: 0 },
  assetBars: { type: Array, default: () => [] },
  todayOncall: { type: Object, default: () => ({}) },
  taskCards: { type: Array, default: () => [] },
  incidentLevels: { type: Array, default: () => [] },
  metrics: { type: Object, default: () => ({}) },
  priorityItems: { type: Array, default: () => [] }
})

const emit = defineEmits(['navigate', 'open-priority'])

const responseBars = computed(() => {
  const max = Math.max(...props.incidentLevels.map((item) => Number(item.value || 0)), 1)
  return props.incidentLevels.map((item) => ({
    ...item,
    height: `${Math.max(12, Math.round((Number(item.value || 0) / max) * 50))}px`
  }))
})

function formatRate(value) {
  return Number.isFinite(value) ? `${value}%` : '--'
}

function formatDuration(value) {
  return Number.isFinite(value) ? `${value}m` : '--'
}
</script>

<template>
  <section class="dashboard">
    <div class="kpis">
      <button class="kpi-card kpi-assets" @click="emit('navigate', 'cmdb')">
        <span class="kpi-icon"><SvgIcon name="asset" /></span>
        <span class="kpi-copy"><small>纳管资产</small></span>
        <span class="kpi-value"><strong>{{ assetCount }}</strong><small>项</small></span>
        <span class="kpi-mini asset-bars">
          <span v-for="item in assetBars" :key="item.label" :class="['asset-bar', item.tone]">
            <b>{{ item.label }}</b>
            <i :style="{ height: item.height }"><em>{{ item.value }}</em></i>
          </span>
        </span>
        <span class="kpi-status">纳管覆盖正常</span>
      </button>
      <button class="kpi-card kpi-oncall" @click="emit('navigate', 'oncall')">
        <span class="kpi-icon"><SvgIcon name="calendar" /></span>
        <span class="kpi-copy"><small>今日值班</small></span>
        <span class="kpi-value"><strong>{{ oncallCount }}</strong><small>人</small></span>
        <span class="kpi-mini oncall-roster">
          <span><small>主值</small><b>{{ todayOncall.primary || '未配置' }}</b></span>
          <span><small>备值</small><b>{{ todayOncall.backup || '未配置' }}</b></span>
          <em>{{ todayOncall.ruleType === 'weekly' ? '本周轮换' : '08:00-20:00' }}</em>
        </span>
        <span class="kpi-status">响应窗口已覆盖</span>
      </button>
      <button class="kpi-card kpi-tasks" @click="emit('navigate', 'tasks')">
        <span class="kpi-icon"><SvgIcon name="task" /></span>
        <span class="kpi-copy"><small>进行中任务</small></span>
        <span class="kpi-value"><strong>{{ taskCount }}</strong><small>项</small></span>
        <span class="kpi-mini task-status-grid">
          <span v-for="item in taskCards" :key="item.label" class="task-status-item">
            <small><i :style="{ background: item.color }"></i>{{ item.label }}</small>
            <b>{{ item.value }}</b>
            <em><i :style="{ width: item.width, background: item.color }"></i></em>
          </span>
        </span>
        <span class="kpi-status">闭环节奏稳定</span>
      </button>
      <button class="kpi-card kpi-incidents" @click="emit('navigate', 'incidents')">
        <span class="kpi-icon"><SvgIcon name="incident" /></span>
        <span class="kpi-copy"><small>活跃事件</small></span>
        <span class="kpi-value"><strong>{{ incidentCount }}</strong><small>起</small></span>
        <span class="kpi-mini incident-levels">
          <span v-for="item in incidentLevels" :key="item.label" :class="item.tone">
            <b>{{ item.label }}</b>
            <strong>{{ item.value }}</strong>
            <em>{{ item.desc }}</em>
          </span>
        </span>
        <span class="kpi-status">风险持续跟进</span>
      </button>
    </div>

    <div class="metric-grid">
      <article class="metric metric-gauge">
        <div class="metric-copy"><span>资产健康率</span><strong>{{ formatRate(metrics.assetHealthRate) }}</strong><small>{{ metrics.assetHealthy || 0 }} 健康 · {{ metrics.assetAbnormal || 0 }} 异常</small></div>
        <div class="gauge" :style="{ '--value': metrics.assetHealthRate || 0 }"><b>{{ formatRate(metrics.assetHealthRate) }}</b></div>
      </article>
      <article class="metric metric-bars">
        <div class="metric-copy"><span>事件平均响应</span><strong>{{ formatDuration(metrics.averageResponseMinutes) }}</strong><small>{{ metrics.responseSampleCount || 0 }} 个有效恢复样本</small></div>
        <div class="response-bars"><i v-for="item in responseBars" :key="item.label" :style="{ height: item.height }"><em>{{ item.label }}</em></i></div>
      </article>
      <article class="metric metric-stack">
        <div class="metric-copy"><span>任务闭环率</span><strong>{{ formatRate(metrics.taskClosureRate) }}</strong><small>{{ metrics.taskClosed || 0 }} 已闭环 · {{ metrics.taskOpen || 0 }} 未闭环</small></div>
        <div class="stacked-progress"><i class="ok" :style="{ width: `${Number.isFinite(metrics.taskClosureRate) ? metrics.taskClosureRate : 0}%` }"></i><i class="warn" :style="{ width: `${Number.isFinite(metrics.taskClosureRate) ? 100 - metrics.taskClosureRate : 0}%` }"></i></div>
        <div class="metric-legend"><span>已闭环</span><span>未闭环</span></div>
      </article>
      <article class="metric metric-risk">
        <div class="metric-copy"><span>事件闭环率</span><strong>{{ formatRate(metrics.incidentClosureRate) }}</strong><small>{{ metrics.incidentClosed || 0 }} 已关闭 · {{ metrics.incidentActive || 0 }} 持续跟进</small></div>
        <div class="risk-orbit"><i></i><i></i><i></i><i></i><b>闭环</b></div>
      </article>
    </div>

    <div class="split">
      <section class="panel">
        <h3>事件与任务优先级</h3>
        <div class="priority-list">
          <button v-for="item in priorityItems" :key="item.id" class="priority-row" @click="emit('open-priority', item)">
            <span :class="['pill', item.type === '事件' ? 'danger' : '']">{{ item.type }} {{ item.badge }}</span>
            <strong>{{ item.title }}</strong>
            <span>{{ item.owner }} · {{ item.status }} · {{ item.meta }}</span>
          </button>
          <p v-if="!priorityItems.length" class="empty">暂无待关注事件或任务</p>
        </div>
      </section>
      <section class="panel">
        <h3>事件响应流程</h3>
        <div class="flow-diagram" aria-label="事件响应流程图">
          <div class="flow-node active"><span>01</span><strong>创建事件</strong><small>分级与关联资产</small></div>
          <i class="flow-arrow">→</i>
          <div class="flow-node"><span>02</span><strong>协同处置</strong><small>主值/负责人跟进</small></div>
          <i class="flow-arrow">→</i>
          <div class="flow-node"><span>03</span><strong>恢复确认</strong><small>验证业务影响</small></div>
          <i class="flow-arrow">→</i>
          <div class="flow-node done"><span>04</span><strong>关闭复盘</strong><small>沉淀改进项</small></div>
        </div>
        <div class="flow-legend"><span><i class="legend-dot danger"></i>P1/P2 拉起响应</span><span><i class="legend-dot success"></i>恢复后关闭</span></div>
      </section>
    </div>
  </section>
</template>
