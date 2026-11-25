<script setup>
import { computed } from 'vue'
import PaginationBar from './PaginationBar.vue'
import WorkspacePageHeader from './WorkspacePageHeader.vue'
import BulkDataTools from './BulkDataTools.vue'
import { bulkSchemas } from '../bulk-data'

const props = defineProps({
  businesses: { type: Array, default: () => [] },
  canDelete: { type: Function, required: true },
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
  formCredentialReady: { type: Boolean, default: true },
  formOpen: { type: Boolean, default: false },
  isSample: { type: Function, required: true },
  networkZones: { type: Array, default: () => [] },
  page: { type: Number, required: true },
  pageCount: { type: Number, required: true },
  pageSize: { type: Number, required: true },
  pagedAssets: { type: Array, default: () => [] },
  selectedAsset: { type: Object, default: null },
  selectedItems: { type: Array, default: () => [] },
  spec: { type: Function, required: true },
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
const pageSelectionCount = computed(() => props.pagedAssets.filter((item) => selectedIds.value.has(item.id)).length)
const pageFullySelected = computed(() => props.pagedAssets.length > 0 && pageSelectionCount.value === props.pagedAssets.length)
const pagePartiallySelected = computed(() => pageSelectionCount.value > 0 && !pageFullySelected.value)

const assetFields = bulkSchemas.assets.map(([key, label, required]) => ({
  key,
  label: ({ type: '资产类型', cpuArch: 'CPU 架构', cpu: 'CPU 规格', memory: '内存规格', disk: '磁盘规格', status: '资产状态' })[key] || label,
  required: Boolean(required),
  options: ({ type: ['物理机', '虚拟机'], environment: ['生产', '仿真', '研发'], status: ['运行中', '维护中', '停用', '故障'] })[key]
}))

function fieldRequired(field) {
  return field.required || (field.key === 'assetNo' && Boolean(props.form.id))
}

function isSelected(id) {
  return selectedIds.value.has(id)
}

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
  <section class="panel" role="region" aria-label="资产台账工作区">
    <WorkspacePageHeader view="cmdb" />
    <div class="section-head actions-only">
      <div class="page-actions">
        <button v-if="selectedAsset" @click="$emit('export-item', selectedAsset)"><span class="btn-icon">↧</span>导出</button>
        <button class="primary" :disabled="!canWrite" @click="$emit('open-form')"><span class="btn-icon">＋</span>新增资产</button>
        <BulkDataTools :can-write="canWrite" :busy="bulkBusy" :can-bulk-delete="canBulkDelete" :selected-count="selectedItems.length" label="资产" @import-file="$emit('import-file', $event)" @export-selected="$emit('export-selected')" @delete-selected="$emit('delete-selected')" @clear-selection="$emit('clear-selection')" @download-template="$emit('download-template')" />
      </div>
    </div>

    <div class="cmdb-layout" :class="{ 'single-column': !selectedAsset }">
      <section>
        <div class="query-card">
          <div class="query-main">
            <input v-model="filters.keyword" aria-label="资产关键词" placeholder="搜索资产编号、业务、IP、负责人、部署信息" @input="resetPage" />
            <select v-model="filters.type" aria-label="资产类型" @change="resetPage"><option value="">全部类型</option><option>物理机</option><option>虚拟机</option></select>
            <select v-model="filters.environment" aria-label="资产环境" @change="resetPage"><option value="">全部环境</option><option>生产</option><option>仿真</option><option>研发</option></select>
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
          </div>
        </div>

        <section v-if="formOpen" class="editor-panel" tabindex="-1" @focusout="closeEditorOnFocusOut">
          <h3>{{ form.id ? '编辑资产' : '新增资产' }}</h3>
          <div class="form-grid cmdb-form">
            <label v-for="field in assetFields" :key="field.key">
              <span>{{ field.label }}{{ fieldRequired(field) ? '（必填）' : '（可选）' }}</span>
              <select v-if="field.options" v-model="form[field.key]" :aria-label="field.label" :required="fieldRequired(field)" :aria-required="fieldRequired(field)"><option v-for="option in field.options" :key="option">{{ option }}</option></select>
              <input v-else v-model="form[field.key]" :aria-label="field.label" :placeholder="field.key === 'assetNo' && !form.id ? '留空自动生成' : field.label" :required="fieldRequired(field)" :aria-required="fieldRequired(field)" />
            </label>
            <section v-if="canManageCredentials" class="credential-inline form-wide">
              <div>
                <strong>登录信息</strong>
                <span>保存后加密存储，列表不展示；查看密码需统一二次校验。</span>
              </div>
              <div class="form-grid credential-form-inline">
                <label>登录地址（可选）<input v-model="formCredential.loginUrl" :disabled="!formCredentialReady" aria-label="登录地址" placeholder="登录地址" /></label>
                <label>登录用户名（可选）<input v-model="formCredential.username" :disabled="!formCredentialReady" aria-label="登录用户名" placeholder="登录用户名" /></label>
                <label>登录密码或密钥（可选）<input v-model="formCredential.secret" :disabled="!formCredentialReady" aria-label="登录密码或密钥" placeholder="留空则保留原值" type="password" /></label>
                <label>登录信息备注（可选）<input v-model="formCredential.notes" :disabled="!formCredentialReady" aria-label="登录信息备注" placeholder="备注" /></label>
              </div>
            </section>
            <button class="primary" :disabled="!canWrite || !formCredentialReady" @click="$emit('save')">{{ form.id ? '保存修改' : '保存资产' }}</button>
            <button @click="$emit('close-form')">取消</button>
          </div>
        </section>

        <div class="table-wrap">
          <table>
            <thead><tr><th class="selection-cell"><input type="checkbox" aria-label="选择本页资产" :checked="pageFullySelected" :indeterminate="pagePartiallySelected" @click.stop @change="$emit('select-page', $event.target.checked, pagedAssets)" /></th><th>资产编号</th><th>类型</th><th>环境</th><th>网络区域</th><th>IP</th><th>配置规格</th><th>所属业务</th><th>部署信息</th><th>状态</th><th>负责人</th><th>操作</th></tr></thead>
            <tbody>
              <tr v-for="asset in pagedAssets" :key="asset.id" :class="{ selected: selectedAsset?.id === asset.id }" class="clickable-row" tabindex="0" @click="$emit('choose', asset)" @keyup.enter="$emit('choose', asset)" @keyup.space.prevent="$emit('choose', asset)">
                <td class="selection-cell"><input type="checkbox" :aria-label="`选择资产 ${asset.assetNo || asset.id}`" :checked="isSelected(asset.id)" @click.stop @change="$emit('toggle-select', asset)" /></td><td>{{ asset.assetNo }}</td><td>{{ asset.type }}</td><td>{{ asset.environment }}</td><td>{{ asset.networkZone }}</td><td>{{ asset.ipv4 || asset.ipv6 }}</td><td>{{ spec(asset) }}</td><td>{{ asset.business }}</td><td>{{ asset.deploymentInfo || '-' }}</td><td>{{ asset.status }}</td><td>{{ asset.owner || '-' }}</td>
                <td class="row-actions"><button class="link" @click.stop="$emit('choose', asset)">详情</button><button class="link" :disabled="!canWrite || isSample(asset)" @click.stop="$emit('edit', asset)">编辑</button><button class="link danger-text" :disabled="!canDelete(asset)" @click.stop="$emit('delete', asset)">删除</button></td>
              </tr>
              <tr v-if="!pagedAssets.length"><td colspan="12" class="empty">未找到符合条件的资产</td></tr>
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

      <aside v-if="selectedAsset" class="asset-detail">
        <div class="detail-title">
          <h3>资产详情</h3>
          <button class="detail-close" aria-label="隐藏资产详情" @click="$emit('hide-detail')">隐藏</button>
        </div>
        <dl>
          <dt>资产编号</dt><dd>{{ selectedAsset.assetNo }}</dd>
          <dt>厂商 / SN</dt><dd>{{ selectedAsset.vendor || '-' }} / {{ selectedAsset.sn || '-' }}</dd>
          <dt>配置规格</dt><dd>{{ spec(selectedAsset) }}</dd>
          <dt>部署信息</dt><dd>{{ selectedAsset.deploymentInfo || '-' }}</dd>
          <dt>所在宿主机</dt><dd>{{ selectedAsset.hostMachine || '-' }}</dd>
          <dt>负责人</dt><dd>{{ selectedAsset.owner || '-' }}</dd>
        </dl>
        <div class="detail-actions">
          <button class="primary" :disabled="!canWrite || isSample(selectedAsset)" @click="$emit('edit', selectedAsset)">编辑资产</button>
          <button class="danger-action" :disabled="!canDelete(selectedAsset)" @click="$emit('delete', selectedAsset)">删除资产</button>
        </div>
        <div class="credential-box">
          <h4>登录信息</h4>
          <template v-if="canManageCredentials">
            <p class="muted">登录地址、账号与备注可维护；密码/密钥默认隐藏，需输入权限管理中配置的统一二次校验密码（不是登录密码）后查看。</p>
            <div class="credential-actions">
              <button @click="$emit('load-credential')">加载登录信息</button>
              <span v-if="credential.hasSecret" class="pill danger">已保存密钥</span>
              <span v-else class="pill">未保存密钥</span>
            </div>
            <div class="form-grid credential-form">
              <input v-model="credential.loginUrl" aria-label="登录地址" placeholder="登录地址" />
              <input v-model="credential.username" aria-label="登录账号" placeholder="账号" />
              <input v-model="credential.secret" aria-label="密码或密钥" placeholder="密码 / 密钥（留空则保留原值）" :type="credentialReveal.revealed && credentialReveal.visible ? 'text' : 'password'" />
              <button v-if="credentialReveal.revealed" :aria-pressed="credentialReveal.visible" @click="credentialReveal.visible = !credentialReveal.visible">{{ credentialReveal.visible ? '隐藏密码/密钥' : '显示密码/密钥' }}</button>
              <input v-model="credential.notes" aria-label="登录信息备注" placeholder="备注" />
              <button class="primary" @click="$emit('save-credential')">保存登录信息</button>
            </div>
            <div class="credential-reveal">
              <input v-model="credentialReveal.password" aria-label="统一二次校验密码" placeholder="输入统一二次校验密码查看密码/密钥" type="password" @keyup.enter="$emit('reveal-credential')" />
              <button @click="$emit('reveal-credential')">二次校验查看</button>
              <span v-if="credentialReveal.revealed" class="pill success">已校验</span>
            </div>
          </template>
          <p v-else class="muted">当前角色无权查看登录信息。运维工程师默认不可见。</p>
          <p v-if="credentialMessage" class="error inline">{{ credentialMessage }}</p>
        </div>
      </aside>
    </div>
  </section>
</template>
