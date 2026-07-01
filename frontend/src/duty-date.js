export function toISODate(date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export function offsetISODate(anchorDate, offset) {
  const date = new Date(anchorDate)
  date.setDate(date.getDate() + offset)
  return toISODate(date)
}

export function createDutyAssignments(anchorDate, rotations, days = 10) {
  const assignments = {}
  if (!rotations.length || days <= 0) return assignments

  for (let offset = 0; offset < days; offset += 1) {
    const date = new Date(anchorDate)
    date.setDate(date.getDate() + offset)
    const [primary, backup] = rotations[offset % rotations.length]
    assignments[toISODate(date)] = { primary, backup }
  }
  return assignments
}

export function getUpcomingAssignments(assignments, today, limit = 3) {
  const todayKey = toISODate(today)
  return Object.entries(assignments)
    .filter(([date]) => date > todayKey)
    .sort(([a], [b]) => a.localeCompare(b))
    .slice(0, limit)
    .map(([date, value]) => ({ date, ...value }))
}

export function buildDutyCalendarCells(monthDate, assignments, today) {
  const year = monthDate.getFullYear()
  const month = monthDate.getMonth()
  const first = new Date(year, month, 1)
  const last = new Date(year, month + 1, 0)
  const todayKey = toISODate(today)
  const cells = Array.from({ length: first.getDay() }, (_, index) => ({ key: `blank-${index}`, blank: true }))

  for (let day = 1; day <= last.getDate(); day += 1) {
    const date = new Date(year, month, day)
    const key = toISODate(date)
    const assignment = assignments[key]
    cells.push({
      key,
      day,
      isToday: key === todayKey,
      isWeekend: date.getDay() === 0 || date.getDay() === 6,
      primary: assignment?.primary || '',
      backup: assignment?.backup || ''
    })
  }
  return cells
}

export function calculateDutyBalance(counts) {
  if (!counts.length) return null
  const max = Math.max(...counts)
  if (max === 0) return 100
  const min = Math.min(...counts)
  return Math.max(0, Math.round((1 - (max - min) / max) * 100))
}
