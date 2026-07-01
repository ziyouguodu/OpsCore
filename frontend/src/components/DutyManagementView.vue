<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { api } from '../api'
import { buildDutyCalendarCells, calculateDutyBalance, createDutyAssignments, getUpcomingAssignments, toISODate } from '../duty-date'

const props = defineProps({
  active: { type: Boolean, default: false },
  canWrite: { type: Boolean, default: false },
  requestConfirm: { type: Function, required: true },
  users: { type: Array, default: () => [] }
})

const emit = defineEmits(['error'])

const dutySections = [
  { id: 'overview', label: '概览' },
  { id: 'calendar', label: '排班日历' },
  { id: 'schedules', label: '排班配置' },
  { id: 'roster', label: '值班列表' },
  { id: 'handover', label: '交接班日志' },
  { id: 'escalation', label: '升级策略' }
]
const weekdayLabels = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
const dutyToday = new Date()
const dutyTodayKey = toISODate(dutyToday)
const dutyRevision = ref(0)
const dutyLoaded = ref(false)
const dutySaving = ref(false)
const dutyTeams = ref([{ label: '全部团队', count: 0 }])
const dutySection = ref('overview')
const dutyTeamFilter = ref('全部团队')
const dutyGlobalSearch = ref('')
const dutyRosterSearch = ref('')
const dutyCalendarMonth = ref(new Date(dutyToday.getFullYear(), dutyToday.getMonth(), 1))
const dutyToasts = ref([])
const dutyAssignModalOpen = ref(false)
const dutyScheduleModalOpen = ref(false)
const dutyHandoverModalOpen = ref(false)
const dutyEscalationModalOpen = ref(false)
const dutyEscalationSnapshot = ref(null)
const dutyUserModalOpen = ref(false)
const dutyTeamModalOpen = ref(false)
const dutyTeamDrafts = ref([])
const dutySelectedDate = ref(dutyTodayKey)
const dutyScheduleModalMode = ref('create')
const dutyEditingScheduleId = ref(null)
const dutyEditingUserId = ref(null)

const dutyAssignForm = reactive({ date: dutyTodayKey, type: 'primary', person: '', note: '', rosterId: null })
const dutyScheduleForm = reactive({ name: '', team: '', rotation: 'weekly', time: '08:00-20:00', members: [] })
const dutyHandoverForm = reactive({ from: '', to: '', content: '', complete: true })
const dutyEscalationForm = reactive({ name: '', team: '', severity: 'P1' })
const dutyUserForm = reactive({ userId: '', team: '', role: '运维工程师', next: '待安排' })

const dutyEscalationLevels = ref([])
const dutyCurrentPeople = ref([])
const dutySchedules = ref([])
const dutyRoster = ref([])
const dutyHandovers = ref([])
const dutyAssignments = reactive(createDutyAssignments(dutyToday, [], 0))
const dutyPeopleOptions = computed(() => dutyRoster.value.map((person) => person.name))

const dutyFilteredCurrent = computed(() => dutyTeamFilter.value === '全部团队'
  ? dutyCurrentPeople.value
  : dutyCurrentPeople.value.filter((person) => person.team === dutyTeamFilter.value))
const dutyFilteredRoster = computed(() => dutyRoster.value.filter((person) => {
  const matchesTeam = dutyTeamFilter.value === '全部团队' || person.team === dutyTeamFilter.value
  const keyword = dutyRosterSearch.value.trim().toLowerCase()
  const matchesKeyword = !keyword || [person.name, person.team, person.role, person.status].some((value) => String(value).toLowerCase().includes(keyword))
  return matchesTeam && matchesKeyword
}))
const dutyUpcomingAssignments = computed(() => getUpcomingAssignments(dutyAssignments, dutyToday, 3))
const dutyCalendarMonthLabel = computed(() => `${dutyCalendarMonth.value.getFullYear()}年${dutyCalendarMonth.value.getMonth() + 1}月`)
const dutyCalendarCells = computed(() => buildDutyCalendarCells(dutyCalendarMonth.value, dutyAssignments, dutyToday))
const dutyTotalCount = computed(() => dutyRoster.value.reduce((sum, person) => sum + Number(person.count || 0), 0))
const dutyReports = computed(() => {
  const counts = dutyRoster.value.map((person) => Number(person.count || 0))
  const balance = calculateDutyBalance(counts)
  return [
    { label: '值班均衡度', value: balance === null ? '--' : `${balance}%`, detail: `${dutyRoster.value.length} 名值班人员`, width: `${balance || 0}%` },
    { label: '累计值班次数', value: String(dutyTotalCount.value), detail: '依据值班列表统计', width: `${Math.min(100, dutyTotalCount.value * 4)}%` },
    { label: '交接记录', value: String(dutyHandovers.value.length), detail: `${dutyHandovers.value.filter((item) => !item.complete).length} 条待确认`, width: `${Math.min(100, dutyHandovers.value.length * 10)}%` },
    { label: '启用排班', value: String(dutySchedules.value.filter((item) => item.active).length), detail: `${dutySchedules.value.length} 个排班模板`, width: `${dutySchedules.value.length ? Math.round(dutySchedules.value.filter((item) => item.active).length / dutySchedules.value.length * 100) : 0}%` }
  ]
})
const dutySystemUserOptions = computed(() => props.users.map((user) => ({
  id: user.id,
  username: user.username,
  name: user.displayName || user.username,
  role: (user.roles || [])[0] || 'ops_engineer'
})))

function serializeDutyData() {
  return {
    teams: dutyTeams.value.filter((team) => team.label !== '全部团队').map((team) => ({ name: team.label })),
    members: dutyRoster.value.map((person) => ({ ...person, id: String(person.id) })),
    schedules: dutySchedules.value.map((schedule) => ({ ...schedule, id: String(schedule.id), members: [...schedule.members] })),
    assignments: Object.fromEntries(Object.entries(dutyAssignments).map(([date, assignment]) => [date, { primary: assignment.primary || '', backup: assignment.backup || '' }])),
    currentPeople: dutyCurrentPeople.value.map((person) => ({ id: String(person.id), name: person.name, role: person.role, team: person.team, since: person.since, until: person.until, phone: person.phone, status: person.status })),
    handovers: dutyHandovers.value.map((item) => ({ ...item, id: String(item.id) })),
    escalation: { ...dutyEscalationForm, levels: dutyEscalationLevels.value.map((level) => ({ ...level })) }
  }
}

function applyDutyState(state) {
  const data = state?.data || {}
  dutyRevision.value = Number(state?.revision || 0)
  dutyTeams.value = [{ label: '全部团队', count: 0 }, ...(data.teams || []).map((team) => ({ label: team.name, count: 0 }))]
  dutyRoster.value = (data.members || []).map((member) => ({ ...member }))
  dutySchedules.value = (data.schedules || []).map((schedule) => ({ ...schedule, members: [...(schedule.members || [])] }))
  dutyCurrentPeople.value = (data.currentPeople || []).map((person) => ({ ...person, avatar: person.name?.slice(0, 1) || '' }))
  dutyHandovers.value = (data.handovers || []).map((item) => ({ ...item }))
  for (const key of Object.keys(dutyAssignments)) delete dutyAssignments[key]
  Object.assign(dutyAssignments, data.assignments || {})
  Object.assign(dutyEscalationForm, data.escalation || { name: '', team: '', severity: 'P1' })
  dutyEscalationLevels.value = (data.escalation?.levels || []).map((level) => ({ ...level }))
  refreshDutyTeamCounts()
  dutyLoaded.value = true
}

async function loadDutyCenter(force = false) {
  if (dutyLoaded.value && !force) return
  try {
    applyDutyState(await api('/duty-center'))
  } catch (err) {
    emit('error', `值班数据加载失败：${err.message}`)
  }
}

async function persistDutyCenter(message) {
  if (!props.canWrite || dutySaving.value) return false
  dutySaving.value = true
  try {
    const saved = await api('/duty-center', {
      method: 'PUT',
      body: JSON.stringify({ revision: dutyRevision.value, data: serializeDutyData() })
    })
    applyDutyState(saved)
    if (message) pushDutyToast(message)
    return true
  } catch (err) {
    emit('error', `值班数据保存失败：${err.message}`)
    await loadDutyCenter(true)
    return false
  } finally {
    dutySaving.value = false
  }
}

function pushDutyToast(message, tone = 'success') {
  const toast = { id: Date.now() + Math.random(), message, tone }
  dutyToasts.value.push(toast)
  window.setTimeout(() => {
    dutyToasts.value = dutyToasts.value.filter((item) => item.id !== toast.id)
  }, 2600)
}

function setDutySection(section) {
  if (dutySections.some((item) => item.id === section)) dutySection.value = section
}

function setDutyTeam(team) {
  dutyTeamFilter.value = team
  pushDutyToast(`已切换到${team}`, 'info')
}

function refreshDutyTeamCounts() {
  const teamLabels = dutyTeams.value.filter((team) => team.label !== '全部团队').map((team) => team.label)
  dutyTeams.value = [
    { label: '全部团队', count: dutyRoster.value.length },
    ...teamLabels.map((label) => ({ label, count: dutyRoster.value.filter((person) => person.team === label).length }))
  ]
}

function requireWrite(message) {
  if (props.canWrite) return true
  emit('error', message)
  return false
}

function openDutyTeamModal() {
  if (!requireWrite('当前角色只能查看值班安排，不能维护团队配置')) return
  refreshDutyTeamCounts()
  dutyTeamDrafts.value = dutyTeams.value.filter((team) => team.label !== '全部团队').map((team) => ({ original: team.label, label: team.label, count: team.count }))
  dutyTeamModalOpen.value = true
}

function addDutyTeamDraft() {
  dutyTeamDrafts.value.push({ original: '', label: `新运维组 ${dutyTeamDrafts.value.length + 1}`, count: 0 })
}

function removeDutyTeamDraft(index) {
  const target = dutyTeamDrafts.value[index]
  if (!target) return
  const referenced = Number(target.count || 0) > 0 || dutySchedules.value.some((item) => item.team === target.original) || dutyCurrentPeople.value.some((item) => item.team === target.original) || dutyEscalationForm.team === target.original
  if (referenced) {
    pushDutyToast('该团队仍被人员、排班或升级策略引用，请先迁移关联数据', 'warning')
    return
  }
  dutyTeamDrafts.value.splice(index, 1)
  pushDutyToast('空团队已移除', 'info')
}

async function saveDutyTeams() {
  const labels = dutyTeamDrafts.value.map((team) => team.label.trim()).filter(Boolean)
  if (!labels.length) return pushDutyToast('至少保留一个团队', 'warning')
  if (new Set(labels).size !== labels.length) return pushDutyToast('团队名称不能重复', 'warning')
  dutyTeamDrafts.value.forEach((team) => {
    const nextLabel = team.label.trim()
    if (!team.original || team.original === nextLabel) return
    dutyRoster.value.forEach((person) => { if (person.team === team.original) person.team = nextLabel })
    dutyCurrentPeople.value.forEach((person) => { if (person.team === team.original) person.team = nextLabel })
    dutySchedules.value.forEach((schedule) => { if (schedule.team === team.original) schedule.team = nextLabel })
    if (dutyEscalationForm.team === team.original) dutyEscalationForm.team = nextLabel
  })
  dutyTeams.value = [{ label: '全部团队', count: dutyRoster.value.length }, ...labels.map((label) => ({ label, count: 0 }))]
  if (!dutyTeams.value.some((team) => team.label === dutyTeamFilter.value)) dutyTeamFilter.value = '全部团队'
  refreshDutyTeamCounts()
  dutyTeamModalOpen.value = false
  await persistDutyCenter('团队配置已更新')
}

function dutySearchSubmit() {
  const keyword = dutyGlobalSearch.value.trim()
  if (!keyword) return
  const person = dutyRoster.value.find((item) => item.name.includes(keyword))
  if (person) {
    dutyRosterSearch.value = keyword
    dutySection.value = 'roster'
    pushDutyToast(`已定位到 ${person.name}`, 'info')
    return
  }
  const date = Object.entries(dutyAssignments).find(([, value]) => value.primary.includes(keyword) || value.backup.includes(keyword))
  if (date) {
    dutySelectedDate.value = date[0]
    dutySection.value = 'calendar'
    pushDutyToast(`已定位到 ${date[0]} 的排班`, 'info')
    return
  }
  pushDutyToast('未找到匹配的值班人员或排班', 'warning')
}

function moveDutyMonth(offset) {
  const next = new Date(dutyCalendarMonth.value)
  next.setMonth(next.getMonth() + offset)
  dutyCalendarMonth.value = next
}

function resetDutyCalendar() {
  dutyCalendarMonth.value = new Date(dutyToday.getFullYear(), dutyToday.getMonth(), 1)
  pushDutyToast('日历已重置到当前月份', 'info')
}

function openDutyAssignModal(date = dutySelectedDate.value, type = 'primary') {
  if (!requireWrite('当前角色只能查看值班安排，不能分配值班')) return
  const assignment = dutyAssignments[date] || {}
  const people = dutyPeopleOptions.value
  if (!people.length) return pushDutyToast('请先在值班列表中添加系统用户', 'warning')
  Object.assign(dutyAssignForm, { date, type, person: type === 'primary' ? (assignment.primary || people[0]) : (assignment.backup || people[1] || people[0]), note: '', rosterId: null })
  dutyAssignModalOpen.value = true
}

async function confirmDutyAssign() {
  if (!dutyAssignForm.date || !dutyAssignForm.person) return
  const existing = dutyAssignments[dutyAssignForm.date] || { primary: '', backup: '' }
  dutyAssignments[dutyAssignForm.date] = { ...existing, [dutyAssignForm.type]: dutyAssignForm.person }
  const rosterItem = dutyRoster.value.find((person) => person.id === dutyAssignForm.rosterId || person.name === dutyAssignForm.person)
  if (rosterItem) {
    rosterItem.status = '值班中'
    rosterItem.next = dutyAssignForm.date
    rosterItem.count = Number(rosterItem.count || 0) + 1
    if (!dutyCurrentPeople.value.some((person) => person.name === rosterItem.name)) {
      dutyCurrentPeople.value.push({ id: Date.now(), name: rosterItem.name, role: dutyAssignForm.type === 'primary' ? '主值班' : '备份值班', team: rosterItem.team, avatar: rosterItem.name.slice(0, 1), since: '08:00', until: '20:00', phone: '待补充', status: '在线' })
    }
    refreshDutyTeamCounts()
  }
  dutySelectedDate.value = dutyAssignForm.date
  dutyAssignModalOpen.value = false
  await persistDutyCenter(`${dutyAssignForm.person} 已分配为${dutyAssignForm.type === 'primary' ? '主值班' : '备份值班'}`)
}

function openCreateDutyScheduleModal() {
  if (!requireWrite('当前角色只能查看值班安排，不能创建排班模板')) return
  const firstTeam = dutyTeams.value.find((team) => team.label !== '全部团队')?.label || ''
  if (!firstTeam) return pushDutyToast('请先创建值班团队', 'warning')
  dutyScheduleModalMode.value = 'create'
  dutyEditingScheduleId.value = null
  Object.assign(dutyScheduleForm, { name: '', team: firstTeam, rotation: 'weekly', time: '08:00-20:00', members: [] })
  dutyScheduleModalOpen.value = true
}

async function saveDutyScheduleTemplate() {
  if (!dutyScheduleForm.name.trim() || !dutyScheduleForm.team || !dutyScheduleForm.members.length) return pushDutyToast('请填写排班名称、团队并选择值班成员', 'warning')
  const payload = { name: dutyScheduleForm.name.trim(), team: dutyScheduleForm.team, rotation: dutyScheduleForm.rotation, time: dutyScheduleForm.time || '08:00-20:00', members: [...dutyScheduleForm.members], active: true }
  if (dutyScheduleModalMode.value === 'edit') {
    const target = dutySchedules.value.find((item) => item.id === dutyEditingScheduleId.value)
    if (target) Object.assign(target, payload, { active: target.active })
  } else {
    dutySchedules.value.unshift({ id: Date.now(), ...payload })
  }
  dutyScheduleModalOpen.value = false
  await persistDutyCenter(dutyScheduleModalMode.value === 'edit' ? '排班模板已更新' : '新排班模板创建成功')
}

function editDutySchedule(schedule) {
  dutyScheduleModalMode.value = 'edit'
  dutyEditingScheduleId.value = schedule.id
  Object.assign(dutyScheduleForm, { name: schedule.name, team: schedule.team, rotation: schedule.rotation, time: schedule.time, members: [...schedule.members] })
  dutyScheduleModalOpen.value = true
}

async function toggleDutySchedule(schedule) {
  schedule.active = !schedule.active
  await persistDutyCenter(schedule.active ? '排班已启用' : '排班已停用')
}

async function deleteDutySchedule(schedule) {
  if (!schedule) return
  await props.requestConfirm({
    title: '删除排班模板',
    message: '删除后该排班规则不再用于后续值班生成。',
    target: `${schedule.team || '未命名团队'} · ${schedule.rotation === 'weekly' ? '按周轮换' : '按天轮换'}`,
    confirmLabel: '确认删除模板'
  }, async () => {
    dutySchedules.value = dutySchedules.value.filter((item) => item.id !== schedule.id)
    await persistDutyCenter('排班模板已删除')
  })
}

async function quickDutyTakeover() {
  if (!requireWrite('当前角色只能查看值班安排，不能执行接班')) return
  const person = dutyRoster.value.find((item) => item.status === '空闲')
  if (!person) return pushDutyToast('暂无可接班人员', 'warning')
  dutyCurrentPeople.value = [...dutyCurrentPeople.value, { id: Date.now(), name: person.name, role: '接班值班', team: person.team, avatar: person.name.slice(0, 1), since: '现在', until: '20:00', phone: '待补充', status: '在线' }]
  person.status = '值班中'
  person.next = dutyTodayKey
  await persistDutyCenter(`${person.name} 已接班`)
}

function quickAssignDutyPerson(person) {
  const targetDate = /^\d{4}-\d{2}-\d{2}$/.test(person.next) ? person.next : dutySelectedDate.value
  openDutyAssignModal(targetDate, 'primary')
  if (!dutyAssignModalOpen.value) return
  dutyAssignForm.person = person.name
  dutyAssignForm.rosterId = person.id
}

function openAddDutyPerson() {
  if (!requireWrite('当前角色只能查看值班安排，不能添加人员')) return
  const firstUser = dutySystemUserOptions.value[0]
  const firstTeam = dutyTeams.value.find((team) => team.label !== '全部团队')?.label || ''
  if (!firstTeam) return pushDutyToast('请先创建值班团队', 'warning')
  dutyEditingUserId.value = null
  Object.assign(dutyUserForm, { userId: firstUser?.id || '', team: firstTeam, role: firstUser?.role === 'super_admin' ? 'SRE 负责人' : '运维工程师', next: '待安排' })
  dutyUserModalOpen.value = true
}

function editDutyPerson(person) {
  if (!requireWrite('当前角色只能查看值班安排，不能编辑人员')) return
  dutyEditingUserId.value = person.id
  Object.assign(dutyUserForm, { userId: person.userId, team: person.team, role: person.role, next: person.next })
  dutyUserModalOpen.value = true
}

async function saveDutyUser() {
  const selected = dutySystemUserOptions.value.find((item) => String(item.id) === String(dutyUserForm.userId))
  if (!selected) return pushDutyToast('请选择系统用户', 'warning')
  if (dutyRoster.value.some((item) => item.id !== dutyEditingUserId.value && item.userId && String(item.userId) === String(selected.id))) return pushDutyToast('该系统用户已在值班列表中', 'warning')
  const existing = dutyRoster.value.find((item) => item.id === dutyEditingUserId.value)
  if (existing) {
    Object.assign(existing, { userId: selected.id, username: selected.username, name: selected.name, team: dutyUserForm.team, role: dutyUserForm.role, next: dutyUserForm.next || '待安排' })
    dutyCurrentPeople.value.forEach((person) => { if (person.name === existing.name) Object.assign(person, { team: existing.team, role: existing.role }) })
  } else {
    dutyRoster.value.push({ id: String(Date.now()), userId: selected.id, username: selected.username, name: selected.name, team: dutyUserForm.team, role: dutyUserForm.role, count: 0, next: dutyUserForm.next || '待安排', status: '空闲' })
  }
  refreshDutyTeamCounts()
  dutyUserModalOpen.value = false
  await persistDutyCenter(existing ? '值班人员已更新' : '系统用户已加入值班列表')
}

async function removeDutyPerson(person) {
  const referenced = dutyCurrentPeople.value.some((item) => item.name === person.name) || dutySchedules.value.some((item) => item.members.includes(person.name)) || Object.values(dutyAssignments).some((item) => item.primary === person.name || item.backup === person.name) || dutyHandovers.value.some((item) => item.from === person.name || item.to === person.name)
  if (referenced) return pushDutyToast('该人员仍在当前值班、排班模板或日历中，请先调整关联', 'warning')
  await props.requestConfirm({
    title: '移除值班人员',
    message: '仅从值班名单移除，不会删除对应系统账号。',
    target: `${person.name} · ${person.team}`,
    confirmLabel: '确认移除'
  }, async () => {
    dutyRoster.value = dutyRoster.value.filter((item) => item.id !== person.id)
    refreshDutyTeamCounts()
    await persistDutyCenter('值班人员已移除')
  })
}

function openDutyHandoverModal() {
  if (!requireWrite('当前角色只能查看值班安排，不能提交交接班')) return
  const people = dutyPeopleOptions.value
  if (people.length < 2) return pushDutyToast('至少需要两名值班人员才能提交交接', 'warning')
  Object.assign(dutyHandoverForm, { from: dutyCurrentPeople.value[0]?.name || people[0], to: dutyCurrentPeople.value[1]?.name || people[1], content: '', complete: true })
  dutyHandoverModalOpen.value = true
}

async function submitDutyHandover() {
  if (!dutyHandoverForm.from || !dutyHandoverForm.to || !dutyHandoverForm.content.trim()) return pushDutyToast('请选择交接双方并填写交接内容', 'warning')
  dutyHandovers.value.unshift({ id: Date.now(), from: dutyHandoverForm.from, to: dutyHandoverForm.to, time: `${dutyTodayKey} 20:00`, content: dutyHandoverForm.content || '系统运行正常，无未交接风险。', complete: dutyHandoverForm.complete })
  dutyHandoverModalOpen.value = false
  await persistDutyCenter('交接班记录已提交')
}

function openDutyEscalationModal() {
  if (!requireWrite('当前角色只能查看值班安排，不能编辑升级策略')) return
  const firstTeam = dutyTeams.value.find((team) => team.label !== '全部团队')?.label || ''
  if (!firstTeam) return pushDutyToast('请先创建值班团队', 'warning')
  if (!dutyEscalationForm.name) Object.assign(dutyEscalationForm, { name: 'P1 严重告警升级流程', team: firstTeam, severity: 'P1' })
  if (!dutyEscalationLevels.value.length) dutyEscalationLevels.value = [{ level: 1, target: '主值班', delay: '立即通知', channel: '电话 / 短信 / IM' }]
  dutyEscalationSnapshot.value = {
    form: { ...dutyEscalationForm },
    levels: dutyEscalationLevels.value.map((level) => ({ ...level }))
  }
  dutyEscalationModalOpen.value = true
}

function addDutyEscalationLevel() {
  dutyEscalationLevels.value.push({ level: dutyEscalationLevels.value.length + 1, target: '', delay: '', channel: '' })
}

function removeDutyEscalationLevel(index) {
  if (dutyEscalationLevels.value.length <= 1) return pushDutyToast('至少保留一个升级层级', 'warning')
  dutyEscalationLevels.value.splice(index, 1)
  dutyEscalationLevels.value.forEach((level, levelIndex) => { level.level = levelIndex + 1 })
}

function closeDutyEscalationModal() {
  if (dutyEscalationSnapshot.value) {
    Object.assign(dutyEscalationForm, dutyEscalationSnapshot.value.form)
    dutyEscalationLevels.value = dutyEscalationSnapshot.value.levels.map((level) => ({ ...level }))
  }
  dutyEscalationSnapshot.value = null
  dutyEscalationModalOpen.value = false
}

async function saveDutyEscalation() {
  if (!dutyEscalationForm.name.trim() || !dutyEscalationForm.team || !dutyEscalationLevels.value.length) return pushDutyToast('请完善策略名称、团队和升级层级', 'warning')
  dutyEscalationSnapshot.value = null
  dutyEscalationModalOpen.value = false
  await persistDutyCenter(`${dutyEscalationForm.severity} 升级策略已更新`)
}

function closeDutyTransientState() {
  dutyAssignModalOpen.value = false
  dutyScheduleModalOpen.value = false
  dutyHandoverModalOpen.value = false
  dutyUserModalOpen.value = false
  dutyTeamModalOpen.value = false
  closeDutyEscalationModal()
}

watch(() => props.active, (active, wasActive) => {
  if (wasActive && !active) closeDutyTransientState()
  if (active) loadDutyCenter()
})

refreshDutyTeamCounts()
if (props.active) loadDutyCenter()
</script>

<template>
  <section class="duty-center" role="region" aria-label="值班管理工作区">
    <div class="duty-integrated-head status-only">
      <div class="duty-status-strip">
        <span :class="['pill', dutyCurrentPeople.length ? 'success' : '']">{{ dutyCurrentPeople.length ? '当前值班已覆盖' : '当前未配置值班人员' }}</span>
        <span v-if="dutyRevision">数据版本 {{ dutyRevision }}</span>
        <span v-if="dutySaving">正在保存...</span>
      </div>
    </div>

    <div class="duty-tabbar">
      <button v-for="item in dutySections" :key="item.id" :class="{ active: dutySection === item.id }" @click="setDutySection(item.id)">{{ item.label }}</button>
    </div>

    <div class="duty-filter-strip">
      <label class="duty-search">
        <input v-model="dutyGlobalSearch" placeholder="搜索值班人员、排班或告警..." @keyup.enter="dutySearchSubmit" />
        <button @click="dutySearchSubmit">查询</button>
      </label>
      <div class="duty-team-tabs">
        <button v-for="team in dutyTeams" :key="team.label" :class="{ active: dutyTeamFilter === team.label }" @click="setDutyTeam(team.label)">{{ team.label }} <em>{{ team.count }}</em></button>
      </div>
      <button class="duty-team-config" :disabled="!canWrite" @click="openDutyTeamModal">团队配置</button>
    </div>

    <main class="duty-main">
      <section v-if="dutySection === 'overview'" class="duty-view">
        <div class="duty-page-head">
          <div><h2>值班概览</h2></div>
          <div class="duty-page-actions"><button class="primary" :disabled="!canWrite" @click="quickDutyTakeover">立即接班</button></div>
        </div>

        <div class="duty-stats">
          <article><small>当前值班人员</small><strong>{{ dutyFilteredCurrent.length }}</strong><span>{{ dutyFilteredCurrent.length ? '当前响应窗口已覆盖' : '需要安排接班' }}</span></article>
          <article><small>累计值班次数</small><strong>{{ dutyTotalCount }}</strong><span>依据值班人员记录</span></article>
          <article><small>启用排班模板</small><strong>{{ dutySchedules.filter((item) => item.active).length }}</strong><span>共 {{ dutySchedules.length }} 个模板</span></article>
          <article><small>待确认交接</small><strong>{{ dutyHandovers.filter((item) => !item.complete).length }}</strong><span>共 {{ dutyHandovers.length }} 条交接记录</span></article>
        </div>

        <div class="duty-overview-grid">
          <section class="duty-panel duty-current-panel">
            <div class="duty-panel-head"><div><h3>当前值班</h3></div><button @click="setDutySection('roster')">查看全部</button></div>
            <div class="duty-current-list">
              <article v-for="person in dutyFilteredCurrent" :key="person.id" class="duty-person-card">
                <span class="duty-avatar">{{ person.avatar }}</span>
                <div><strong>{{ person.name }}</strong><small>{{ person.role }} · {{ person.team }}</small><em>{{ person.since }}-{{ person.until }} · {{ person.phone }}</em></div>
                <b>{{ person.status }}</b>
              </article>
              <p v-if="!dutyFilteredCurrent.length" class="empty-state">暂无当前值班人员，请从值班列表安排值班。</p>
            </div>
          </section>

          <section class="duty-panel">
            <div class="duty-panel-head"><div><h3>即将值班（未来3天）</h3></div><button @click="setDutySection('calendar')">查看完整日历</button></div>
            <div class="duty-upcoming-list">
              <button v-for="item in dutyUpcomingAssignments" :key="item.date" @click="openDutyAssignModal(item.date, 'primary')"><span>{{ item.date }}</span><strong>{{ item.primary }}</strong><em>备份：{{ item.backup }}</em></button>
              <p v-if="!dutyUpcomingAssignments.length" class="empty-state">未来三天暂无排班。</p>
            </div>
          </section>
        </div>

        <section class="duty-panel">
          <div class="duty-panel-head"><div><h3>统计报表</h3></div><select><option>本月</option><option>本周</option><option>本季度</option></select></div>
          <div class="duty-report-grid">
            <article v-for="item in dutyReports" :key="item.label"><small>{{ item.label }}</small><strong>{{ item.value }}</strong><span>{{ item.detail }}</span><em><i :style="{ width: item.width }"></i></em></article>
          </div>
        </section>
      </section>

      <section v-if="dutySection === 'calendar'" class="duty-view">
        <div class="duty-page-head">
          <div><h2>排班日历</h2></div>
          <div class="duty-calendar-tools"><button @click="moveDutyMonth(-1)">上一月</button><strong>{{ dutyCalendarMonthLabel }}</strong><button @click="moveDutyMonth(1)">下一月</button><button @click="resetDutyCalendar">重置视图</button></div>
        </div>
        <div class="duty-calendar-legend"><span><i></i>主值班</span><span><i class="backup"></i>备份值班</span><button class="primary" :disabled="!canWrite" @click="openDutyAssignModal(dutyTodayKey, 'primary')">手动分配值班</button></div>
        <div class="duty-calendar">
          <strong v-for="label in weekdayLabels" :key="label" class="duty-week-label">{{ label }}</strong>
          <button v-for="cell in dutyCalendarCells" :key="cell.key" :class="['duty-day', { blank: cell.blank, today: cell.isToday, weekend: cell.isWeekend }]" :disabled="cell.blank" @click="!cell.blank && openDutyAssignModal(cell.key, 'primary')">
            <template v-if="!cell.blank"><span>{{ cell.day }}</span><em v-if="cell.primary">主 {{ cell.primary }}</em><em v-else class="empty">+ 分配</em><small v-if="cell.backup">备 {{ cell.backup }}</small></template>
          </button>
        </div>
      </section>

      <section v-if="dutySection === 'schedules'" class="duty-view">
        <div class="duty-page-head"><div><h2>排班配置</h2></div><button class="primary" :disabled="!canWrite" @click="openCreateDutyScheduleModal">新建排班</button></div>
        <div class="duty-schedule-grid">
          <article v-for="schedule in dutySchedules" :key="schedule.id" class="duty-schedule-card">
            <div><strong>{{ schedule.name }}</strong><span>{{ schedule.team }} · {{ schedule.rotation === 'weekly' ? '按周轮换' : '按天轮换' }}</span></div>
            <p>{{ schedule.time }} · {{ schedule.members.join(' / ') }}</p>
            <em :class="{ active: schedule.active }">{{ schedule.active ? '启用中' : '已停用' }}</em>
            <div class="duty-card-actions"><button :disabled="!canWrite" @click="editDutySchedule(schedule)">编辑</button><button :disabled="!canWrite" @click="toggleDutySchedule(schedule)">{{ schedule.active ? '停用' : '启用' }}</button><button class="danger-text" :disabled="!canWrite" @click="deleteDutySchedule(schedule)">删除</button></div>
          </article>
          <p v-if="!dutySchedules.length" class="empty-state">暂无排班模板。</p>
        </div>
      </section>

      <section v-if="dutySection === 'roster'" class="duty-view">
        <div class="duty-page-head"><div><h2>值班人员列表</h2></div><button class="primary" :disabled="!canWrite" @click="openAddDutyPerson">添加人员</button></div>
        <div class="duty-filter-row"><input v-model="dutyRosterSearch" placeholder="搜索姓名..." /></div>
        <div class="table-wrap duty-table-wrap">
          <table>
            <thead><tr><th>人员</th><th>团队</th><th>角色</th><th>本月值班</th><th>下次值班</th><th>状态</th><th>操作</th></tr></thead>
            <tbody>
              <tr v-for="person in dutyFilteredRoster" :key="person.id">
                <td><span class="duty-table-person"><b>{{ person.name.slice(0, 1) }}</b>{{ person.name }}</span></td><td>{{ person.team }}</td><td>{{ person.role }}</td><td>{{ person.count }}</td><td>{{ person.next }}</td><td><span :class="['pill', person.status === '值班中' ? 'success' : '']">{{ person.status }}</span></td>
                <td class="row-actions"><span class="inline-actions"><button class="link" :disabled="!canWrite" @click.stop="quickAssignDutyPerson(person)">安排值班</button><button class="link" :disabled="!canWrite" @click.stop="editDutyPerson(person)">编辑</button><button class="link danger-text" :disabled="!canWrite" @click.stop="removeDutyPerson(person)">移除</button></span></td>
              </tr>
              <tr v-if="!dutyFilteredRoster.length"><td colspan="7" class="empty-state">暂无值班人员。</td></tr>
            </tbody>
          </table>
        </div>
      </section>

      <section v-if="dutySection === 'handover'" class="duty-view">
        <div class="duty-page-head"><div><h2>交接班日志</h2></div><button class="primary" :disabled="!canWrite" @click="openDutyHandoverModal">提交交接</button></div>
        <div class="duty-handover-list">
          <article v-for="item in dutyHandovers" :key="item.id"><div><strong>{{ item.from }} → {{ item.to }}</strong><small>{{ item.time }}</small></div><p>{{ item.content }}</p><span>{{ item.complete ? '交接完成' : '待确认' }}</span></article>
          <p v-if="!dutyHandovers.length" class="empty-state">暂无交接班记录。</p>
        </div>
      </section>

      <section v-if="dutySection === 'escalation'" class="duty-view">
        <div class="duty-page-head"><div><h2>升级策略</h2></div><button class="primary" :disabled="!canWrite" @click="openDutyEscalationModal">编辑策略</button></div>
        <section class="duty-panel">
          <div class="duty-panel-head"><div><h3>{{ dutyEscalationForm.name }}（{{ dutyEscalationForm.team }}）</h3></div></div>
          <div class="duty-escalation-flow">
            <template v-for="(level, index) in dutyEscalationLevels" :key="level.level"><article><small>Level {{ level.level }}</small><strong>{{ level.target }}</strong><span>{{ level.delay }} · {{ level.channel }}</span></article><i v-if="index < dutyEscalationLevels.length - 1"></i></template>
          </div>
        </section>
      </section>
    </main>

    <div class="duty-toast-stack"><div v-for="toast in dutyToasts" :key="toast.id" :class="['duty-toast', toast.tone]">{{ toast.message }}</div></div>

    <div v-if="dutyAssignModalOpen" class="duty-modal-backdrop">
      <section class="duty-modal" role="dialog" aria-modal="true" aria-labelledby="duty-assign-title">
        <div class="duty-modal-head"><h3 id="duty-assign-title">分配值班</h3><button aria-label="关闭分配值班" @click="dutyAssignModalOpen = false">×</button></div>
        <label>值班日期<input v-model="dutyAssignForm.date" type="date" /></label>
        <div class="duty-radio-row"><label><input v-model="dutyAssignForm.type" type="radio" value="primary" />主值班</label><label><input v-model="dutyAssignForm.type" type="radio" value="backup" />备份值班</label></div>
        <label>值班人员<select v-model="dutyAssignForm.person"><option v-for="person in dutyPeopleOptions" :key="person">{{ person }}</option></select></label>
        <label>备注<textarea v-model="dutyAssignForm.note" placeholder="填写特殊说明或覆盖范围"></textarea></label>
        <div class="duty-modal-actions"><button @click="dutyAssignModalOpen = false">取消</button><button class="primary" @click="confirmDutyAssign">确认分配</button></div>
      </section>
    </div>

    <div v-if="dutyScheduleModalOpen" class="duty-modal-backdrop">
      <section class="duty-modal" role="dialog" aria-modal="true" aria-labelledby="duty-schedule-title">
        <div class="duty-modal-head"><h3 id="duty-schedule-title">{{ dutyScheduleModalMode === 'edit' ? '编辑排班模板' : '新建排班模板' }}</h3><button aria-label="关闭排班模板" @click="dutyScheduleModalOpen = false">×</button></div>
        <label>模板名称<input v-model="dutyScheduleForm.name" /></label>
        <label>团队<select v-model="dutyScheduleForm.team"><option v-for="team in dutyTeams.filter((item) => item.label !== '全部团队')" :key="team.label">{{ team.label }}</option></select></label>
        <label>轮换周期<select v-model="dutyScheduleForm.rotation"><option value="weekly">按周轮换</option><option value="daily">按天轮换</option></select></label>
        <label>值班时段<input v-model="dutyScheduleForm.time" /></label>
        <label>值班成员<select v-model="dutyScheduleForm.members" multiple><option v-for="person in dutyPeopleOptions" :key="person">{{ person }}</option></select></label>
        <p class="muted">成员预览：{{ dutyScheduleForm.members.join(' / ') || '未选择成员' }}</p>
        <div class="duty-modal-actions"><button @click="dutyScheduleModalOpen = false">取消</button><button class="primary" @click="saveDutyScheduleTemplate">{{ dutyScheduleModalMode === 'edit' ? '保存修改' : '创建排班' }}</button></div>
      </section>
    </div>

    <div v-if="dutyUserModalOpen" class="duty-modal-backdrop">
      <section class="duty-modal" role="dialog" aria-modal="true" aria-labelledby="duty-user-title">
        <div class="duty-modal-head"><h3 id="duty-user-title">{{ dutyEditingUserId ? '编辑值班人员' : '添加值班人员' }}</h3><button aria-label="关闭添加值班人员" @click="dutyUserModalOpen = false">×</button></div>
        <label>关联系统用户<select v-model="dutyUserForm.userId" :disabled="Boolean(dutyEditingUserId)"><option value="">请选择系统用户</option><option v-for="user in dutySystemUserOptions" :key="user.id" :value="user.id">{{ user.name }}（{{ user.username }}）</option></select></label>
        <label>所属团队<select v-model="dutyUserForm.team"><option v-for="team in dutyTeams.filter((item) => item.label !== '全部团队')" :key="team.label">{{ team.label }}</option></select></label>
        <label>值班角色<select v-model="dutyUserForm.role"><option>SRE</option><option>SRE 负责人</option><option>运维工程师</option><option>网络工程师</option><option>DBA</option><option>开发工程师</option></select></label>
        <label>下次值班<input v-model="dutyUserForm.next" placeholder="例如：YYYY-MM-DD 或 待安排" /></label>
        <p class="muted">值班人员应绑定系统用户，后续可复用账号状态、角色权限、通知渠道和审计记录。</p>
        <div class="duty-modal-actions"><button @click="dutyUserModalOpen = false">取消</button><button class="primary" @click="saveDutyUser">{{ dutyEditingUserId ? '保存修改' : '添加人员' }}</button></div>
      </section>
    </div>

    <div v-if="dutyTeamModalOpen" class="duty-modal-backdrop">
      <section class="duty-modal duty-modal-wide" role="dialog" aria-modal="true" aria-labelledby="duty-team-title">
        <div class="duty-modal-head"><h3 id="duty-team-title">团队配置</h3><button aria-label="关闭团队配置" @click="dutyTeamModalOpen = false">×</button></div>
        <p class="muted">维护值班团队名称。保存后会同步更新人员列表、当前值班和排班模板中的团队归属。</p>
        <div class="duty-team-editor">
          <article v-for="(team, index) in dutyTeamDrafts" :key="team.original || index"><label>团队名称<input v-model="team.label" /></label><span>{{ team.count }} 人</span><button class="duty-team-delete" @click="removeDutyTeamDraft(index)">删除</button></article>
        </div>
        <button class="duty-secondary-action" @click="addDutyTeamDraft">新增团队</button>
        <div class="duty-modal-actions"><button @click="dutyTeamModalOpen = false">取消</button><button class="primary" @click="saveDutyTeams">保存团队</button></div>
      </section>
    </div>

    <div v-if="dutyHandoverModalOpen" class="duty-modal-backdrop">
      <section class="duty-modal" role="dialog" aria-modal="true" aria-labelledby="duty-handover-title">
        <div class="duty-modal-head"><h3 id="duty-handover-title">提交交接班</h3><button aria-label="关闭提交交接班" @click="dutyHandoverModalOpen = false">×</button></div>
        <label>交接人<select v-model="dutyHandoverForm.from"><option v-for="person in dutyPeopleOptions" :key="person">{{ person }}</option></select></label>
        <label>接班人<select v-model="dutyHandoverForm.to"><option v-for="person in dutyPeopleOptions" :key="person">{{ person }}</option></select></label>
        <label>交接内容<textarea v-model="dutyHandoverForm.content" placeholder="填写系统状态、未完成事项、风险和注意事项"></textarea></label>
        <label class="inline-check"><input v-model="dutyHandoverForm.complete" type="checkbox" />交接已完成，系统运行正常</label>
        <div class="duty-modal-actions"><button @click="dutyHandoverModalOpen = false">取消</button><button class="primary" @click="submitDutyHandover">提交交接</button></div>
      </section>
    </div>

    <div v-if="dutyEscalationModalOpen" class="duty-modal-backdrop">
      <section class="duty-modal duty-modal-wide" role="dialog" aria-modal="true" aria-labelledby="duty-escalation-title">
        <div class="duty-modal-head"><h3 id="duty-escalation-title">编辑升级策略</h3><button aria-label="关闭升级策略" @click="closeDutyEscalationModal">×</button></div>
        <label>策略名称<input v-model="dutyEscalationForm.name" /></label>
        <label>团队<select v-model="dutyEscalationForm.team"><option v-for="team in dutyTeams.filter((item) => item.label !== '全部团队')" :key="team.label">{{ team.label }}</option></select></label>
        <label>告警等级<select v-model="dutyEscalationForm.severity"><option>P1</option><option>P2</option><option>P3</option><option>P4</option></select></label>
        <div class="duty-level-editor">
          <article v-for="(level, index) in dutyEscalationLevels" :key="level.level"><strong>Level {{ level.level }}</strong><label>通知对象<input v-model="level.target" /></label><label>升级时间<input v-model="level.delay" /></label><label>通知渠道<input v-model="level.channel" /></label><button class="danger-text" @click="removeDutyEscalationLevel(index)">删除层级</button></article>
        </div>
        <button class="duty-secondary-action" @click="addDutyEscalationLevel">新增升级层级</button>
        <div class="duty-modal-actions"><button @click="closeDutyEscalationModal">取消</button><button class="primary" @click="saveDutyEscalation">保存策略</button></div>
      </section>
    </div>
  </section>
</template>
