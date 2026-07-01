export const roleCards = [
  { code: 'super_admin', name: '超级管理员', icon: 'shield', tone: 'blue', desc: '拥有用户、角色、资产、协同事件、敏感凭据策略和系统配置全部权限。' },
  { code: 'ops_engineer', name: '运维工程师', icon: 'change', tone: 'green', desc: '可维护资产和实例信息，查看值班情况，处理任务，并跟进事件。' }
]
export const permissionRows = [
  { scope: '用户与角色管理', admin: '全部', ops: '-', note: '仅超级管理员配置账号、角色和菜单' },
  { scope: '资产台账（CMDB）', admin: '全部', ops: '查看 / 修改', note: '运维工程师可查看与配置修改' },
  { scope: '中间件与数据库', admin: '全部', ops: '查看 / 修改', note: '实例基础信息与关联资产管理' },
  { scope: '资产登录信息查看', admin: '校验后查看', ops: '-', note: '查看前输入超级管理员设置的校验密码' },
  { scope: '值班情况', admin: '全部', ops: '查看', note: '运维工程师可查看值班安排' },
  { scope: '任务处理', admin: '全部', ops: '处理', note: '处理本人或分派任务' },
  { scope: '事件跟进', admin: '全部', ops: '跟进', note: '查看协同事件并更新进展' }
]

export const menuPermissionRows = [
  { menu: '首页健康总览', stage: '一期启用', superAdmin: '可查看', opsEngineer: '可查看', note: '聚合健康、影响、值班和待办' },
  { menu: '资产台账（CMDB）', stage: '一期启用', superAdmin: '全部', opsEngineer: '查看 / 修改', note: '登录信息需额外校验权限' },
  { menu: '中间件与数据库', stage: '一期启用', superAdmin: '全部', opsEngineer: '查看 / 修改', note: '实例账号密码默认不可见' },
  { menu: '值班管理', stage: '一期启用', superAdmin: '全部', opsEngineer: '查看', note: '新增、编辑、删除仅超级管理员' },
  { menu: '任务跟踪', stage: '一期启用', superAdmin: '全部', opsEngineer: '处理', note: '支持状态流转与闭环跟进' },
  { menu: '事件管理', stage: '一期启用', superAdmin: '全部', opsEngineer: '跟进', note: '支持 P1-P4 与恢复关闭流转' },
  { menu: 'SSO/LDAP、审批审计、SRE、可观测性', stage: '灰度占位', superAdmin: '灰度', opsEngineer: '-', note: '只展示入口，不开放业务操作' }
]

export const copilotProviders = [
  { id: 'local', name: '本地模型', badge: 'Local', endpoint: 'http://host.docker.internal:11434', models: ['qwen2.5:7b', 'deepseek-r1:7b', 'llama3.1:8b'], desc: '适合内网部署、敏感数据不出域和离线推理场景；Docker 栈访问宿主机 Ollama 时使用 host.docker.internal。' },
  { id: 'openai', name: 'OpenAI GPT', badge: 'GPT', endpoint: 'https://api.openai.com/v1', models: ['gpt-4.1', 'gpt-4o', 'o4-mini'], desc: '适合综合分析、复杂推理和通用 Copilot 能力。' },
  { id: 'anthropic', name: 'Anthropic Claude', badge: 'Claude', endpoint: 'https://api.anthropic.com', models: ['claude-3-7-sonnet', 'claude-3-5-sonnet'], desc: '适合长上下文总结、变更评审和事件复盘。' },
  { id: 'google', name: 'Google Gemini', badge: 'Gemini', endpoint: 'https://generativelanguage.googleapis.com', models: ['gemini-1.5-pro', 'gemini-1.5-flash'], desc: '适合多模态、知识检索和轻量分析场景。' },
  { id: 'compatible', name: 'OpenAI 兼容接口', badge: 'API', endpoint: 'https://llm.example.com/v1', models: ['custom-chat', 'ops-model'], desc: '适配私有云网关、代理平台或统一模型路由。' }
]
