<script setup>
defineProps({
  auth: { type: Object, required: true },
  error: { type: String, default: '' },
  loading: { type: Boolean, default: false },
  needsInitialPassword: { type: Boolean, default: false },
  passwordInit: { type: Object, required: true },
  pending: { type: Boolean, default: false }
})

defineEmits(['change-password', 'logout', 'submit-login'])
</script>

<template>
  <section v-if="!auth.token" class="auth-screen">
    <div class="auth-hero">
      <div class="auth-brand">
        <img src="/opscore-ai-ops-logo.png" alt="OpsCore logo" />
        <span>OpsCore</span>
      </div>
      <h1>智能运维中枢指挥平台</h1>
      <p>连接资产、可观测、事件、变更、知识与自动化能力，构建以业务连续性为核心、以 AI 辅助决策和受控自动化为增强的统一运维控制平面。</p>
      <div class="auth-command-map" aria-label="OpsCore 智能运维中枢示意">
        <div class="command-orbit"></div>
        <div class="command-axis"></div>
        <div class="command-core">
          <span>OpsCore</span>
          <strong>智能运维中枢</strong>
        </div>
        <span class="command-node data">统一运维数据底座</span>
        <span class="command-node continuity">业务连续性保障</span>
        <span class="command-node automation">自动化处置闭环</span>
        <span class="command-node governance">治理与审计</span>
        <span class="command-node ai">AI 决策中枢</span>
        <div class="command-flow">
          <span>健康</span>
          <i></i>
          <span>影响</span>
          <i></i>
          <span>根因</span>
          <i></i>
          <span>处置</span>
          <i></i>
          <span>复盘</span>
        </div>
      </div>
    </div>
    <form class="login-card auth-card" @submit.prevent="$emit('submit-login')">
      <small>账号密码登录</small>
      <h2>登录 OpsCore</h2>
      <p>使用受控账号密码体系进入运维控制台。</p>
      <span v-if="error" class="error">{{ error }}</span>
      <label>账号<input v-model="auth.username" autocomplete="username" /></label>
      <label>密码<input v-model="auth.password" type="password" autocomplete="current-password" /></label>
      <button class="primary" type="submit">进入控制台</button>
      <div class="login-meta" aria-label="登录安全能力">
        <span>凭据加密</span>
        <span>RBAC</span>
        <span>审计预留</span>
      </div>
    </form>
  </section>

  <section v-else-if="pending" class="auth-screen auth-loading">
    <div class="auth-hero">
      <div class="auth-brand">
        <img src="/opscore-ai-ops-logo.png" alt="OpsCore logo" />
        <span>OpsCore</span>
      </div>
      <h1>正在恢复控制台会话</h1>
      <p>正在校验登录状态并加载资产、值班、任务与事件数据。</p>
    </div>
    <div class="login-card auth-card">
      <small>Session Check</small>
      <h2>连接 OpsCore</h2>
      <p>请稍候，登录态校验完成后会自动进入控制台。</p>
      <span v-if="error" class="error">{{ error }}</span>
      <button type="button" @click="$emit('logout')">重新登录</button>
    </div>
  </section>

  <section v-else-if="needsInitialPassword" class="auth-screen">
    <div class="auth-hero">
      <div class="auth-brand">
        <img src="/opscore-ai-ops-logo.png" alt="OpsCore logo" />
        <span>OpsCore</span>
      </div>
      <h1>初始化安全访问</h1>
      <p>首次进入控制台前，需要完成超级管理员初始化密码，确保敏感凭据和运维操作受控。</p>
      <div class="auth-command-map compact" aria-label="OpsCore 安全初始化示意">
        <div class="command-orbit"></div>
        <div class="command-axis"></div>
        <div class="command-core">
          <span>OpsCore</span>
          <strong>安全访问</strong>
        </div>
        <span class="command-node data">凭据加密</span>
        <span class="command-node continuity">权限隔离</span>
        <span class="command-node automation">操作闭环</span>
        <span class="command-node governance">治理审计</span>
        <span class="command-node ai">受控接入</span>
      </div>
    </div>
    <form class="login-card auth-card password-card" @submit.prevent="$emit('change-password')">
      <small>首次登录</small>
      <h2>初始化管理员密码</h2>
      <p>检测到当前账号仍在使用初始化密码。请先修改密码，再进入 OpsCore。</p>
      <span v-if="error" class="error">{{ error }}</span>
      <label>当前密码<input v-model="passwordInit.currentPassword" type="password" autocomplete="current-password" /></label>
      <label>新密码<input v-model="passwordInit.newPassword" type="password" placeholder="至少 8 位" autocomplete="new-password" /></label>
      <label>确认新密码<input v-model="passwordInit.confirmPassword" type="password" autocomplete="new-password" /></label>
      <button class="primary" :disabled="loading" type="submit">完成初始化</button>
      <button type="button" @click="$emit('logout')">返回登录</button>
    </form>
  </section>
</template>
