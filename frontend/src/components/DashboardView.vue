<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { dateISO, formatRate, greeting, periods, trendGeometry } from '../dashboard-workspace'
import SvgIcon from './SvgIcon.vue'
import WorkspaceDialog from './WorkspaceDialog.vue'

const props = defineProps({
  workspace: { type: Object, required: true }, preview: { type: Object, default: null },
  previewEnabled: { type: Boolean, default: false }, previewLoading: { type: Boolean, default: false },
  previewError: { type: String, default: '' }, username: { type: String, default: '' }
})
const emit = defineEmits(['navigate', 'open-priority', 'open-copilot', 'preview-query', 'toggle-preview', 'notify'])
const selectedSystem = ref('')
const selectedDemoSystem = ref('all')
const systemSelection = computed({
  get: () => props.previewEnabled ? selectedDemoSystem.value : selectedSystem.value,
  set: value => { if (props.previewEnabled) selectedDemoSystem.value = value; else selectedSystem.value = value }
})
const selectedPeriod = ref('today')
const start = ref(dateISO(new Date(Date.now() - 6 * 86400000)))
const end = ref(dateISO())
const filter = ref('all')
const selectedRiskId = ref('')
const dialogOpen = ref(false)
const dialogKind = ref('')
const chart = ref(null)
const chartWidth = ref(620)
const chartHeight = ref(245)
const now = ref(new Date())
let resizeObserver, clock
const stats = computed(() => props.workspace.statistics || {})
const demo = computed(() => props.previewEnabled && props.preview?.source === 'demo' ? props.preview : null)
const risks = computed(() => demo.value?.risks || [])
const visibleRisks = computed(() => risks.value.filter(item => filter.value === 'all' || item.level === filter.value))
const risk = computed(() => visibleRisks.value.find(item => item.id === selectedRiskId.value) || visibleRisks.value[0] || null)
const insight = computed(() => demo.value?.insights.find(item => item.riskId === risk.value?.id) || null)
const incidents = computed(() => (props.workspace.incidents || []).filter(item => item.status !== '已关闭'))
const people = computed(() => [props.workspace.duty.primary, props.workspace.duty.backup].filter(Boolean))
const primary = computed(() => props.workspace.duty.primary)
const backup = computed(() => props.workspace.duty.backup)
const handovers = computed(() => props.workspace.duty.handovers || [])
const geometry = computed(() => trendGeometry((props.previewEnabled ? demo.value?.trend.points : props.workspace.trend.points) || [], chartWidth.value, chartHeight.value, selectedPeriod.value))
const trendAreas = computed(() => geometry.value.segments.map(segment => {
  const points = segment.split(' ')
  return `${points[0].split(',')[0]},${geometry.value.bottom} ${segment} ${points.at(-1).split(',')[0]},${geometry.value.bottom}`
}))
const periodLabel = computed(() => selectedPeriod.value === 'custom' ? `${start.value} 至 ${end.value}` : periods.find(item => item.value === selectedPeriod.value)?.label)
const systemLabel = computed(() => props.previewEnabled ? demo.value?.businessSystems.find(item => item.id === selectedDemoSystem.value)?.name || '全部业务系统' : selectedSystem.value || '全部业务系统')
const assetFoot = computed(() => Object.entries(stats.value.assetTypeCounts || {}).map(([name, count]) => `${name} ${count}`).join(' · ') || '暂无资产分类')
const incidentFoot = computed(() => Object.entries(stats.value.incidentLevelCounts || {}).map(([name, count]) => `${name} ${count}`).join(' · ') || '已形成事件单')
const alertFoot = computed(() => Object.entries(demo.value?.alerts?.levelCounts || {}).map(([level, count]) => `${level} ${count}`).join(' · ') || '告警中心与监控数据待接入')
const validDates = computed(() => Boolean(start.value && end.value && start.value <= end.value && end.value <= dateISO()))

function requestPreview() {
  emit('preview-query', { period: selectedPeriod.value, system: selectedDemoSystem.value, from: start.value, to: end.value })
}
function choosePeriod(value) {
  if (value === 'custom') { openDialog('dates'); return }
  selectedPeriod.value = value
  requestPreview()
}
function openDialog(kind) { dialogKind.value = kind; dialogOpen.value = true }
function applyDates() {
  if (!validDates.value) return
  selectedPeriod.value = 'custom'
  dialogOpen.value = false
  requestPreview()
}
function confirmPlan() {
  dialogOpen.value = false
  emit('notify', '已记录当前页面演示意向；未审批、未执行任何生产处置。')
}
function openIncident(item) {
  emit('open-priority', { target: 'incidents', source: item })
}
function countRisks(level) { return risks.value.filter(item => level === 'all' || item.level === level).length }
watch(() => props.previewEnabled, requestPreview)
onMounted(() => {
  resizeObserver = new ResizeObserver(([entry]) => {
    chartWidth.value = entry.contentRect.width
    chartHeight.value = entry.contentRect.height
  })
  resizeObserver.observe(chart.value)
  clock = window.setInterval(() => { now.value = new Date() }, 60000)
})
onBeforeUnmount(() => { resizeObserver?.disconnect(); window.clearInterval(clock) })
const dialogTitles = { dates: '自定义趋势时间', methodology: '业务系统服务可用性 · 口径说明', summary: '巡检摘要 · 预设演示', detail: '风险详情 · 演示数据', plan: '预案审阅 · 演示数据', duty: '值班交接 · 真实业务数据' }
</script>

<template>
  <section class="review-dashboard" aria-label="首页健康总览">
    <div class="demo"><b>业务数据：真实统计与值班 · 未接入模块：{{ demo ? '演示数据 / source: demo' : '待接入' }}</b><label class="preview-choice"><input type="checkbox" :checked="previewEnabled" @change="emit('toggle-preview', $event.target.checked)" />开启独立演示</label></div>
    <p v-if="previewLoading" class="source-message" role="status">正在读取独立演示…</p><p v-if="previewError" class="source-message red" role="alert">{{ previewError }}</p>
    <section class="hero glass"><div><h1>{{ greeting(now) }}，{{ username }}</h1><div class="hero-meta"><span class="green"><i class="dot"></i>首页健康总览</span><span>真实业务工作区</span></div></div><button class="summary" type="button" @click="demo ? openDialog('summary') : emit('open-copilot')"><div class="summary-head"><SvgIcon class="icon cyan" name="copilot" /><strong>AI 巡检摘要</strong><span class="tag">{{ demo ? '预设演示' : '待分析' }}</span></div><p><span :class="{ 'summary-risk': risk }">{{ risk ? `优先研判：${risk.title}` : '当前没有可用的实时观测摘要' }}</span><br>{{ insight?.analysis || (demo ? '预设虚构洞察，根因须人工核验。' : '打开真实 Copilot，查询授权业务上下文。') }}</p><span class="summary-link">查看摘要与证据 →</span></button></section>
    <section class="kpis" aria-label="核心指标">
      <button class="kpi glass" type="button" @click="emit('navigate', 'cmdb')"><div class="kpi-head"><span>纳管资产</span><SvgIcon class="icon cyan" name="server" /></div><strong>{{ stats.assetCount ?? 0 }}<small>真实数据</small></strong><div class="kpi-foot">{{ assetFoot }}</div></button>
      <article class="kpi glass kpi-alert"><div class="kpi-head"><span class="alert-icon"><SvgIcon class="icon" name="bell" /></span><span>紧急告警</span><span class="tag">{{ demo ? '演示' : '待接入' }}</span></div><strong class="red">{{ demo?.alerts?.urgentCount ?? '--' }}<small>{{ demo ? '演示数据' : '无有效观测' }}</small></strong><div class="kpi-foot">{{ alertFoot }}</div></article>
      <button class="kpi glass" type="button" @click="emit('navigate', 'incidents')"><div class="kpi-head"><span>待处理事件</span><SvgIcon class="icon cyan" name="dashboard" /></div><strong>{{ stats.activeIncidentCount ?? 0 }}<small>真实数据</small></strong><div class="kpi-foot">{{ incidentFoot }}</div></button>
      <article class="kpi glass"><div class="kpi-head"><span>核心服务可用性</span><span class="tag">{{ demo ? '演示' : 'SLA 待接入' }}</span></div><strong>{{ formatRate(geometry.mean) }}<small>{{ demo ? '演示数据' : '无有效样本' }}</small></strong><div class="kpi-foot">{{ periodLabel }}有效请求观测 · 目标 99.9%</div></article>
      <article class="kpi glass"><div class="kpi-head"><span>自动化执行成功率</span><span class="tag">{{ demo ? '演示' : '待接入' }}</span></div><strong>{{ formatRate(demo?.automation?.successRate) }}<small>{{ demo ? '演示数据' : '无执行样本' }}</small></strong><div class="kpi-foot">{{ demo ? `模拟记录：成功 ${demo.automation?.succeeded ?? 0} · 失败 ${demo.automation?.failed ?? 0}` : '建议态与执行态分离' }}</div></article>
      <article class="kpi glass kpi-duty"><div class="kpi-head"><button class="kpi-title" type="button" @click="emit('navigate', 'oncall')">当前值班</button><span class="tag">真实数据</span></div><strong class="duty-person">{{ primary?.name || '未安排' }}<small>{{ primary?.role || '当前无主值班' }}{{ backup ? ` · 备班 ${backup.name}` : '' }}</small></strong><p class="duty-shift">{{ primary ? `本班 ${primary.since || '--'}–${primary.until || '--'}` : '暂无当前班次' }}</p><div class="kpi-foot duty-handoff"><span class="amber">待交接 {{ workspace.duty.pendingCount ?? 0 }} 项</span><button type="button" @click="openDialog('duty')">查看交接</button></div></article>
    </section>
    <div class="middle">
      <section class="glass"><div class="head trend-head"><h2><SvgIcon class="icon" name="activity" />业务系统健康趋势 <span class="tag trend-status">{{ demo ? '演示数据' : '设计口径 / 待接入' }}</span></h2><div class="trend-controls"><button id="trendInfo" type="button" @click="openDialog('methodology')">口径说明</button><div class="segments" aria-label="趋势时间范围"><button v-for="item in periods" :key="item.value" type="button" :class="{ active: selectedPeriod === item.value }" :aria-pressed="selectedPeriod === item.value" @click="choosePeriod(item.value)">{{ item.label }}</button></div></div></div>
        <div class="trend-scope"><div><strong>{{ systemLabel }}</strong> <span class="muted">· {{ previewEnabled ? '演示目录 / source: demo' : '真实业务目录 / source: real' }}</span></div><div class="scope-controls"><label for="systemSelect">业务系统</label><select id="systemSelect" v-model="systemSelection" aria-label="选择业务系统" @change="requestPreview"><option :value="previewEnabled ? 'all' : ''">全部业务系统</option><template v-if="previewEnabled"><option v-for="item in (demo?.businessSystems || []).filter(item => item.id !== 'all')" :key="item.id" :value="item.id">{{ item.name }} · 演示</option></template><template v-else><option v-for="item in workspace.businessSystems" :key="item" :value="item">{{ item }}</option></template></select></div></div>
        <div class="chart-caption"><span class="metric-switch"><strong>核心服务可用性 / %</strong></span><span>成功请求 / 总请求{{ selectedPeriod === 'today' ? ' · 当日有效观测' : ' · 时间窗内聚合' }}</span><span>┄ 目标 99.90%</span></div>
        <svg id="trend" ref="chart" :viewBox="`0 0 ${chartWidth} ${chartHeight}`" role="img" :aria-label="`${periodLabel}${systemLabel}核心服务可用性趋势，${demo ? '演示数据' : '无有效观测'}`">
          <g v-for="tick in geometry.ticks" :key="tick.value"><line :x1="geometry.left" :y1="tick.y" :x2="geometry.right" :y2="tick.y" stroke="var(--line)" stroke-opacity=".3" /><text :x="geometry.left - 6" :y="tick.y + 4" text-anchor="end">{{ tick.value.toFixed(2) }}%</text></g>
          <defs><linearGradient id="workspace-trend-area" x1="0" y1="0" x2="0" y2="1"><stop stop-color="#69c5cd" stop-opacity=".25" /><stop offset="1" stop-color="#69c5cd" stop-opacity=".01" /></linearGradient></defs>
          <polygon v-for="(area, index) in trendAreas" :key="`area-${index}`" :points="area" fill="url(#workspace-trend-area)" stroke="none" />
          <line :x1="geometry.left" :x2="geometry.right" :y1="geometry.targetY" :y2="geometry.targetY" stroke="var(--amber)" stroke-opacity=".6" stroke-dasharray="5 5" />
          <polyline v-for="(segment, index) in geometry.segments" :key="index" :points="segment" fill="none" stroke="var(--cyan)" stroke-width="2.8" stroke-linejoin="round" />
          <template v-for="(point, index) in geometry.nodes" :key="index"><circle v-if="point.y !== null" :cx="point.x" :cy="point.y" r="3" fill="var(--cyan)"><title>{{ point.time }}：{{ formatRate(point.availability) }} · 演示数据</title></circle></template>
          <text v-for="(label, index) in geometry.labels" :key="index" :x="label.x" :y="chartHeight - 9" text-anchor="middle">{{ label.label }}</text>
          <text v-if="geometry.mean === null" :x="(geometry.left + geometry.right) / 2" :y="chartHeight / 2" text-anchor="middle">-- · 核心服务观测数据待接入</text>
        </svg>
        <div class="trend-note"><strong>{{ demo ? '虚构演示数据' : '无有效观测' }}</strong> · 趋势只展示核心服务可用性；资产状态健康率单独呈现，不纳入此趋势。真实监控待接入。</div><div class="chart-foot"><span>{{ periodLabel }} · 均值 {{ formatRate(geometry.mean) }}</span><span>事件 {{ stats.activeIncidentCount ?? 0 }} · 真实独立计数</span></div>
      </section>
      <section class="glass"><div class="head section-head"><div class="section-head-copy"><h2><SvgIcon class="icon" name="shield" />风险与优先级</h2><p class="section-note">来自告警、SLO、依赖或证书信号，先排序研判，未必已形成事件单。</p></div><span class="muted">{{ demo ? '演示' : '待接入' }} · {{ risks.length }} 项</span></div><div class="filters" aria-label="风险等级筛选"><button v-for="level in ['all', 'P1', 'P2', 'P3']" :key="level" :data-level="level" :class="{ active: filter === level }" :aria-pressed="filter === level" type="button" @click="filter = level">{{ level === 'all' ? '全部' : level }} {{ countRisks(level) }}</button></div><div class="risk-list"><button v-for="item in visibleRisks" :key="item.id" class="risk-row" :class="{ selected: risk?.id === item.id }" :data-severity="item.level" :aria-pressed="risk?.id === item.id" @click="selectedRiskId = item.id"><span class="level" :class="item.level">{{ item.level }}</span><span class="risk-copy"><strong>{{ item.title }}</strong><small>{{ item.asset }} · 影响：{{ item.impact }}</small></span><span class="risk-time">{{ item.time }} ›</span></button><p v-if="!visibleRisks.length" class="review-empty">{{ demo ? '当前等级暂无演示风险' : '风险信号待接入' }}</p></div></section>
    </div>
    <div class="bottom">
      <section class="glass"><div class="head section-head"><div class="section-head-copy"><h2>待处理事件</h2><p class="section-note">已形成事件单，需要负责人推动处理 / 恢复 / 关闭。</p></div><button class="tag" type="button" @click="emit('navigate', 'incidents')">真实数据</button></div><table><thead><tr><th>等级</th><th>事件名称</th><th>负责人</th><th>状态</th></tr></thead><tbody><tr v-for="item in incidents.slice(0, 4)" :key="item.id"><td><span class="level" :class="item.level">{{ item.level }}</span></td><td><button class="event-name" @click="openIncident(item)">{{ item.title }}</button></td><td>{{ item.owner || '未指定' }}</td><td class="cyan">{{ item.status }}</td></tr><tr v-if="!incidents.length"><td colspan="4" class="review-empty">暂无未关闭事件</td></tr></tbody></table><div class="table-note">优先展示 {{ Math.min(incidents.length, 4) }} / {{ stats.activeIncidentCount ?? 0 }} 项 · 真实全局事件</div></section>
      <section class="glass"><div class="head"><h2>服务健康状况</h2><span class="muted">24h · {{ demo ? '演示数据' : '待接入' }}</span></div><div class="service muted"><span>服务名称</span><small>可用率</small><small>P95</small></div><div v-for="item in demo?.services || []" :key="item.id || item.name" class="service"><span><i class="dot" :class="item.status === 'warning' ? 'amber' : 'green'"></i>{{ item.name }}</span><b>{{ formatRate(item.availability) }}</b><small>{{ Number.isFinite(item.p95Ms) ? `${item.p95Ms} ms` : '--' }}</small></div><p v-if="!demo?.services?.length" class="review-empty">暂无核心服务观测数据</p></section>
      <section id="aiPanel" class="glass" :data-severity="risk?.level || 'P3'"><div class="head"><h2><SvgIcon class="icon" name="copilot" />AI 运维洞察</h2><span class="tag">{{ demo ? '预设演示' : '建议态' }}</span></div><div class="ai-body" aria-live="polite"><div id="aiContext" class="ai-context">{{ risk ? `${risk.id} · ${risk.level} · 预设关联分析` : '尚无观测上下文' }}</div><h3>{{ risk?.title || '暂无可用洞察' }}</h3><p>{{ insight?.analysis || risk?.analysis || '真实 Copilot 可基于已授权业务数据回答；当前未生成巡检洞察。' }}</p><div class="ai-evidence">{{ insight?.evidence || risk?.evidence || '无有效证据' }}{{ demo ? ' · 演示证据，根因待人工核验。' : '' }}</div><div class="ai-actions"><button type="button" :disabled="!risk" @click="openDialog('detail')">查看详情</button><button class="primary" type="button" :disabled="!risk" @click="openDialog('plan')">审阅预案</button></div><button v-if="!demo" class="copilot-query" type="button" @click="emit('open-copilot')">打开真实 Copilot</button></div></section>
    </div>
    <footer class="footer glass"><strong><i class="dot"></i>业务工作区 · 真实数据</strong><span class="stat">未闭环任务 {{ stats.activeTaskCount ?? 0 }}</span><span class="stat">观测 {{ demo ? '演示数据' : '待接入' }}</span><div class="quick"><span>快捷入口</span><button disabled>告警中心 ↗</button><button @click="emit('navigate', 'incidents')">事件中心 ↗</button><button disabled>自动化任务 ↗</button><button disabled>知识库 ↗</button></div></footer><div class="brand-line"><span>首页看健康，异常看影响，告警看根因，处置看流程，复盘看改进。</span><span>AI 贯穿查询、分析、建议和自动化 · OpsCore</span></div>
    <WorkspaceDialog v-model:open="dialogOpen" :title="dialogTitles[dialogKind] || '详情'">
      <template v-if="dialogKind === 'dates'"><p>选择趋势统计的起止日期。</p><div class="date-fields"><label class="date-field">开始日期<input v-model="start" type="date" :max="end || dateISO()" /></label><label class="date-field">结束日期<input v-model="end" type="date" :min="start" :max="dateISO()" /></label></div><p class="date-help">仅对未接入模块演示趋势生效，不代表真实观测。</p><p v-if="!validDates" role="alert">请选择不晚于今日且开始不晚于结束的有效日期。</p></template>
      <template v-else-if="dialogKind === 'methodology'"><ol><li>只统计所选业务系统关联核心服务的成功请求 / 总请求；无有效请求数据时保留 --。</li><li>资产状态健康率单独呈现，不纳入此趋势。</li><li>今日按小时展示，近 7 天、近 30 天和自定义按时间窗聚合。</li><li>告警与事件独立计数，不共用可用性纵轴。</li></ol><div class="warning">真实监控待接入。演示源为 source: demo，虚构数据不可用于生产判断。</div></template>
      <template v-else-if="dialogKind === 'duty'"><span class="tag">真实数据 / source: real</span><p v-for="person in people" :key="person.id">{{ person.name }} · {{ person.role }} · {{ person.since }}–{{ person.until }}</p><p v-if="!people.length">暂无当前值班人员</p><section v-for="item in handovers" :key="item.id"><h3>{{ item.from || '未指定' }} → {{ item.to || '未指定' }} · {{ item.time }}</h3><p>{{ item.content }}</p></section><p v-if="!handovers.length">暂无交接记录</p><button @click="dialogOpen = false; emit('navigate', 'oncall')">进入值班管理</button></template>
      <template v-else><span class="tag">预设虚构数据 / source: demo</span><h3>{{ risk?.title }}</h3><p>{{ insight?.analysis || risk?.analysis }}</p><p>影响：{{ risk?.impact }}</p><p>证据：{{ insight?.evidence || risk?.evidence }}</p><p v-if="dialogKind === 'summary'">共 {{ risks.length }} 项风险，P1 {{ countRisks('P1') }} 项。没有调用真实 AI。</p><ol v-if="dialogKind === 'plan'"><li v-for="(step, index) in (insight?.plan || risk?.plan || '').split('；').filter(Boolean)" :key="index">{{ step }}</li></ol><div class="warning">{{ dialogKind === 'plan' ? '确认仅记录当前页面模拟意向，不代表审批通过或生产处置已执行。' : '风险与真实事件分属独立清单；未转换、未写入业务数据。' }}</div></template>
      <template #actions><button v-if="dialogKind === 'dates'" class="primary" :disabled="!validDates" @click="applyDates">应用</button><button v-else-if="dialogKind === 'plan'" class="primary" @click="confirmPlan">确认演示意向</button></template>
    </WorkspaceDialog>
  </section>
</template>
