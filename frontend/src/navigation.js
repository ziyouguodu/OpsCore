export const routeViews = ['dashboard', 'cmdb', 'middleware', 'oncall', 'tasks', 'incidents', 'permissions', 'audit', 'copilot-settings']
export const permissionTabs = ['users', 'resources']

export const controlPrinciple = '首页看健康，异常看影响，告警看根因，处置看流程，复盘看改进，AI 贯穿查询、分析、建议和自动化。'

export const viewMeta = {
  dashboard: {
    title: '首页健康总览',
    breadcrumb: '工作台',
    subtitle: '围绕业务连续性汇总健康、影响、待办和事件风险。'
  },
  cmdb: {
    title: '资产台账（CMDB）',
    breadcrumb: '资产管理',
    subtitle: '维护服务器资产、部署信息、网络区域和受控登录信息。'
  },
  middleware: {
    title: '中间件与数据库',
    breadcrumb: '资产管理',
    subtitle: '管理数据库、中间件与组件实例，补齐部署位置和影响范围。'
  },
  oncall: {
    title: '值班管理',
    breadcrumb: '协同与事件响应',
    subtitle: '面向事件响应连续性，统一管理当前值班、排班日历、交接日志与升级策略。'
  },
  tasks: {
    title: '任务跟踪',
    breadcrumb: '协同与事件响应',
    subtitle: '跟踪派发、处理、确认、完成和关闭的任务闭环。'
  },
  incidents: {
    title: '事件管理',
    breadcrumb: '协同与事件响应',
    subtitle: '按 P1-P4 管理影响、状态、恢复和复盘动作。'
  },
  permissions: {
    title: '权限管理',
    breadcrumb: '权限管理',
    subtitle: '管理一期角色、账号密码登录、菜单资源授权和敏感凭据查看边界。'
  },
  audit: {
    title: '操作审计',
    breadcrumb: '权限管理',
    subtitle: '追踪登录、权限、凭据、值班、任务和事件等关键操作。'
  },
  'copilot-settings': {
    title: 'AI Copilot 配置',
    breadcrumb: '系统配置',
    subtitle: '配置 AI Copilot 的模型来源、上下文权限、审计和受控自动化边界。'
  }
}

export const menu = [
  { id: 'dashboard', label: '首页仪表盘', icon: 'dashboard', enabled: true },
  { label: '资产管理', icon: 'server', children: [
    { id: 'cmdb', label: '资产台账（CMDB）', icon: 'cmdb', enabled: true },
    { id: 'middleware', label: '中间件与数据库', icon: 'database', enabled: true }
  ] },
  { label: '协同与事件响应', icon: 'collab', children: [
    { id: 'incidents', label: '事件管理', icon: 'incident', enabled: true },
    { id: 'oncall', label: '值班管理', icon: 'calendar', enabled: true },
    { id: 'tasks', label: '任务跟踪', icon: 'task', enabled: true },
    { label: 'SLA 管理', icon: 'activity', enabled: false }
  ] },
  { label: 'AI 运维', icon: 'copilot', children: [
    { label: 'AI Copilot', icon: 'copilot', action: 'copilot', enabled: true },
    { label: '根因分析', icon: 'activity', enabled: false }
  ] },
  { label: '权限管理', icon: 'shield', children: [
    { id: 'permissions', label: '用户与角色', icon: 'users', enabled: true, permissionTab: 'users' },
    { id: 'permissions', label: '菜单与资源权限', icon: 'lock', enabled: true, permissionTab: 'resources' },
    { id: 'audit', label: '操作审计', icon: 'audit', enabled: true }
  ] },
  { label: '系统配置', icon: 'settings', children: [
    { id: 'copilot-settings', label: 'AI Copilot 配置', icon: 'settings', enabled: true },
    { label: '通知渠道', icon: 'bell', enabled: false }
  ] },
  { label: '监控与可观测性', icon: 'eye', enabled: false },
  { label: '告警中心', icon: 'bell', enabled: false },
  { label: '任务与自动化', icon: 'change', enabled: false }
]
