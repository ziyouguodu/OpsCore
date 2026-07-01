<script setup>
defineProps({
  canManage: { type: Boolean, default: false },
  config: { type: Object, required: true },
  connection: { type: Object, required: true },
  providers: { type: Array, default: () => [] },
  selectedProvider: { type: Object, required: true }
})

defineEmits(['save', 'select-provider', 'select-provider-id', 'test'])
</script>

<template>
  <section class="panel" role="region" aria-label="AI Copilot 配置工作区">
    <div class="section-head actions-only">
      <div class="page-actions">
        <span class="pill success">建议态 AI</span>
        <button :disabled="!canManage || connection.testing" @click="$emit('test')">
          {{ connection.testing ? '测试中...' : '测试连接' }}
        </button>
        <button class="primary" :disabled="!canManage" @click="$emit('save')">保存配置</button>
      </div>
    </div>

    <div class="copilot-config-layout">
      <section class="provider-grid" aria-label="模型厂商">
        <button
          v-for="provider in providers"
          :key="provider.id"
          type="button"
          :class="['provider-card', { active: config.provider === provider.id }]"
          :aria-pressed="config.provider === provider.id"
          :disabled="!canManage"
          @click="$emit('select-provider', provider)"
        >
          <span>{{ provider.badge }}</span>
          <strong>{{ provider.name }}</strong>
          <small>{{ provider.desc }}</small>
        </button>
      </section>

      <section class="config-card">
        <div class="config-title">
          <span class="role-line-icon"><img class="copilot-icon-img" src="/copilot-root-cause-icon.png" alt="" /></span>
          <div>
            <h3>{{ selectedProvider.name }}</h3>
            <p class="muted">{{ selectedProvider.desc }}</p>
          </div>
        </div>
        <div class="form-grid copilot-form">
          <label>模型厂商
            <select v-model="config.provider" :disabled="!canManage" @change="$emit('select-provider-id')">
              <option value="local">本地模型</option>
              <option value="openai">OpenAI GPT</option>
              <option value="anthropic">Anthropic Claude</option>
              <option value="google">Google Gemini</option>
              <option value="compatible">OpenAI 兼容接口</option>
            </select>
          </label>
          <label>API Endpoint
            <input v-model="config.endpoint" :disabled="!canManage" placeholder="模型服务地址" />
          </label>
          <label>模型名称
            <input v-model="config.model" :disabled="!canManage" placeholder="例如 gpt-4.1 / claude-3-7-sonnet" />
          </label>
          <label>API Key
            <input v-model="config.apiKey" type="password" :disabled="!canManage" :placeholder="config.hasApiKey ? '已托管密钥；留空则继续使用' : '输入后由后端加密托管'" />
          </label>
          <label>本地模型地址
            <input v-model="config.localEndpoint" :disabled="!canManage" placeholder="Docker 下例如 http://host.docker.internal:11434" />
          </label>
          <label>本地模型
            <input v-model="config.localModel" :disabled="!canManage" placeholder="例如 qwen2.5:7b" />
          </label>
          <label>Temperature
            <input v-model="config.temperature" :disabled="!canManage" />
          </label>
          <label>Max Tokens
            <input v-model="config.maxTokens" :disabled="!canManage" />
          </label>
        </div>
        <div class="config-secret-state">
          <span :class="['pill', config.hasApiKey ? 'success' : '']">{{ config.hasApiKey ? 'API Key 已托管' : '未配置托管密钥' }}</span>
          <small>{{ config.hasApiKey ? '保存时留空会继续使用当前托管密钥；切换厂商或 Endpoint 时请重新输入匹配的 Key。' : 'Hosted 模型需要填写 API Key 后保存或直接测试。' }}</small>
        </div>
        <div
          v-if="connection.message"
          :class="['connection-result', connection.ok ? 'success' : 'danger']"
        >
          <div>
            <strong>{{ connection.ok ? '连接可用' : '连接未通过' }}</strong>
            <span>{{ connection.message }}</span>
          </div>
          <small v-if="connection.latencyMs !== null">
            响应 {{ connection.latencyMs }} ms
            <template v-if="connection.statusCode"> · HTTP {{ connection.statusCode }}</template>
          </small>
        </div>
      </section>

      <section class="config-card">
        <h3>上下文授权</h3>
        <div class="context-grid">
          <label class="inline-check"><input v-model="config.enableAssetContext" type="checkbox" :disabled="!canManage" />资产与实例上下文</label>
          <label class="inline-check"><input v-model="config.enableIncidentContext" type="checkbox" :disabled="!canManage" />事件影响上下文</label>
          <label class="inline-check"><input v-model="config.enableTaskContext" type="checkbox" :disabled="!canManage" />任务闭环上下文</label>
          <label class="inline-check"><input v-model="config.enableOncallContext" type="checkbox" :disabled="!canManage" />值班与交接上下文</label>
          <label class="inline-check"><input v-model="config.auditEnabled" type="checkbox" :disabled="!canManage" />启用问答审计</label>
        </div>
        <div class="ai-guardrails">
          <article><strong>权限感知</strong><span>Copilot 只应读取当前账号有权访问的数据。</span></article>
          <article><strong>建议态输出</strong><span>自动化执行前必须有人审、权限、审计和回滚策略。</span></article>
          <article><strong>密钥托管</strong><span>生产 API Key 不落前端，后续由后端加密保存并代理调用。</span></article>
        </div>
      </section>
    </div>
  </section>
</template>
