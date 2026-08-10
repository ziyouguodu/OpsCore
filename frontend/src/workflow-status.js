const taskTransitions = Object.freeze({
  '待处理': ['处理中', '已关闭'],
  '处理中': ['待确认', '已关闭'],
  '待确认': ['已完成', '处理中', '已关闭'],
  '已完成': ['已关闭'],
  '已关闭': []
})

const incidentTransitions = Object.freeze({
  '新建': ['处理中', '已关闭'],
  '处理中': ['已恢复', '已关闭'],
  '已恢复': ['已关闭', '处理中'],
  '已关闭': []
})

function statusOptions(transitions, current) {
  return [current, ...(transitions[current] || [])]
}

export function taskStatusOptions(current) {
  return statusOptions(taskTransitions, current)
}

export function incidentStatusOptions(current) {
  return statusOptions(incidentTransitions, current)
}
