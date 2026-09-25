<script setup>
import { computed } from 'vue'
import PaginationBar from './PaginationBar.vue'
import BulkDataTools from './BulkDataTools.vue'

const props = defineProps({
  associatedAssetName: { type: Function, required: true },
  businesses: { type: Array, default: () => [] },
  canManageCredentials: { type: Boolean, default: false },
  canWrite: { type: Boolean, default: false },
  bulkBusy: { type: Boolean, default: false },
  canBulkDelete: { type: Boolean, default: false },
  credential: { type: Object, required: true },
  credentialMessage: { type: String, default: '' },
  credentialReveal: { type: Object, required: true },
  filters: { type: Object, required: true },
  form: { type: Object, required: true },
  formCredential: { type: Object, required: true },
  formOpen: { type: Boolean, default: false },
  isSample: { type: Function, required: true },
  networkZones: { type: Array, default: () => [] },
  page: { type: Number, required: true },
  pageCount: { type: Number, required: true },
  pageSize: { type: Number, required: true },
  pagedItems: { type: Array, default: () => [] },
  selectedItem: { type: Object, default: null },
  selectedItems: { type: Array, default: () => [] },
  total: { type: Number, default: 0 }
})

const emit = defineEmits([
  'choose',
  'close-form',
  'delete',
  'edit',
  'export-item',
  'import-file',
  'export-selected',
  'delete-selected',
  'clear-selection',
  'toggle-select',
  'select-page',
  'download-template',
  'hide-detail',
  'load-credential',
  'open-form',
  'reset-filters',
  'reveal-credential',
  'save',
  'save-credential',
  'update:page',
  'update:pageSize'
])

const selectedIds = computed(() => new Set(props.selectedItems.map((item) => item.id)))
const pageSelectionCount = computed(() => props.pagedItems.filter((item) => selectedIds.value.has(item.id)).length)
const pageFullySelected = computed(() => props.pagedItems.length > 0 && pageSelectionCount.value === props.pagedItems.length)
const pagePartiallySelected = computed(() => pageSelectionCount.value > 0 && !pageFullySelected.value)

function isSelected(id) {
  return selectedIds.value.has(id)
}

const middlewareKinds = ['MySQL', 'Redis', 'Kafka', 'PostgreSQL', '达梦', 'Nginx', 'ElasticSearch', 'Nacos', 'RocketMQ', 'MinIO']

function resetPage() {
  emit('update:page', 1)
}

function closeEditorOnFocusOut(event) {
  const nextTarget = event.relatedTarget
  if (!nextTarget || !event.currentTarget.contains(nextTarget)) {
    emit('close-form')
  }
}
</script>

<template>
  <section class="panel" role="region" aria-label="实例管理工作区">
    <div class="section-head actions-only">
      <div class="page-actions">
        <button v-if="selectedItem" @click="$emit('export-item', selectedItem)"><span class="btn-icon">↧</span>导出</button>
        <button class="primary" :disabled="!canWrite" @click="$emit('open-form')"><span class="btn-icon">＋</span>新增实例</button>
        <BulkDataTools :can-write="canWrite" :busy="bulkBusy" :can-bulk-delete="canBulkDelete" :selected-count="selectedItems.length" label="实例" @import-file="$emit('import-file', $event)" @export-selected="$emit('export-selected')" @delete-selected="$emit('delete-selected')" @clear-selection="$emit('clear-selection')" @download-template="$emit('download-template')" />
      </div>
    </div>

    <div class="work-layout" :class="{ 'single-column': !selectedItem }">
      <section>
        <div class="query-card">
          <div class="query-main">
            <input v-model="filters.keyword" aria-label="实例关键词" placeholder="搜索实例名称、类型、访问地址、业务、负责人" @input="resetPage" />
            <select v-model="filters.kind" aria-label="实例类型" @change="resetPage">
              <option value="">全部类型</option>
              <option v-for="kind in middlewareKinds" :key="kind">{{ kind }}</option>
            </select>
            <select v-model="filters.environment" aria-label="实例环境" @change="resetPage"><option value="">全部环境</option><option>生产</option><option>仿真</option><option>研发</option></select>
            <button class="primary" @click="resetPage">查询</button>
            <button @click="$emit('reset-filters')">重置</button>
            <button class="link" @click="filters.advanced = !filters.advanced">{{ filters.advanced ? '收起高级搜索' : '高级搜索' }}</button>
          </div>
          <div v-if="filters.advanced" class="query-extra">
            <textarea v-model="filters.ips" aria-label="多个 IP 地址" rows="2" placeholder="多个 IP 地址，支持空格、逗号或换行分隔" @input="resetPage"></textarea>
            <select v-model="filters.business" aria-label="所属业务" @change="resetPage">
              <option value="">全部所属业务</option>
              <option v-for="item in businesses" :key="item">{{ item }}</option>
            </select>
            <select v-model="filters.networkZone" aria-label="网络区域" @change="resetPage">
              <option value="">全部网络区域</option>
              <option v-for="item in networkZones" :key="item">{{ item }}</option>
            </select>
            <select v-model="filters.status" aria-label="实例状态" @change="resetPage"><option value="">全部状态</option><option>运行中</option><option>维护中</option><option>停用</option><option>故障</option></select>
          </div>
        </div>

        <section v-if="formOpen" class="editor-panel" tabindex="-1" @focusout="closeEditorOnFocusOut">
          <h3>{{ form.id ? '编辑实例' : '新增实例' }}</h3>
          <div class="form-grid">
            <input v-model="form.name" aria-label="实例名称" placeholder="实例名称" />
            <select v-model="form.kind" aria-label="实例类型"><option v-for="kind in middlewareKinds" :key="kind">{{ kind }}</option></select>
            <input v-model="form.version" aria-label="版本" placeholder="版本" />
            <select v-model="form.environment" aria-label="环境"><option>生产</option><option>仿真</option><option>研发</option></select>
            <input v-model="form.networkZone" aria-label="网络区域" placeholder="网络区域" />
            <input v-model="form.endpoint" aria-label="访问地址或端口" placeholder="访问地址 / 端口" />
            <input v-model="form.business" aria-label="所属业务" placeholder="所属业务" />
            <input v-model="form.owner" aria-label="负责人" placeholder="负责人" />
            <input v-model="form.assetId" aria-label="关联资产 ID" placeholder="关联资产 ID（非必填）" />
            <select v-model="form.status" aria-label="实例状态"><option>运行中</option><option>维护中</option><option>停用</option><option>故障</option></select>
            <section v-if="canManageCredentials" class="credential-inline form-wide">
              <div>
                <strong>实例登录信息</strong>
                <span>保存后加密存储，列表不展示；查看密码需统一二次校验。</span>
              </div>
              <div class="form-grid credential-form-inline">
                <input v-model="formCredential.loginUrl" aria-label="管理地址或连接入口" placeholder="管理地址 / 连接入口（可选）" />
                <input v-model="formCredential.username" aria-label="登录用户名" placeholder="登录用户名" />
                <input v-model="formCredential.secret" aria-label="登录密码或密钥" placeholder="登录密码 / 密钥" type="password" />
                <input v-model="formCredential.notes" aria-label="登录信息备注" placeholder="备注" />
              </div>
            </section>
            <button class="primary" :disabled="!canWrite" @click="$emit('save')">{{ form.id ? '保存修改' : '保存实例' }}</button>
            <button @click="$emit('close-form')">取消</button>
          </div>
        </section>

        <div class="table-wrap">
          <table>
            <thead><tr><th class="selection-cell"><input type="checkbox" aria-label="选择本页实例" :checked="pageFullySelected" :indeterminate="pagePartiallySelected" @click.stop @change="$emit('select-page', $event.target.checked, pagedItems)" /></th><th>实例名称</th><th>类型</th><th>环境</th><th>网络区域</th><th>访问地址</th><th>所属业务</th><th>关联资产</th><th>状态</th><th>操作</th></tr></thead>
            <tbody>
              <tr v-for="item in pagedItems" :key="item.id" :class="{ selected: selectedItem?.id === item.id }" class="clickable-row" tabindex="0" @click="$emit('choose', item)" @keyup.enter="$emit('choose', item)" @keyup.space.prevent="$emit('choose', item)">
                <td class="selection-cell"><input type="checkbox" :aria-label="`选择实例 ${item.name || item.id}`" :checked="isSelected(item.id)" @click.stop @change="$emit('toggle-select', item)" /></td><td>{{ item.name }}</td><td>{{ item.kind }}</td><td>{{ item.environment }}</td><td>{{ item.networkZone || '-' }}</td><td>{{ item.endpoint }}</td><td>{{ item.business }}</td><td>{{ associatedAssetName(item) }}</td><td>{{ item.status }}</td>
                <td class="row-actions"><button class="link" @click.stop="$emit('choose', item)">详情</button><button class="link" :disabled="!canWrite || isSample(item)" @click.stop="$emit('edit', item)">编辑</button><button class="link danger-text" :disabled="!canWrite || isSample(item)" @click.stop="$emit('delete', item)">删除</button></td>
              </tr>
              <tr v-if="!pagedItems.length"><td colspan="10" class="empty">未找到符合条件的实例</td></tr>
            </tbody>
          </table>
        </div>
        <PaginationBar
          :page="page"
          :page-size="pageSize"
          :total="total"
          :page-count="pageCount"
          @update:page="$emit('update:page', $event)"
          @update:page-size="$emit('update:pageSize', $event)"
        />
      </section>

      <aside v-if="selectedItem" class="detail-card">
        <div class="detail-title">
          <h3>{{ selectedItem.name }}</h3>
          <button class="detail-close" aria-label="隐藏实例详情" @click="$emit('hide-detail')">隐藏</button>
        </div>
        <dl>
          <dt>类型 / 版本</dt><dd>{{ selectedItem.kind }} / {{ selectedItem.version || '-' }}</dd>
          <dt>环境</dt><dd>{{ selectedItem.environment }}</dd>
          <dt>网络区域</dt><dd>{{ selectedItem.networkZone || '-' }}</dd>
          <dt>访问地址</dt><dd>{{ selectedItem.endpoint }}</dd>
          <dt>所属业务</dt><dd>{{ selectedItem.business }}</dd>
          <dt>负责人</dt><dd>{{ selectedItem.owner || '-' }}</dd>
          <dt>关联资产</dt><dd>{{ associatedAssetName(selectedItem) }}</dd>
          <dt>状态</dt><dd>{{ selectedItem.status }}</dd>
        </dl>
        <div class="detail-actions">
          <button class="primary" :disabled="!canWrite || isSample(selectedItem)" @click="$emit('edit', selectedItem)">编辑实例</button>
          <button :disabled="!canWrite || isSample(selectedItem)" @click="$emit('edit', selectedItem)">编辑关联资产</button>
          <button class="danger-action" :disabled="!canWrite || isSample(selectedItem)" @click="$emit('delete', selectedItem)">删除实例</button>
        </div>
        <div class="credential-box">
          <h4>实例账号密码</h4>
          <template v-if="canManageCredentials">
            <p class="muted">用于保存数据库、中间件或组件实例的访问账号；密码/密钥默认隐藏，需输入权限管理中配置的统一校验密码后查看。</p>
            <div class="credential-actions">
              <button @click="$emit('load-credential')">加载账号密码</button>
              <span v-if="credential.hasSecret" class="pill danger">已保存密钥</span>
              <span v-else class="pill">未保存密钥</span>
            </div>
            <div class="form-grid credential-form">
              <input v-model="credential.loginUrl" aria-label="管理地址或连接入口" placeholder="管理地址 / 连接入口" />
              <input v-model="credential.username" aria-label="登录账号" placeholder="账号" />
              <input v-model="credential.secret" aria-label="密码或密钥" placeholder="密码 / 密钥（留空则保留原值）" type="password" />
              <input v-model="credential.notes" aria-label="登录信息备注" placeholder="备注" />
              <button class="primary" @click="$emit('save-credential')">保存账号密码</button>
            </div>
            <div class="credential-reveal">
              <input v-model="credentialReveal.password" aria-label="统一二次校验密码" placeholder="输入统一二次校验密码查看密码/密钥" type="password" @keyup.enter="$emit('reveal-credential')" />
              <button @click="$emit('reveal-credential')">二次校验查看</button>
              <span v-if="credentialReveal.revealed" class="pill success">已校验</span>
            </div>
          </template>
          <p v-else class="muted">当前角色无权查看实例账号密码。运维工程师默认不可见。</p>
          <p v-if="credentialMessage" class="error inline">{{ credentialMessage }}</p>
        </div>
      </aside>
    </div>
  </section>
</template>
