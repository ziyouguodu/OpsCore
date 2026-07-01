import assert from 'node:assert/strict'
import test from 'node:test'

let helpers = {}
try {
  helpers = await import('../src/duty-date.js')
} catch {
  helpers = {}
}

test('builds duty assignments and upcoming rows relative to the current date', () => {
  assert.equal(typeof helpers.toISODate, 'function')
  assert.equal(typeof helpers.createDutyAssignments, 'function')
  assert.equal(typeof helpers.getUpcomingAssignments, 'function')

  const today = new Date(2026, 5, 29)
  const assignments = helpers.createDutyAssignments(today, [
    ['张伟', '李娜'],
    ['陈明', '刘芳']
  ], 5)

  assert.deepEqual(Object.keys(assignments), [
    '2026-06-29',
    '2026-06-30',
    '2026-07-01',
    '2026-07-02',
    '2026-07-03'
  ])
  assert.deepEqual(helpers.getUpcomingAssignments(assignments, today, 3), [
    { date: '2026-06-30', primary: '陈明', backup: '刘芳' },
    { date: '2026-07-01', primary: '张伟', backup: '李娜' },
    { date: '2026-07-02', primary: '陈明', backup: '刘芳' }
  ])
})

test('marks the actual current day in the monthly calendar', () => {
  assert.equal(typeof helpers.buildDutyCalendarCells, 'function')
  const today = new Date(2026, 5, 29)
  const cells = helpers.buildDutyCalendarCells(new Date(2026, 5, 1), {}, today)
  const current = cells.find((cell) => cell.isToday)

  assert.equal(current?.key, '2026-06-29')
  assert.equal(current?.day, 29)
})

test('formats relative duty dates without fixed calendar values', () => {
  assert.equal(typeof helpers.offsetISODate, 'function')
  const today = new Date(2026, 11, 31)

  assert.equal(helpers.offsetISODate(today, 1), '2027-01-01')
  assert.equal(helpers.offsetISODate(today, -7), '2026-12-24')
})

test('does not report a perfect duty balance when no roster data exists', () => {
  assert.equal(helpers.calculateDutyBalance([]), null)
  assert.equal(helpers.calculateDutyBalance([4, 4]), 100)
  assert.equal(helpers.calculateDutyBalance([4, 2]), 50)
})
