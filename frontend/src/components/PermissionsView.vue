<script setup>
import SvgIcon from './SvgIcon.vue'

defineProps({
  canManage: { type: Boolean, default: false },
  credentialVerification: { type: Object, required: true },
  currentUserId: { type: [Number, String], default: 0 },
  displayUsers: { type: Array, default: () => [] },
  isSample: { type: Function, required: true },
  menuPermissionRows: { type: Array, default: () => [] },
  newUser: { type: Object, required: true },
  permissionRows: { type: Array, default: () => [] },
  permissionTab: { type: String, default: 'users' },
  roleCards: { type: Array, default: () => [] },
  userFormOpen: { type: Boolean, default: false }
})

const emit = defineEmits([
  'close-user-form',
  'delete-user',
  'edit-user',
  'navigate-tab',
  'open-user-form',
  'save-credential-password',
  'save-user'
])

function closeEditorOnFocusOut(event) {
  const nextTarget = event.relatedTarget
  if (!nextTarget || !event.currentTarget.contains(nextTarget)) {
    emit('close-user-form')
  }
}
</script>

<template>
  <section class="panel" role="region" aria-label="权限管理工作区">
    <div class="section-head actions-only">
      <div class="page-actions">
        <span class="pill">账号密码登录</span>
        <button v-if="canManage && permissionTab === 'users'" class="primary" @click="$emit('open-user-form')"><span class="btn-icon">＋</span>新增用户</button>
      </div>
    </div>
    <div class="segmented-tabs">
      <button :class="{ active: permissionTab === 'users' }" @click="$emit('navigate-tab', 'users')">用户与角色</button>
      <button :class="{ active: permissionTab === 'resources' }" @click="$emit('navigate-tab', 'resources')">菜单与资源权限</button>
    </div>

    <template v-if="permissionTab === 'users'">
      <div class="role-grid">
        <article v-for="role in roleCards" :key="role.code" :class="['role-card', role.tone]">
          <span class="role-line-icon"><SvgIcon :name="role.icon" /></span>
          <div>
            <strong>{{ role.name }}</strong>
            <small>{{ role.code }}</small>
            <p>{{ role.desc }}</p>
          </div>
        </article>
      </div>

      <div class="work-layout permission-workspace">
        <section v-if="canManage">
          <h3>账号列表</h3>
          <section v-if="userFormOpen" class="editor-panel" tabindex="-1" @focusout="closeEditorOnFocusOut">
            <h3>{{ newUser.id ? '编辑用户' : '新增用户' }}</h3>
            <div class="form-grid user-form">
              <input v-model="newUser.username" placeholder="登录账号" />
              <input v-model="newUser.displayName" placeholder="姓名 / 显示名" />
              <input v-model="newUser.password" type="password" :placeholder="newUser.id ? '新密码（留空不修改）' : '初始密码'" />
              <select v-model="newUser.role" :disabled="newUser.id === currentUserId && newUser.role === 'super_admin' && displayUsers.filter((user) => user.roles?.includes('super_admin')).length <= 1"><option value="ops_engineer">运维工程师</option><option value="super_admin">超级管理员</option></select>
              <label class="inline-check"><input v-model="newUser.mustChangePassword" type="checkbox" />首次登录修改密码</label>
              <button class="primary" @click="$emit('save-user')">{{ newUser.id ? '保存用户' : '新增用户' }}</button>
              <button @click="$emit('close-user-form')">取消</button>
            </div>
          </section>
          <div class="table-wrap">
            <table>
              <thead><tr><th>账号</th><th>姓名</th><th>角色</th><th>密码状态</th><th>操作</th></tr></thead>
              <tbody>
                <tr v-for="user in displayUsers" :key="user.id">
                  <td>{{ user.username }}</td>
                  <td>{{ user.displayName }}</td>
                  <td><span v-for="role in user.roles" :key="role" class="pill">{{ role === 'super_admin' ? '超级管理员' : '运维工程师' }}</span></td>
                  <td><span :class="['pill', user.mustChangePassword ? 'danger' : 'success']">{{ user.mustChangePassword ? '需初始化' : '正常' }}</span></td>
                  <td class="row-actions"><button class="link" :disabled="isSample(user)" @click="$emit('edit-user', user)">编辑</button><button class="link danger-text" :disabled="user.id === currentUserId || isSample(user)" @click="$emit('delete-user', user)">删除</button></td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
        <section v-else class="access-note">
          <h3>当前权限</h3>
          <p class="muted">当前账号仅可查看一期权限边界说明。用户新增、角色调整、密码初始化策略和菜单资源权限由超级管理员维护。</p>
        </section>
        <aside class="detail-card">
          <h3>敏感凭据策略</h3>
          <p class="muted">资产与实例账号密码单独加密存储，列表页不显示。查看前必须输入统一二次校验密码。</p>
          <dl>
            <dt>登录方式</dt><dd>一期账号密码</dd>
            <dt>初始账号</dt><dd>admin / 首次登录修改初始化密码</dd>
            <dt>SSO/LDAP</dt><dd>灰度占位，暂不接入</dd>
            <dt>凭据查看</dt><dd>{{ credentialVerification.hasPassword ? '已配置统一校验密码' : '未配置，暂回退登录密码' }}</dd>
          </dl>
          <section v-if="canManage" class="credential-policy">
            <h4>统一二次校验密码</h4>
            <input v-model="credentialVerification.password" type="password" placeholder="设置统一校验密码" />
            <input v-model="credentialVerification.confirm" type="password" placeholder="再次确认校验密码" />
            <button class="primary" @click="$emit('save-credential-password')">{{ credentialVerification.hasPassword ? '更新校验密码' : '设置校验密码' }}</button>
            <p v-if="credentialVerification.message" :class="['inline', credentialVerification.message.includes('失败') || credentialVerification.message.includes('不') || credentialVerification.message.includes('至少') || credentialVerification.message.includes('无权') ? 'error' : 'muted']">{{ credentialVerification.message }}</p>
          </section>
        </aside>
      </div>
    </template>

    <template v-else>
      <section class="permission-matrix">
        <h3>菜单授权概览</h3>
        <div class="table-wrap">
          <table>
            <thead><tr><th>菜单 / 资源</th><th>阶段</th><th>超级管理员</th><th>运维工程师</th><th>说明</th></tr></thead>
            <tbody>
              <tr v-for="row in menuPermissionRows" :key="row.menu">
                <td>{{ row.menu }}</td>
                <td><span :class="['pill', row.stage.includes('灰度') ? '' : 'success']">{{ row.stage }}</span></td>
                <td>{{ row.superAdmin }}</td>
                <td>{{ row.opsEngineer }}</td>
                <td>{{ row.note }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="permission-matrix">
        <h3>资源权限矩阵</h3>
        <div class="table-wrap">
          <table>
            <thead><tr><th>功能范围</th><th>超级管理员</th><th>运维工程师</th><th>说明</th></tr></thead>
            <tbody>
              <tr v-for="row in permissionRows" :key="row.scope">
                <td>{{ row.scope }}</td>
                <td><span class="pill success">{{ row.admin }}</span></td>
                <td><span class="pill">{{ row.ops }}</span></td>
                <td>{{ row.note }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <div class="permission-grid gray-roles">
        <article><strong>灰度角色</strong><span>管理人员、SRE、研发、值班人员、只读访客先保留名称，后续再扩展细粒度权限。</span></article>
        <article><strong>认证灰度</strong><span>SSO/LDAP 保留配置入口，一期不影响账号密码登录上线。</span></article>
      </div>
    </template>
  </section>
</template>
