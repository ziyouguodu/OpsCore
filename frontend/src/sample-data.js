export const sampleAssets = [
  {
    id: 'sample-asset-1',
    assetNo: 'ASSET-DEMO-0001',
    type: '物理机',
    vendor: 'Dell',
    cpuArch: 'x86_64',
    sn: 'DEMO-SN-001',
    location: 'A 区 03 柜',
    business: '支付服务',
    ipv4: '10.20.3.11',
    ipv6: '',
    environment: '生产',
    os: 'Ubuntu 22.04',
    hostname: 'pay-core-01',
    networkZone: 'prod-app',
    cpu: '16C',
    memory: '64GB',
    disk: '1TB',
    deploymentInfo: '支付核心 API',
    owner: '李明',
    status: '运行中',
    hostMachine: '',
    __sample: true
  },
  {
    id: 'sample-asset-2',
    assetNo: 'ASSET-DEMO-0002',
    type: '虚拟机',
    vendor: 'VMware',
    cpuArch: 'x86_64',
    sn: '',
    location: '',
    business: '订单服务',
    ipv4: '10.20.5.21',
    ipv6: '',
    environment: '生产',
    os: 'Rocky Linux 9',
    hostname: 'order-worker-02',
    networkZone: 'prod-worker',
    cpu: '8C',
    memory: '32GB',
    disk: '500GB',
    deploymentInfo: '订单异步处理',
    owner: '王敏',
    status: '维护中',
    hostMachine: 'hv-prod-06',
    __sample: true
  }
]

export const sampleMiddleware = [
  { id: 'sample-mw-1', name: 'pay-mysql-primary', kind: 'MySQL', version: '8.0', environment: '生产', networkZone: 'prod-db', endpoint: '10.20.3.16:3306', business: '支付服务', owner: '陈浩', status: '运行中', assetId: '', __sample: true },
  { id: 'sample-mw-2', name: 'pay-cache-cluster', kind: 'Redis', version: '7.2', environment: '生产', networkZone: 'prod-cache', endpoint: '10.20.4.18:6379', business: '支付服务', owner: '赵晨', status: '运行中', assetId: '', __sample: true },
  { id: 'sample-mw-3', name: 'gateway-nginx', kind: 'Nginx', version: '1.26', environment: '生产', networkZone: 'dmz', endpoint: '10.10.1.10:443', business: '网关', owner: '刘洋', status: '运行中', assetId: '', __sample: true }
]

export const sampleOncalls = [
  { id: 'sample-oncall-1', ruleType: 'daily', date: '今天', week: '', primary: '李明', backup: '王敏', swapFrom: '', swapTo: '', notes: '覆盖生产与核心数据库', __sample: true },
  { id: 'sample-oncall-2', ruleType: 'weekly', date: '', week: '本周', primary: '赵晨', backup: '陈浩', swapFrom: '刘洋', swapTo: '赵晨', notes: '换班已确认', __sample: true }
]

export const sampleTasks = [
  { id: 'sample-task-1', title: '核心数据库巡检结果确认', type: '任务', assignee: 'SRE', status: '待确认', dueAt: '今天 18:00', description: '确认慢查询和备份状态', __sample: true },
  { id: 'sample-task-2', title: '支付服务异常峰值协同处理', type: '事件任务', assignee: '运维工程师 / 研发', status: '处理中', dueAt: '剩余 42 分钟', description: '跟进 P1 事件关联任务', __sample: true },
  { id: 'sample-task-3', title: '新增云主机归属确认', type: '资产', assignee: '运维工程师', status: '待处理', dueAt: '明天 12:00', description: '补齐业务归属与负责人', __sample: true }
]

export const sampleIncidents = [
  { id: 'sample-incident-1', title: '支付服务异常峰值', level: 'P1', status: '处理中', owner: '李明', business: '支付服务', startedAt: '09:18', recoveredAt: '', summary: '接口错误率升高，已关联支付服务资产与 MySQL 主库', __sample: true },
  { id: 'sample-incident-2', title: '缓存集群连接抖动', level: 'P2', status: '已恢复', owner: '赵晨', business: '支付服务', startedAt: '08:42', recoveredAt: '09:05', summary: 'Redis 节点短时抖动，待关闭确认', __sample: true },
  { id: 'sample-incident-3', title: '网关证书到期预警', level: 'P3', status: '新建', owner: '刘洋', business: '网关', startedAt: '10:12', recoveredAt: '', summary: 'Nginx 网关证书剩余 7 天', __sample: true }
]

export const sampleUsers = [
  { id: 'sample-user-1', username: 'admin', displayName: '超级管理员', roles: ['super_admin'], mustChangePassword: false, __sample: true },
  { id: 'sample-user-2', username: 'ops-demo', displayName: '运维工程师', roles: ['ops_engineer'], mustChangePassword: true, __sample: true }
]
