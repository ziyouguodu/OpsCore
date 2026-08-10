<script setup>
const props = defineProps({
  canManage: { type: Boolean, default: false },
  config: { type: Object, required: true },
  connection: { type: Object, required: true },
  editorMode: { type: String, default: 'closed' },
  editorOpen: { type: Boolean, default: false },
  profiles: { type: Array, default: () => [] },
  providers: { type: Array, default: () => [] },
  selectedProvider: { type: Object, required: true }
})

defineEmits([
  'activate', 'close-editor', 'delete', 'edit', 'open-create', 'save',
  'select-provider', 'select-provider-id', 'test', 'test-profile'
])

function providerName(providerID) {
  return props.providers.find((provider) => provider.id === providerID)?.name || providerID
}

function profileModel(profile) {
  return profile.provider === 'local' ? profile.localModel : profile.model
}

function profileEndpoint(profile) {
  return profile.provider === 'local' ? profile.localEndpoint : profile.endpoint
}
</script>

<template>
  <section class="panel" role="region" aria-label="AI Copilot 配置工作区">
    <div class="section-head actions-only">
      <div class="page-actions">
        <span class="pill success">建议态 AI</span>
        <button class="primary" :disabled="!canManage" @click="$emit('open-create')">新增配置</button>
      </div>
    </div>

    <div class="copilot-config-layout">
      <section class="provider-grid" aria-label="模型厂商">
        <button
          v-for="provider in providers"
          :key="provider.id"
          type="button"
          :class="['provider-card', { active: editorOpen && config.provider === provider.id }]"
          :aria-pressed="editorOpen && config.provider === provider.id"
          :disabled="!canManage"
          @click="$emit('select-provider', provider)"
        >
          <span>{{ provider.badge }}</span>
          <strong>{{ provider.name }}</strong>
          <small>{{ provider.desc }}</small>
        </button>
      </section>

      <section class="config-card copilot-profile-workspace">
        <div class="config-title copilot-list-title">
          <div>
            <h3>已保存模型 <span class="muted">（{{ profiles.length }}）</span></h3>
            <p class="muted">当前启用配置用于 Copilot 问答，其余配置可测试后切换。</p>
          </div>
        </div>

        <div v-if="profiles.length" class="table-wrap">
          <table class="table copilot-profile-table">
            <thead>
              <tr>
                <th>配置名称</th>
                <th>厂商</th>
                <th>模型</th>
                <th>Endpoint</th>
                <th>密钥状态</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="profile in profiles" :key="profile.id" :class="{ 'is-active-profile': profile.isActive }">
                <td><strong>{{ profile.name }}</strong></td>
                <td>{{ providerName(profile.provider) }}</td>
                <td><code>{{ profileModel(profile) }}</code></td>
                <td class="endpoint-cell" :title="profileEndpoint(profile)">{{ profileEndpoint(profile) }}</td>
                <td>
                  <span v-if="profile.provider === 'local'" class="pill">无需密钥</span>
                  <span v-else :class="['pill', { success: profile.hasApiKey }]">{{ profile.hasApiKey ? '已托管' : '未配置' }}</span>
                </td>
                <td><span :class="['pill', { success: profile.isActive }]">{{ profile.isActive ? '当前启用' : '已保存' }}</span></td>
                <td>
                  <div class="table-actions copilot-profile-actions">
                    <button type="button" @click="$emit('edit', profile)">编辑</button>
                    <button type="button" @click="$emit('test-profile', profile)">测试</button>
                    <button v-if="!profile.isActive" type="button" @click="$emit('activate', profile)">设为当前</button>
                    <button
                      class="danger-text"
                      type="button"
                      :disabled="profile.isActive"
                      :title="profile.isActive ? '请先启用其他模型配置' : '删除模型配置'"
                      @click="$emit('delete', profile)"
                    >删除</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="empty-state copilot-empty-state">
          <strong>尚未保存模型配置</strong>
          <span>新增首个模型配置后，系统会将其设为当前启用模型。</span>
          <button class="primary" :disabled="!canManage" @click="$emit('open-create')">新增配置</button>
        </div>
      </section>

      <section v-if="editorOpen" class="config-card copilot-profile-editor">
        <div class="config-title">
          <span class="role-line-icon"><img class="copilot-icon-img" src="/copilot-root-cause-icon.png" alt="" /></span>
          <div>
            <h3>{{ editorMode === 'edit' ? '编辑模型配置' : '新增模型配置' }}</h3>
            <p class="muted">{{ selectedProvider.name }} · {{ selectedProvider.desc }}</p>
          </div>
          <div class="page-actions editor-actions">
            <button :disabled="!canManage || connection.testing" @click="$emit('test')">
              {{ connection.testing ? '测试中...' : '测试连接' }}
            </button>
            <button class="primary" :disabled="!canManage" @click="$emit('save')">保存配置</button>
            <button @click="$emit('close-editor')">取消</button>
          </div>
        </div>

        <div class="form-grid copilot-form">
          <label>配置名称
            <input v-model="config.name" :disabled="!canManage" maxlength="80" placeholder="例如：生产分析模型" />
          </label>
          <label>模型厂商
            <select v-model="config.provider" :disabled="!canManage" @change="$emit('select-provider-id')">
              <option value="local">本地模型</option>
              <option value="openai">OpenAI GPT</option>
              <option value="anthropic">Anthropic Claude</option>
              <option value="google">Google Gemini</option>
              <option value="compatible">OpenAI 兼容接口</option>
            </select>
          </label>

          <template v-if="config.provider === 'local'">
            <label>本地模型地址
              <input v-model="config.localEndpoint" :disabled="!canManage" placeholder="http://host.docker.internal:11434" />
            </label>
            <label>本地模型
              <input v-model="config.localModel" :disabled="!canManage" placeholder="例如 qwen2.5:7b" />
            </label>
          </template>
          <template v-else>
            <label>API Endpoint
              <input v-model="config.endpoint" :disabled="!canManage" placeholder="模型服务地址" />
            </label>
            <label>模型名称
              <input v-model="config.model" :disabled="!canManage" placeholder="例如 gpt-4.1 / claude-sonnet" />
            </label>
            <label>API Key
              <input v-model="config.apiKey" type="password" :disabled="!canManage" :placeholder="config.hasApiKey ? '已托管密钥；留空则继续使用' : '输入后由后端加密托管'" />
            </label>
          </template>

          <label>回答随机性（Temperature）
            <input v-model="config.temperature" type="number" min="0" max="2" step="0.1" :disabled="!canManage" />
          </label>
          <label>最大输出长度（Max Tokens）
            <input v-model="config.maxTokens" type="number" min="1" max="4096" step="1" :disabled="!canManage" />
          </label>
        </div>

        <div v-if="config.provider !== 'local'" class="config-secret-state">
          <span :class="['pill', { success: config.hasApiKey }]">{{ config.hasApiKey ? 'API Key 已托管' : '未配置托管密钥' }}</span>
          <small>{{ config.hasApiKey ? '留空会继续使用当前托管密钥；切换厂商或 Endpoint 时需重新输入 Key。' : '保存后 API Key 由后端加密托管。' }}</small>
        </div>

        <div v-if="connection.message" :class="['connection-result', connection.ok ? 'success' : 'danger']">
          <div>
            <strong>{{ connection.ok ? '连接可用' : '连接未通过' }}</strong>
            <span>{{ connection.message }}</span>
          </div>
          <small v-if="connection.latencyMs !== null">
            响应 {{ connection.latencyMs }} ms
            <template v-if="connection.statusCode"> · HTTP {{ connection.statusCode }}</template>
          </small>
        </div>

        <div class="copilot-context-editor">
          <h3>上下文授权</h3>
          <div class="context-grid">
            <label class="inline-check"><input v-model="config.enableAssetContext" type="checkbox" :disabled="!canManage" />资产与实例上下文</label>
            <label class="inline-check"><input v-model="config.enableIncidentContext" type="checkbox" :disabled="!canManage" />事件影响上下文</label>
            <label class="inline-check"><input v-model="config.enableTaskContext" type="checkbox" :disabled="!canManage" />任务闭环上下文</label>
            <label class="inline-check"><input v-model="config.enableOncallContext" type="checkbox" :disabled="!canManage" />值班与交接上下文</label>
            <label class="inline-check"><input type="checkbox" checked disabled />问答审计（强制）</label>
          </div>
        </div>
      </section>

      <section class="config-card">
        <div class="ai-guardrails">
          <article><strong>权限感知</strong><span>Copilot 只读取当前账号有权访问的数据。</span></article>
          <article><strong>建议态输出</strong><span>自动化执行前必须有人审、权限、审计和回滚策略。</span></article>
          <article><strong>密钥托管</strong><span>生产 API Key 不落前端，由后端加密保存并代理调用。</span></article>
        </div>
      </section>
    </div>
  </section>
</template>
