<script setup>
import { computed, nextTick, reactive, ref, watch } from 'vue'
import ContactPhone from './ContactPhone.vue'
import DutyDialog from './DutyDialog.vue'
import WorkspacePageHeader from './WorkspacePageHeader.vue'
import SvgIcon from './SvgIcon.vue'
import { api } from '../api'
import { buildDutyCalendarCells, calculateDutyBalance, createDutyAssignments, getUpcomingAssignments, toISODate } from '../duty-date'

const props = defineProps({
  active: { type: Boolean, default: false },
  canWrite: { type: Boolean, default: false },
  requestConfirm: { type: Function, required: true },
  users: { type: Array, default: () => [] }
})

const emit = defineEmits(['error', 'saved'])

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
const dutyLoading = ref(false)
const dutyLoadError = ref('')
const dutySaveError = ref('')
const dutyCommitted = ref(null)
const dutyTakeoverModalOpen = ref(false)
const dutyTakeoverForm = reactive({ person: '', type: 'primary' })
const dutyPendingCount = computed(() => dutyHandovers.value.filter(item => !item.complete).length)
const dutyPrimary = computed(() => dutySlot('primary'))
const dutyBackup = computed(() => dutySlot('backup'))
function dutySlot(type) {
  const current = dutyFilteredCurrent.value.find(person => type === 'primary' ? ['主值班', 'primary'].includes(person.role) : person.role?.startsWith('备') || person.role === 'backup')
  if (current) return current
  const name = dutyAssignments[dutyTodayKey]?.[type]
  const member = dutyRoster.value.find(person => person.name === name && (dutyTeamFilter.value === '全部团队' || person.team === dutyTeamFilter.value))
  return member ? { ...member, since: '', until: '', status: '已排班' } : null
}
const dutySelectedAssignment = computed(() => dutyAssignments[dutySelectedDate.value] || {})
const dutyFilteredSchedules = computed(() => dutySchedules.value.filter(item => dutyTeamFilter.value === '全部团队' || item.team === dutyTeamFilter.value))
const dutyFilteredHandovers = computed(() => dutyHandovers.value.filter(item => dutyTeamFilter.value === '全部团队' || dutyRoster.value.some(person => person.team === dutyTeamFilter.value && [item.from, item.to].includes(person.name))))
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

const dutyAssignForm = reactive({ date: dutyTodayKey, type: 'primary', person: '', rosterId: null })
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
const dutyMemberByName = computed(() => new Map(dutyRoster.value.map(person => [person.name, person])))
function dutyContactLabel(name) {
  const person = dutyMemberByName.value.get(name)
  return `${name} · ${person?.phone || '未配置联系方式'}`
}

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
const dutySystemUserOptions = computed(() => props.users.map((user) => ({
  id: user.id,
  username: user.username,
  name: user.displayName || user.username,
  role: (user.roles || [])[0] || 'ops_engineer'
})))

function serializeDutyData() {
  return {
    teams: dutyTeams.value.filter((team) => team.label !== '全部团队').map((team) => ({ name: team.label })),
    members: dutyRoster.value.map(({ phone, ...person }) => ({ ...person, id: String(person.id) })),
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

async function loadDutyCenter(force = false, duringSave = false) {
  if (dutyLoaded.value && !force) return
  if (dutyLoading.value || (dutySaving.value && !duringSave)) return
  dutyLoading.value = true
  dutyLoadError.value = ''
  try {
    const saved = await api('/duty-center')
    applyDutyState(saved)
    dutyCommitted.value = JSON.parse(JSON.stringify(saved))
  } catch (err) {
    dutyLoadError.value = `值班数据加载失败：${err.message}`
  } finally {
    dutyLoading.value = false
  }
}

async function persistDutyCenter(message, update) {
  if (!props.canWrite || dutySaving.value || dutyLoading.value || !dutyLoaded.value) return false
  const before = dutyCommitted.value
  dutySaving.value = true
  dutySaveError.value = ''
  try {
    if (update) update()
    const payload = serializeDutyData()
    const saved = await api('/duty-center', {
      method: 'PUT',
      body: JSON.stringify({ revision: dutyRevision.value, data: payload })
    })
    applyDutyState(saved)
    dutyCommitted.value = JSON.parse(JSON.stringify(saved))
    if (message) pushDutyToast(message)
    emit('saved')
    return true
  } catch (err) {
    if (before) applyDutyState(before)
    dutySaveError.value = err.status === 409 ? '值班数据已被其他人修改，已刷新最新版本；请核对输入后重试。' : `值班数据保存失败：${err.message}`
    if (err.status === 409) await loadDutyCenter(true, true)
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
  if (dutySections.some((item) => item.id === section)) {
    dutySection.value = section
    dutySaveError.value = ''
  }
}

async function moveDutyTab(event, index) {
  const offsets = { ArrowRight: 1, ArrowLeft: -1, Home: -index, End: dutySections.length - 1 - index }
  if (!(event.key in offsets)) return
  event.preventDefault()
  const next = (index + offsets[event.key] + dutySections.length) % dutySections.length
  setDutySection(dutySections[next].id)
  await nextTick()
  document.getElementById(`duty-tab-${dutySections[next].id}`)?.focus()
}

function closeDutyDialog(close) {
  if (dutySaving.value) return
  dutySaveError.value = ''
  close()
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
  if (!props.canWrite) {
    emit('error', message)
    return false
  }
  if (dutySaving.value || dutyLoading.value || !dutyLoaded.value) return false
  dutySaveError.value = ''
  return true
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
  const saved = await persistDutyCenter('团队配置已更新', () => {
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
  })
  if (saved) dutyTeamModalOpen.value = false
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
  Object.assign(dutyAssignForm, { date, type, person: type === 'primary' ? (assignment.primary || people[0]) : (assignment.backup || people[1] || people[0]), rosterId: null })
  dutyAssignModalOpen.value = true
}

function updateCurrentDuty(person, type, since = '', until = '') {
  const isSlot = item => type === 'primary' ? ['主值班', 'primary'].includes(item.role) : item.role?.startsWith('备') || item.role === 'backup'
  const previous = dutyCurrentPeople.value.find(isSlot)
  dutyCurrentPeople.value = dutyCurrentPeople.value.filter(item => !isSlot(item) && item.name !== person.name)
  dutyCurrentPeople.value.push({ id: String(person.id), name: person.name, role: type === 'primary' ? '主值班' : '备份值班', team: person.team, since: since || previous?.since || '', until: until || previous?.until || '', phone: person.phone || '', status: '在线' })
  const assignment = dutyAssignments[dutyTodayKey] || { primary: '', backup: '' }
  const otherType = type === 'primary' ? 'backup' : 'primary'
  dutyAssignments[dutyTodayKey] = { ...assignment, [otherType]: assignment[otherType] === person.name ? '' : assignment[otherType], [type]: person.name }
  dutyRoster.value.forEach(item => { item.status = dutyCurrentPeople.value.some(current => current.name === item.name) ? '值班中' : '空闲' })
}

async function confirmDutyAssign() {
  if (!dutyAssignForm.date || !dutyAssignForm.person) return
  const existing = dutyAssignments[dutyAssignForm.date] || { primary: '', backup: '' }
  const other = dutyAssignForm.type === 'primary' ? existing.backup : existing.primary
  if (other === dutyAssignForm.person) return pushDutyToast('主值班和备班不能是同一人', 'warning')
  const saved = await persistDutyCenter('值班分配已保存', () => {
    dutyAssignments[dutyAssignForm.date] = { ...existing, [dutyAssignForm.type]: dutyAssignForm.person }
    const rosterItem = dutyRoster.value.find(person => person.name === dutyAssignForm.person)
    if (rosterItem) {
      rosterItem.next = dutyAssignForm.date
      if (existing[dutyAssignForm.type] !== rosterItem.name) rosterItem.count = Number(rosterItem.count || 0) + 1
      if (dutyAssignForm.date === dutyTodayKey) updateCurrentDuty(rosterItem, dutyAssignForm.type)
    }
    dutySelectedDate.value = dutyAssignForm.date
    refreshDutyTeamCounts()
  })
  if (saved) dutyAssignModalOpen.value = false
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
  const saved = await persistDutyCenter(dutyScheduleModalMode.value === 'edit' ? '排班模板已更新' : '新排班模板创建成功', () => {
    if (dutyScheduleModalMode.value === 'edit') {
      const target = dutySchedules.value.find((item) => item.id === dutyEditingScheduleId.value)
      if (target) Object.assign(target, payload, { active: target.active })
    } else dutySchedules.value.unshift({ id: String(Date.now()), ...payload })
  })
  if (saved) dutyScheduleModalOpen.value = false
}

function editDutySchedule(schedule) {
  dutyScheduleModalMode.value = 'edit'
  dutyEditingScheduleId.value = schedule.id
  Object.assign(dutyScheduleForm, { name: schedule.name, team: schedule.team, rotation: schedule.rotation, time: schedule.time, members: [...schedule.members] })
  dutyScheduleModalOpen.value = true
}

async function toggleDutySchedule(schedule) {
  await persistDutyCenter(schedule.active ? '排班已停用' : '排班已启用', () => { schedule.active = !schedule.active })
}

async function deleteDutySchedule(schedule) {
  if (!schedule) return
  await props.requestConfirm({
    title: '删除排班模板',
    message: '删除后该排班规则不再用于后续值班生成。',
    target: `${schedule.team || '未命名团队'} · ${schedule.rotation === 'weekly' ? '按周轮换' : '按天轮换'}`,
    confirmLabel: '确认删除模板'
  }, async () => {
    await persistDutyCenter('排班模板已删除', () => { dutySchedules.value = dutySchedules.value.filter((item) => item.id !== schedule.id) })
  })
}

function quickDutyTakeover() {
  if (!requireWrite('当前角色只能查看值班安排，不能执行接班')) return
  const person = dutyFilteredRoster.value.find(item => item.status === '空闲')
  if (!person) return pushDutyToast('暂无可接班人员，请先维护值班名单', 'warning')
  Object.assign(dutyTakeoverForm, { person: person.name, type: 'primary' })
  dutyTakeoverModalOpen.value = true
}

async function confirmDutyTakeover() {
  const person = dutyRoster.value.find(item => item.name === dutyTakeoverForm.person)
  if (!person) return pushDutyToast('请选择接班人员', 'warning')
  const other = dutyCurrentPeople.value.find(item => dutyTakeoverForm.type === 'primary' ? item.role?.startsWith('备') || item.role === 'backup' : ['主值班', 'primary'].includes(item.role))
  if (other?.name === person.name) return pushDutyToast('主值班和备班不能是同一人', 'warning')
  const saved = await persistDutyCenter('接班已确认', () => {
    updateCurrentDuty(person, dutyTakeoverForm.type, new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false }))
    const assignment = dutyAssignments[dutyTodayKey] || { primary: '', backup: '' }
    dutyAssignments[dutyTodayKey] = { ...assignment, [dutyTakeoverForm.type]: person.name }
    person.next = dutyTodayKey
  })
  if (saved) dutyTakeoverModalOpen.value = false
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
  const saved = await persistDutyCenter(existing ? '值班人员已更新' : '系统用户已加入值班列表', () => {
    if (existing) {
      Object.assign(existing, { userId: selected.id, username: selected.username, name: selected.name, team: dutyUserForm.team, role: dutyUserForm.role, next: dutyUserForm.next || '待安排' })
      dutyCurrentPeople.value.forEach((person) => { if (person.name === existing.name) person.team = existing.team })
    } else {
      dutyRoster.value.push({ id: String(Date.now()), userId: selected.id, username: selected.username, name: selected.name, team: dutyUserForm.team, role: dutyUserForm.role, count: 0, next: dutyUserForm.next || '待安排', status: '空闲' })
    }
    refreshDutyTeamCounts()
  })
  if (saved) dutyUserModalOpen.value = false
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
    await persistDutyCenter('值班人员已移除', () => {
      dutyRoster.value = dutyRoster.value.filter((item) => item.id !== person.id)
      refreshDutyTeamCounts()
    })
  })
}

function openDutyHandoverModal() {
  if (!requireWrite('当前角色只能查看值班安排，不能提交交接班')) return
  const people = dutyPeopleOptions.value
  if (people.length < 2) return pushDutyToast('至少需要两名值班人员才能提交交接', 'warning')
  const from = dutyPrimary.value?.name || dutyCurrentPeople.value[0]?.name || people[0]
  Object.assign(dutyHandoverForm, { from, to: people.find(name => name !== from) || '', content: '', complete: true })
  dutyHandoverModalOpen.value = true
}

async function submitDutyHandover() {
  if (!dutyHandoverForm.from || !dutyHandoverForm.to || !dutyHandoverForm.content.trim()) return pushDutyToast('请选择交接双方并填写交接内容', 'warning')
  if (dutyHandoverForm.from === dutyHandoverForm.to) return pushDutyToast('交接人和接班人不能相同', 'warning')
  if (dutyHandoverForm.complete && !dutyCurrentPeople.value.some(person => person.name === dutyHandoverForm.from)) return pushDutyToast('交接人不在当前班次，请核对后再确认完成', 'warning')
  if (dutyHandoverForm.complete && dutyCurrentPeople.value.some(person => person.name === dutyHandoverForm.to)) return pushDutyToast('接班人已在当前其他岗位，请先调整班次后再交接', 'warning')
  const saved = await persistDutyCenter('交接班记录已提交', () => {
    const item = { id: String(Date.now()), from: dutyHandoverForm.from, to: dutyHandoverForm.to, time: new Date().toLocaleString('sv-SE', { hour12: false }), content: dutyHandoverForm.content.trim(), complete: dutyHandoverForm.complete }
    dutyHandovers.value.unshift(item)
    if (item.complete) applyDutyHandover(item)
  })
  if (saved) dutyHandoverModalOpen.value = false
}

function applyDutyHandover(item) {
  const outgoing = dutyCurrentPeople.value.find(person => person.name === item.from)
  const incoming = dutyRoster.value.find(person => person.name === item.to)
  if (!outgoing || !incoming) return
  const type = outgoing.role?.startsWith('备') || outgoing.role === 'backup' ? 'backup' : 'primary'
  updateCurrentDuty(incoming, type, new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false }), outgoing.until)
  dutyAssignments[dutyTodayKey] = { ...(dutyAssignments[dutyTodayKey] || { primary: '', backup: '' }), [type]: incoming.name }
}

async function confirmDutyHandover(item) {
  if (!props.canWrite || dutySaving.value || dutyLoading.value) return
  if (!dutyCurrentPeople.value.some(person => person.name === item.from)) return pushDutyToast('原交班人已不在当前班次，请核对后重新提交交接', 'warning')
  if (dutyCurrentPeople.value.some(person => person.name === item.to)) return pushDutyToast('接班人已在当前其他岗位，请先调整班次后再交接', 'warning')
  await props.requestConfirm({ title: '确认交接', message: '确认后将同步当前班次的接班人员。', target: `${item.from} → ${item.to}`, confirmLabel: '确认交接' }, async () => {
    await persistDutyCenter('交接已确认', () => { item.complete = true; applyDutyHandover(item) })
  })
}

function openDutyEscalationModal() {
  if (!requireWrite('当前角色只能查看值班安排，不能编辑升级策略')) return
  const firstTeam = dutyTeams.value.find((team) => team.label !== '全部团队')?.label || ''
  if (!firstTeam) return pushDutyToast('请先创建值班团队', 'warning')
  dutyEscalationSnapshot.value = {
    form: { ...dutyEscalationForm },
    levels: dutyEscalationLevels.value.map((level) => ({ ...level }))
  }
  if (!dutyEscalationForm.name) Object.assign(dutyEscalationForm, { name: 'P1 严重告警升级流程', team: firstTeam, severity: 'P1' })
  if (!dutyEscalationLevels.value.length) dutyEscalationLevels.value = [{ level: 1, target: '主值班', delay: '立即通知', channel: '电话 / 短信 / IM' }]
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
    const policy = dutyCommitted.value?.data.escalation
    Object.assign(dutyEscalationForm, policy || dutyEscalationSnapshot.value.form)
    dutyEscalationLevels.value = (policy?.levels || dutyEscalationSnapshot.value.levels).map(level => ({ ...level }))
  }
  dutyEscalationSnapshot.value = null
  dutyEscalationModalOpen.value = false
}

async function saveDutyEscalation() {
  if (!dutyEscalationForm.name.trim() || !dutyEscalationForm.team || !dutyEscalationLevels.value.length) return pushDutyToast('请完善策略名称、团队和升级层级', 'warning')
  if (dutyEscalationLevels.value.some(level => !level.target.trim() || !level.delay.trim() || !level.channel.trim())) return pushDutyToast('请完整填写每一级的通知对象、升级时间和通知渠道', 'warning')
  const draft = { form: { ...dutyEscalationForm }, levels: dutyEscalationLevels.value.map(level => ({ ...level })) }
  const saved = await persistDutyCenter(`${dutyEscalationForm.severity} 升级策略已更新`)
  if (saved) {
    dutyEscalationSnapshot.value = null
    dutyEscalationModalOpen.value = false
  } else {
    Object.assign(dutyEscalationForm, draft.form)
    dutyEscalationLevels.value = draft.levels
  }
}

function closeDutyTransientState() {
  dutyTakeoverModalOpen.value = false
  dutyAssignModalOpen.value = false
  dutyScheduleModalOpen.value = false
  dutyHandoverModalOpen.value = false
  dutyUserModalOpen.value = false
  dutyTeamModalOpen.value = false
  closeDutyEscalationModal()
}

watch(() => props.active, (active, wasActive) => {
  if (wasActive && !active) closeDutyTransientState()
  if (active) loadDutyCenter(true)
})

refreshDutyTeamCounts()
if (props.active) loadDutyCenter()
</script>

<template>
  <section class="duty-center" role="region" aria-label="值班管理工作区">
    <WorkspacePageHeader v-if="active" view="oncall" />
    <div class="duty-workspace-header">
      <div class="duty-status-strip" role="status"><span class="pill" :class="{ success: dutyPrimary && dutyBackup }">{{ dutyPrimary && dutyBackup ? '主备值班已覆盖' : dutyPrimary ? '备班待安排' : '主值班待安排' }}</span><span v-if="dutyRevision">数据版本 {{ dutyRevision }}</span><span v-if="!canWrite" class="pill">只读</span><span v-if="dutySaving">正在保存...</span></div>
      <div class="duty-toolbar-actions"><button :disabled="dutySaving || dutyLoading" @click="loadDutyCenter(true)"><SvgIcon name="activity" />刷新</button><button :disabled="!canWrite || dutySaving || dutyLoading || !dutyLoaded" @click="openDutyTeamModal">团队配置</button></div>
    </div>
    <div class="duty-tabbar" role="tablist" aria-label="值班工作区" :inert="dutySaving || undefined">
      <button v-for="(item, index) in dutySections" :id="`duty-tab-${item.id}`" :key="item.id" role="tab" :aria-selected="dutySection === item.id" :aria-controls="`duty-panel-${item.id}`" :tabindex="dutySection === item.id ? 0 : -1" :class="{ active: dutySection === item.id }" @click="setDutySection(item.id)" @keydown="moveDutyTab($event, index)">{{ item.label }}</button>
    </div>
    <div v-if="dutyLoadError" class="duty-workspace-error" role="alert">{{ dutyLoadError }}<button @click="loadDutyCenter(true)">重试</button></div>
    <p v-if="dutyLoading" role="status">正在加载值班数据…</p>
    <div v-if="dutySaveError" class="duty-workspace-error" role="alert">{{ dutySaveError }}</div>
    <div class="duty-filter-strip" :inert="dutySaving || undefined">
      <label>团队范围<select v-model="dutyTeamFilter" aria-label="值班团队筛选"><option v-for="team in dutyTeams" :key="team.label" :value="team.label">{{ team.label }} · {{ team.count }} 人</option></select></label>
      <form class="duty-search" @submit.prevent="dutySearchSubmit"><input v-model="dutyGlobalSearch" aria-label="搜索值班人员或排班" placeholder="搜索值班人员或排班" /><button type="submit">查询</button></form>
    </div>
    <main class="duty-main" :inert="dutySaving || undefined">
      <section v-if="dutySection === 'overview'" id="duty-panel-overview" class="duty-view" role="tabpanel" aria-labelledby="duty-tab-overview">
        <div class="duty-page-head"><h2>当班与交接</h2><div class="duty-page-actions"><button :disabled="!canWrite || dutySaving || dutyLoading" @click="openDutyHandoverModal">提交交接</button><button class="primary" :disabled="!canWrite || dutySaving || dutyLoading" @click="quickDutyTakeover">立即接班</button></div></div>
        <div class="duty-shift-grid">
          <article v-for="slot in [{ label: '主值班', person: dutyPrimary, type: 'primary' }, { label: '备班', person: dutyBackup, type: 'backup' }]" :key="slot.type" class="duty-role-card" :class="slot.type">
            <div class="role-card-head"><span>{{ slot.label }}</span><span class="pill" :class="{ success: slot.person }">{{ slot.person ? (slot.person.status === '已排班' ? '已排班' : '当班') : '未安排' }}</span></div>
            <strong class="role-person">{{ slot.person?.name || '暂无值班人员' }}</strong>
            <p class="role-meta">{{ slot.person?.team || '尚无团队归属' }}<span v-if="slot.person"> · {{ slot.person.since || '--' }}–{{ slot.person.until || '--' }}</span></p>
            <ContactPhone :phone="slot.person?.phone || ''" :name="slot.person?.name || slot.label" />
            <button class="duty-empty-action" :disabled="!canWrite || dutySaving || dutyLoading" @click="openDutyAssignModal(dutyTodayKey, slot.type)">{{ slot.person ? '调整班次' : '安排值班' }}</button>
          </article>
        </div>
        <div class="duty-stats"><article><small>值班成员</small><strong>{{ dutyFilteredRoster.length }}</strong><span>当前团队范围</span></article><article><small>启用排班</small><strong>{{ dutyFilteredSchedules.filter(item => item.active).length }}</strong><span>已保存模板</span></article><article><small>待确认交接</small><strong>{{ dutyPendingCount }}</strong><span>全局待办</span></article><article><small>值班均衡度</small><strong>{{ dutyTotalCount ? `${calculateDutyBalance(dutyRoster.map(person => Number(person.count || 0)))}%` : '--' }}</strong><span>依据累计分配次数</span></article></div>
        <div class="duty-overview-grid">
          <section class="duty-panel"><div class="duty-panel-head"><h3>未来 3 天</h3><button @click="setDutySection('calendar')">查看完整日历</button></div><div class="duty-upcoming-list"><button v-for="item in dutyUpcomingAssignments" :key="item.date" @click="dutySelectedDate = item.date; dutyCalendarMonth = new Date(`${item.date}T12:00:00`); setDutySection('calendar')"><span>{{ item.date }}</span><strong>主值班 {{ item.primary || '未安排' }}</strong><em>备班 {{ item.backup || '未安排' }}</em></button><p v-if="!dutyUpcomingAssignments.length" class="empty-state">未来三天暂无排班。<button @click="setDutySection('calendar')">查看排班日历</button></p></div></section>
          <section class="duty-panel"><div class="duty-panel-head"><h3>交接待办</h3><button @click="setDutySection('handover')">查看交接日志</button></div><div class="duty-handover-preview"><article v-for="item in dutyHandovers.filter(item => !item.complete).slice(0, 3)" :key="item.id"><strong>{{ item.from }} → {{ item.to }}</strong><p>{{ item.content }}</p><small>{{ item.time }}</small></article><p v-if="!dutyPendingCount" class="empty-state">暂无待确认交接</p></div></section>
        </div>
      </section>

      <section v-if="dutySection === 'calendar'" id="duty-panel-calendar" class="duty-view" role="tabpanel" aria-labelledby="duty-tab-calendar">
        <div class="duty-page-head">
          <div><h2>排班日历</h2></div>
          <div class="duty-calendar-tools"><button @click="moveDutyMonth(-1)">上一月</button><strong>{{ dutyCalendarMonthLabel }}</strong><button @click="moveDutyMonth(1)">下一月</button><button @click="resetDutyCalendar">重置视图</button></div>
        </div>
        <div class="duty-calendar-legend"><span><i></i>主值班</span><span><i class="backup"></i>备份值班</span><button class="primary" :disabled="!canWrite || dutySaving || dutyLoading || !dutyLoaded" @click="openDutyAssignModal(dutyTodayKey, 'primary')">手动分配值班</button></div>
        <div class="duty-calendar-layout"><div class="duty-calendar-grid-wrap"><div class="duty-calendar">
          <strong v-for="label in weekdayLabels" :key="label" class="duty-week-label">{{ label }}</strong>
          <button v-for="cell in dutyCalendarCells" :key="cell.key" :aria-label="cell.blank ? '' : `${cell.key} 主值班 ${cell.primary || '未安排'} 备班 ${cell.backup || '未安排'}`" :aria-pressed="dutySelectedDate === cell.key" :class="['duty-day', { blank: cell.blank, today: cell.isToday, weekend: cell.isWeekend, selected: dutySelectedDate === cell.key }]" :disabled="cell.blank" @click="dutySelectedDate = cell.key">
            <template v-if="!cell.blank"><span>{{ cell.day }}<small v-if="cell.isToday">今日</small></span><em>主 {{ cell.primary || '未安排' }}</em><small>备 {{ cell.backup || '未安排' }}</small></template>
          </button>
        </div></div><aside class="duty-day-detail"><h3>{{ dutySelectedDate }}</h3><dl><dt>主值班</dt><dd>{{ dutySelectedAssignment.primary || '未安排' }}<ContactPhone :phone="dutyMemberByName.get(dutySelectedAssignment.primary)?.phone || ''" :name="dutySelectedAssignment.primary || '主值班'" /></dd><dt>备班</dt><dd>{{ dutySelectedAssignment.backup || '未安排' }}<ContactPhone :phone="dutyMemberByName.get(dutySelectedAssignment.backup)?.phone || ''" :name="dutySelectedAssignment.backup || '备班'" /></dd></dl><button :disabled="!canWrite || dutySaving || dutyLoading" @click="openDutyAssignModal(dutySelectedDate, 'primary')">调整主值班</button><button :disabled="!canWrite || dutySaving || dutyLoading" @click="openDutyAssignModal(dutySelectedDate, 'backup')">调整备班</button><p v-if="!canWrite" class="muted">当前角色仅可查看排班</p></aside></div>
      </section>

      <section v-if="dutySection === 'schedules'" id="duty-panel-schedules" class="duty-view" role="tabpanel" aria-labelledby="duty-tab-schedules">
        <div class="duty-page-head"><h2>排班模板 <small>{{ dutyFilteredSchedules.length }} 个</small></h2><button class="primary" :disabled="!canWrite || dutySaving || dutyLoading" @click="openCreateDutyScheduleModal">新建排班</button></div>
        <div class="table-wrap duty-template-table"><table><thead><tr><th>模板名称</th><th>团队</th><th>轮换周期</th><th>时段</th><th>成员</th><th>状态</th><th>操作</th></tr></thead><tbody><tr v-for="schedule in dutyFilteredSchedules" :key="schedule.id"><td>{{ schedule.name }}</td><td>{{ schedule.team }}</td><td>{{ schedule.rotation === 'weekly' ? '按周轮换' : '按天轮换' }}</td><td>{{ schedule.time }}</td><td>{{ schedule.members.join(' / ') }}</td><td><span class="pill" :class="{ success: schedule.active }">{{ schedule.active ? '启用中' : '已停用' }}</span></td><td class="row-actions"><span class="inline-actions"><button class="link" :disabled="!canWrite || dutySaving || dutyLoading" @click="editDutySchedule(schedule)">编辑</button><button class="link" :disabled="!canWrite || dutySaving || dutyLoading" @click="toggleDutySchedule(schedule)">{{ schedule.active ? '停用' : '启用' }}</button><button class="link danger-text" :disabled="!canWrite || dutySaving || dutyLoading" @click="deleteDutySchedule(schedule)">删除</button></span></td></tr><tr v-if="!dutyFilteredSchedules.length"><td colspan="7" class="empty-state">当前团队暂无排班模板</td></tr></tbody></table></div>
      </section>

      <section v-if="dutySection === 'roster'" id="duty-panel-roster" class="duty-view" role="tabpanel" aria-labelledby="duty-tab-roster">
        <div class="duty-page-head"><div><h2>值班人员列表</h2></div><button class="primary" :disabled="!canWrite || dutySaving || dutyLoading || !dutyLoaded" @click="openAddDutyPerson">添加人员</button></div>
        <div class="duty-filter-row"><input v-model="dutyRosterSearch" placeholder="搜索姓名..." /></div>
        <div class="table-wrap duty-table-wrap">
          <table>
            <thead><tr><th>人员</th><th>联系电话</th><th>团队</th><th>角色</th><th>本月值班</th><th>下次值班</th><th>状态</th><th>操作</th></tr></thead>
            <tbody>
              <tr v-for="person in dutyFilteredRoster" :key="person.id">
                <td><span class="duty-table-person"><b>{{ person.name.slice(0, 1) }}</b>{{ person.name }}</span></td><td><ContactPhone :phone="person.phone || ''" :name="person.name" /></td><td>{{ person.team }}</td><td>{{ person.role }}</td><td>{{ person.count }}</td><td>{{ person.next }}</td><td><span :class="['pill', person.status === '值班中' ? 'success' : '']">{{ person.status }}</span></td>
                <td class="row-actions"><span class="inline-actions"><button class="link" :disabled="!canWrite || dutySaving || dutyLoading || !dutyLoaded" @click.stop="quickAssignDutyPerson(person)">安排值班</button><button class="link" :disabled="!canWrite || dutySaving || dutyLoading || !dutyLoaded" @click.stop="editDutyPerson(person)">编辑</button><button class="link danger-text" :disabled="!canWrite || dutySaving || dutyLoading || !dutyLoaded" @click.stop="removeDutyPerson(person)">移除</button></span></td>
              </tr>
              <tr v-if="!dutyFilteredRoster.length"><td colspan="8" class="empty-state">暂无值班人员。</td></tr>
            </tbody>
          </table>
        </div>
      </section>

      <section v-if="dutySection === 'handover'" id="duty-panel-handover" class="duty-view" role="tabpanel" aria-labelledby="duty-tab-handover">
        <div class="duty-page-head"><div><h2>交接班日志</h2></div><button class="primary" :disabled="!canWrite || dutySaving || dutyLoading || !dutyLoaded" @click="openDutyHandoverModal">提交交接</button></div>
        <div class="duty-handover-list">
          <article v-for="item in dutyFilteredHandovers" :key="item.id"><div><strong>{{ item.from }} → {{ item.to }}</strong><small>{{ item.time }}</small></div><p>{{ item.content }}</p><div class="inline-actions"><span class="pill" :class="{ success: item.complete }">{{ item.complete ? '交接完成' : '待确认' }}</span><button v-if="!item.complete" :disabled="!canWrite || dutySaving || dutyLoading" @click="confirmDutyHandover(item)">确认交接</button></div></article>
          <p v-if="!dutyFilteredHandovers.length" class="empty-state">当前团队暂无交接班记录。</p>
        </div>
      </section>

      <section v-if="dutySection === 'escalation'" id="duty-panel-escalation" class="duty-view" role="tabpanel" aria-labelledby="duty-tab-escalation">
        <div class="duty-page-head"><div><h2>升级策略</h2></div><button class="primary" :disabled="!canWrite || dutySaving || dutyLoading || !dutyLoaded" @click="openDutyEscalationModal">编辑策略</button></div>
        <section v-if="dutyEscalationLevels.length" class="duty-panel">
          <div class="duty-policy-meta"><h3>{{ dutyEscalationForm.name }}</h3><span class="pill">{{ dutyEscalationForm.team }}</span><span class="pill">{{ dutyEscalationForm.severity }}</span><span class="muted">仅配置 · 尚未接入通知执行</span></div>
          <div class="duty-escalation-flow">
            <article v-for="level in dutyEscalationLevels" :key="level.level"><small>层级 {{ level.level }}</small><strong>{{ level.target }}</strong><span>{{ level.delay }} · {{ level.channel }}</span></article>
          </div>
        </section><p v-else class="empty-state">暂无升级策略，配置后可查看通知对象与升级顺序。</p>
      </section>
    </main>

    <div class="duty-toast-stack" role="status" aria-live="polite"><div v-for="toast in dutyToasts" :key="toast.id" :class="['duty-toast', toast.tone]">{{ toast.message }}</div></div>

    <DutyDialog :open="dutyAssignModalOpen" :busy="dutySaving" @close="closeDutyDialog(() => dutyAssignModalOpen = false)">
      <section class="duty-modal" role="dialog" aria-modal="true" aria-labelledby="duty-assign-title">
        <div class="duty-modal-head"><h3 id="duty-assign-title">分配值班</h3><button :disabled="dutySaving || dutyLoading" aria-label="关闭分配值班" @click="dutyAssignModalOpen = false">×</button></div>
        <fieldset :disabled="dutySaving || dutyLoading"><p v-if="dutySaveError" class="duty-workspace-error" role="alert">{{ dutySaveError }}</p><label>值班日期<input v-model="dutyAssignForm.date" type="date" /></label>
        <div class="duty-radio-row"><label><input v-model="dutyAssignForm.type" type="radio" value="primary" />主值班</label><label><input v-model="dutyAssignForm.type" type="radio" value="backup" />备份值班</label></div>
        <label>值班人员<select v-model="dutyAssignForm.person"><option v-for="person in dutyPeopleOptions" :key="person" :value="person">{{ dutyContactLabel(person) }}</option></select></label>
        <p class="muted">仅调整所选日期的岗位；未来排班不会立即改变当前值班。</p>
        <div class="duty-modal-actions"><button @click="dutyAssignModalOpen = false">取消</button><button class="primary" @click="confirmDutyAssign">确认分配</button></div>
      </fieldset></section>
    </DutyDialog>

    <DutyDialog :open="dutyScheduleModalOpen" :busy="dutySaving" @close="closeDutyDialog(() => dutyScheduleModalOpen = false)">
      <section class="duty-modal" role="dialog" aria-modal="true" aria-labelledby="duty-schedule-title">
        <div class="duty-modal-head"><h3 id="duty-schedule-title">{{ dutyScheduleModalMode === 'edit' ? '编辑排班模板' : '新建排班模板' }}</h3><button :disabled="dutySaving || dutyLoading" aria-label="关闭排班模板" @click="dutyScheduleModalOpen = false">×</button></div>
        <fieldset :disabled="dutySaving || dutyLoading"><p v-if="dutySaveError" class="duty-workspace-error" role="alert">{{ dutySaveError }}</p><label>模板名称<input v-model="dutyScheduleForm.name" /></label>
        <label>团队<select v-model="dutyScheduleForm.team"><option v-for="team in dutyTeams.filter((item) => item.label !== '全部团队')" :key="team.label">{{ team.label }}</option></select></label>
        <label>轮换周期<select v-model="dutyScheduleForm.rotation"><option value="weekly">按周轮换</option><option value="daily">按天轮换</option></select></label>
        <label>值班时段<input v-model="dutyScheduleForm.time" /></label>
        <label>值班成员<select v-model="dutyScheduleForm.members" multiple><option v-for="person in dutyPeopleOptions" :key="person" :value="person">{{ dutyContactLabel(person) }}</option></select></label>
        <p class="muted">成员预览：{{ dutyScheduleForm.members.join(' / ') || '未选择成员' }}</p>
        <div class="duty-modal-actions"><button @click="dutyScheduleModalOpen = false">取消</button><button class="primary" @click="saveDutyScheduleTemplate">{{ dutyScheduleModalMode === 'edit' ? '保存修改' : '创建排班' }}</button></div>
      </fieldset></section>
    </DutyDialog>

    <DutyDialog :open="dutyUserModalOpen" :busy="dutySaving" @close="closeDutyDialog(() => dutyUserModalOpen = false)">
      <section class="duty-modal" role="dialog" aria-modal="true" aria-labelledby="duty-user-title">
        <div class="duty-modal-head"><h3 id="duty-user-title">{{ dutyEditingUserId ? '编辑值班人员' : '添加值班人员' }}</h3><button :disabled="dutySaving || dutyLoading" aria-label="关闭添加值班人员" @click="dutyUserModalOpen = false">×</button></div>
        <fieldset :disabled="dutySaving || dutyLoading"><p v-if="dutySaveError" class="duty-workspace-error" role="alert">{{ dutySaveError }}</p><label>关联系统用户<select v-model="dutyUserForm.userId" :disabled="Boolean(dutyEditingUserId)"><option value="">请选择系统用户</option><option v-for="user in dutySystemUserOptions" :key="user.id" :value="user.id">{{ user.name }}（{{ user.username }}）</option></select></label>
        <label>所属团队<select v-model="dutyUserForm.team"><option v-for="team in dutyTeams.filter((item) => item.label !== '全部团队')" :key="team.label">{{ team.label }}</option></select></label>
        <label>值班角色<select v-model="dutyUserForm.role"><option>SRE</option><option>SRE 负责人</option><option>运维工程师</option><option>网络工程师</option><option>DBA</option><option>开发工程师</option></select></label>
        <label>下次值班<input v-model="dutyUserForm.next" placeholder="例如：YYYY-MM-DD 或 待安排" /></label>
        <p class="muted">值班人员应绑定系统用户，后续可复用账号状态、角色权限、通知渠道和审计记录。</p>
        <div class="duty-modal-actions"><button @click="dutyUserModalOpen = false">取消</button><button class="primary" @click="saveDutyUser">{{ dutyEditingUserId ? '保存修改' : '添加人员' }}</button></div>
      </fieldset></section>
    </DutyDialog>

    <DutyDialog :open="dutyTeamModalOpen" :busy="dutySaving" @close="closeDutyDialog(() => dutyTeamModalOpen = false)">
      <section class="duty-modal duty-modal-wide" role="dialog" aria-modal="true" aria-labelledby="duty-team-title">
        <div class="duty-modal-head"><h3 id="duty-team-title">团队配置</h3><button :disabled="dutySaving || dutyLoading" aria-label="关闭团队配置" @click="dutyTeamModalOpen = false">×</button></div>
        <fieldset :disabled="dutySaving || dutyLoading"><p v-if="dutySaveError" class="duty-workspace-error" role="alert">{{ dutySaveError }}</p><p class="muted">维护值班团队名称。保存后会同步更新人员列表、当前值班和排班模板中的团队归属。</p>
        <div class="duty-team-editor">
          <article v-for="(team, index) in dutyTeamDrafts" :key="team.original || index"><label>团队名称<input v-model="team.label" /></label><span>{{ team.count }} 人</span><button class="duty-team-delete" @click="removeDutyTeamDraft(index)">删除</button></article>
        </div>
        <button class="duty-secondary-action" @click="addDutyTeamDraft">新增团队</button>
        <div class="duty-modal-actions"><button @click="dutyTeamModalOpen = false">取消</button><button class="primary" @click="saveDutyTeams">保存团队</button></div>
      </fieldset></section>
    </DutyDialog>

    <DutyDialog :open="dutyHandoverModalOpen" :busy="dutySaving" @close="closeDutyDialog(() => dutyHandoverModalOpen = false)">
      <section class="duty-modal" role="dialog" aria-modal="true" aria-labelledby="duty-handover-title">
        <div class="duty-modal-head"><h3 id="duty-handover-title">提交交接班</h3><button :disabled="dutySaving || dutyLoading" aria-label="关闭提交交接班" @click="dutyHandoverModalOpen = false">×</button></div>
        <fieldset :disabled="dutySaving || dutyLoading"><p v-if="dutySaveError" class="duty-workspace-error" role="alert">{{ dutySaveError }}</p><label>交接人<select v-model="dutyHandoverForm.from"><option v-for="person in dutyPeopleOptions" :key="person" :value="person">{{ dutyContactLabel(person) }}</option></select></label>
        <label>接班人<select v-model="dutyHandoverForm.to"><option v-for="person in dutyPeopleOptions" :key="person" :value="person">{{ dutyContactLabel(person) }}</option></select></label>
        <label>交接内容<textarea v-model="dutyHandoverForm.content" placeholder="填写系统状态、未完成事项、风险和注意事项"></textarea></label>
        <label class="inline-check"><input v-model="dutyHandoverForm.complete" type="checkbox" />交接已完成，系统运行正常</label>
        <div class="duty-modal-actions"><button @click="dutyHandoverModalOpen = false">取消</button><button class="primary" @click="submitDutyHandover">提交交接</button></div>
      </fieldset></section>
    </DutyDialog>

    <DutyDialog :open="dutyEscalationModalOpen" :busy="dutySaving" @close="closeDutyDialog(closeDutyEscalationModal)">
      <section class="duty-modal duty-modal-wide" role="dialog" aria-modal="true" aria-labelledby="duty-escalation-title">
        <div class="duty-modal-head"><h3 id="duty-escalation-title">编辑升级策略</h3><button :disabled="dutySaving || dutyLoading" aria-label="关闭升级策略" @click="closeDutyEscalationModal">×</button></div>
        <fieldset :disabled="dutySaving || dutyLoading"><p v-if="dutySaveError" class="duty-workspace-error" role="alert">{{ dutySaveError }}</p><label>策略名称<input v-model="dutyEscalationForm.name" /></label>
        <label>团队<select v-model="dutyEscalationForm.team"><option v-for="team in dutyTeams.filter((item) => item.label !== '全部团队')" :key="team.label">{{ team.label }}</option></select></label>
        <label>告警等级<select v-model="dutyEscalationForm.severity"><option>P1</option><option>P2</option><option>P3</option><option>P4</option></select></label>
        <div class="duty-level-editor">
          <article v-for="(level, index) in dutyEscalationLevels" :key="level.level"><strong>Level {{ level.level }}</strong><label>通知对象<input v-model="level.target" /></label><label>升级时间<input v-model="level.delay" /></label><label>通知渠道<input v-model="level.channel" /></label><button class="danger-text" @click="removeDutyEscalationLevel(index)">删除层级</button></article>
        </div>
        <button class="duty-secondary-action" @click="addDutyEscalationLevel">新增升级层级</button>
        <div class="duty-modal-actions"><button @click="closeDutyEscalationModal">取消</button><button class="primary" @click="saveDutyEscalation">保存策略</button></div>
      </fieldset></section>
    </DutyDialog>
    <DutyDialog :open="dutyTakeoverModalOpen" :busy="dutySaving" @close="closeDutyDialog(() => dutyTakeoverModalOpen = false)"><section class="duty-modal" role="dialog" aria-modal="true" aria-labelledby="duty-takeover-title"><div class="duty-modal-head"><h3 id="duty-takeover-title">确认接班</h3><button :disabled="dutySaving || dutyLoading" aria-label="关闭接班" @click="dutyTakeoverModalOpen = false">×</button></div><fieldset :disabled="dutySaving || dutyLoading"><label>接班岗位<select v-model="dutyTakeoverForm.type"><option value="primary">主值班</option><option value="backup">备班</option></select></label><label>接班人员<select v-model="dutyTakeoverForm.person"><option v-for="person in dutyFilteredRoster" :key="person.id" :value="person.name">{{ person.name }} · {{ person.team }} · {{ person.phone || '未配置联系方式' }}</option></select></label><p class="muted">确认后更新当前岗位和今日排班，不会执行任何生产处置。</p></fieldset><p v-if="dutySaveError" role="alert">{{ dutySaveError }}</p><div class="duty-modal-actions"><button :disabled="dutySaving || dutyLoading" @click="dutyTakeoverModalOpen = false">取消</button><button class="primary" :disabled="dutySaving || dutyLoading" @click="confirmDutyTakeover">{{ dutySaving ? '保存中…' : '确认接班' }}</button></div></section></DutyDialog>
  </section>
</template>
