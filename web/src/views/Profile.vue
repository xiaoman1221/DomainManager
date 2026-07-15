<template>
  <div class="profile">
    <!-- Profile Card -->
    <el-card class="profile-card">
      <template #header>
        <span>个人设置</span>
      </template>
      <div class="profile-header">
        <el-avatar :size="80" :src="avatarUrl" icon="UserFilled" />
        <div class="profile-info">
          <h3>{{ userStore.user?.username }}</h3>
          <p class="role-tag">
            <el-tag :type="userStore.user?.role === 'admin' ? 'danger' : 'info'" size="small">
              {{ userStore.user?.role === 'admin' ? '管理员' : '普通用户' }}
            </el-tag>
          </p>
          <p class="email">{{ userStore.user?.email }}</p>
        </div>
      </div>

      <el-divider />

      <el-form ref="profileFormRef" :model="profileForm" :rules="profileRules" label-width="80px">
        <el-form-item label="用户名">
          <el-input :value="userStore.user?.username" disabled />
        </el-form-item>
        <el-form-item label="昵称" prop="nickname">
          <el-input v-model="profileForm.nickname" placeholder="设置昵称" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="profileForm.email" placeholder="邮箱地址" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSaveProfile" :loading="savingProfile">保存资料</el-button>
        </el-form-item>
      </el-form>

      <el-divider />

      <h4 style="margin-bottom: 16px">修改密码</h4>
      <el-form ref="passwordFormRef" :model="passwordForm" :rules="passwordRules" label-width="100px">
        <el-form-item label="当前密码" prop="old_password">
          <el-input v-model="passwordForm.old_password" type="password" show-password />
        </el-form-item>
        <el-form-item label="新密码" prop="new_password">
          <el-input v-model="passwordForm.new_password" type="password" show-password />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirm_password">
          <el-input v-model="passwordForm.confirm_password" type="password" show-password />
        </el-form-item>
        <el-form-item>
          <el-button type="warning" @click="handleChangePassword" :loading="changingPassword">修改密码</el-button>
        </el-form-item>
      </el-form>

      <el-divider />

      <el-button type="danger" @click="handleLogout">
        <el-icon><SwitchButton /></el-icon>退出登录
      </el-button>
    </el-card>

    <!-- Admin Settings (only for admins) -->
    <el-card v-if="userStore.user?.role === 'admin'" class="admin-card">
      <template #header>
        <span>系统管理</span>
      </template>

      <el-tabs v-model="adminTab">
        <el-tab-pane label="用户管理" name="users">
          <el-table :data="users" style="width: 100%" v-loading="usersLoading" empty-text="暂无用户">
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column prop="username" label="用户名" width="140" />
            <el-table-column prop="email" label="邮箱" min-width="200" />
            <el-table-column prop="nickname" label="昵称" width="120" />
            <el-table-column label="角色" width="120">
              <template #default="{ row }">
                <el-select v-model="row.role" size="small" @change="handleRoleChange(row)" :disabled="row.username === 'admin'">
                  <el-option label="管理员" value="admin" />
                  <el-option label="普通用户" value="user" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column label="注册时间" width="180">
              <template #default="{ row }">
                {{ new Date(row.created_at).toLocaleString('zh-CN') }}
              </template>
            </el-table-column>
            <el-table-column label="操作" width="160" fixed="right">
              <template #default="{ row }">
                <el-button text type="primary" size="small" @click="showResetPassword(row)">重置密码</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="系统信息" name="info">
          <el-descriptions :column="2" border v-if="sysInfo">
            <el-descriptions-item label="域名总数">{{ sysInfo.domains }}</el-descriptions-item>
            <el-descriptions-item label="证书总数">{{ sysInfo.certificates }}</el-descriptions-item>
            <el-descriptions-item label="注册商数量">{{ sysInfo.registrars }}</el-descriptions-item>
            <el-descriptions-item label="用户数量">{{ sysInfo.users }}</el-descriptions-item>
            <el-descriptions-item label="系统版本">{{ sysInfo.version }}</el-descriptions-item>
          </el-descriptions>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- Reset Password Dialog -->
    <el-dialog v-model="resetPwdVisible" title="重置用户密码" width="400px">
      <el-form :model="resetPwdForm" label-width="80px">
        <el-form-item label="用户">
          <el-input :value="resetPwdUser?.username" disabled />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="resetPwdForm.password" type="password" show-password placeholder="输入新密码" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetPwdVisible = false">取消</el-button>
        <el-button type="primary" @click="handleResetPassword" :loading="resettingPwd">确认</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useUserStore } from '../stores/user'
import { updateUserProfile, changePassword, getUsers, updateUserRole, adminUpdatePassword, getSystemInfo } from '../api/settings'

const router = useRouter()
const userStore = useUserStore()

const adminTab = ref('users')
const users = ref([])
const sysInfo = ref(null)
const usersLoading = ref(false)
const savingProfile = ref(false)
const changingPassword = ref(false)

const profileFormRef = ref(null)
const passwordFormRef = ref(null)

const profileForm = ref({
  nickname: '',
  email: ''
})

const passwordForm = ref({
  old_password: '',
  new_password: '',
  confirm_password: ''
})

const profileRules = {
  email: [{ required: true, message: '请输入邮箱', trigger: 'blur' }, { type: 'email', message: '邮箱格式不正确', trigger: 'blur' }]
}

const validateConfirm = (rule, value, callback) => {
  if (value !== passwordForm.value.new_password) {
    callback(new Error('两次输入密码不一致'))
  } else {
    callback()
  }
}

const passwordRules = {
  old_password: [{ required: true, message: '请输入当前密码', trigger: 'blur' }],
  new_password: [{ required: true, message: '请输入新密码', trigger: 'blur' }, { min: 6, message: '密码至少6位', trigger: 'blur' }],
  confirm_password: [{ required: true, message: '请确认密码', trigger: 'blur' }, { validator: validateConfirm, trigger: 'blur' }]
}

const resetPwdVisible = ref(false)
const resetPwdUser = ref(null)
const resetPwdForm = ref({ password: '' })
const resettingPwd = ref(false)

const avatarUrl = computed(() => {
  const email = userStore.user?.email
  if (!email) return ''
  const hash = simpleMD5(email.trim().toLowerCase())
  return `https://www.gravatar.com/avatar/${hash}?d=mp`
})

function simpleMD5(str) {
  // Use crypto.subtle for MD5 - fallback to simple hash
  const encoder = new TextEncoder()
  const data = encoder.encode(str)
  let h1 = 0xdeadbeef, h2 = 0x41c6ce57
  for (let i = 0; i < str.length; i++) {
    const ch = str.charCodeAt(i)
    h1 = Math.imul(h1 ^ ch, 2654435761)
    h2 = Math.imul(h2 ^ ch, 1597334677)
  }
  h1 = Math.imul(h1 ^ (h1 >>> 16), 2246822507)
  h1 ^= Math.imul(h2 ^ (h2 >>> 13), 3266489909)
  h2 = Math.imul(h2 ^ (h2 >>> 16), 2246822507)
  h2 ^= Math.imul(h1 ^ (h1 >>> 13), 3266489909)
  const hash = 4294967296 * (2097151 & h2) + (h1 >>> 0)
  return hash.toString(16).padStart(12, '0')
}

onMounted(() => {
  if (userStore.user) {
    profileForm.value.nickname = userStore.user.nickname || ''
    profileForm.value.email = userStore.user.email || ''
  }
  if (userStore.user?.role === 'admin') {
    fetchUsers()
    fetchSysInfo()
  }
})

async function fetchUsers() {
  usersLoading.value = true
  try {
    const res = await getUsers()
    users.value = res.data || []
  } finally {
    usersLoading.value = false
  }
}

async function fetchSysInfo() {
  try {
    sysInfo.value = await getSystemInfo()
  } catch {}
}

async function handleSaveProfile() {
  try {
    await profileFormRef.value.validate()
  } catch { return }
  savingProfile.value = true
  try {
    await updateUserProfile(profileForm.value)
    await userStore.fetchProfile()
    ElMessage.success('资料已更新')
  } finally {
    savingProfile.value = false
  }
}

async function handleChangePassword() {
  try {
    await passwordFormRef.value.validate()
  } catch { return }
  changingPassword.value = true
  try {
    await changePassword({
      old_password: passwordForm.value.old_password,
      new_password: passwordForm.value.new_password
    })
    ElMessage.success('密码已修改')
    passwordForm.value = { old_password: '', new_password: '', confirm_password: '' }
  } finally {
    changingPassword.value = false
  }
}

function showResetPassword(row) {
  resetPwdUser.value = row
  resetPwdForm.value = { password: '' }
  resetPwdVisible.value = true
}

async function handleResetPassword() {
  if (!resetPwdForm.value.password || resetPwdForm.value.password.length < 6) {
    ElMessage.warning('密码至少6位')
    return
  }
  resettingPwd.value = true
  try {
    await adminUpdatePassword(resetPwdUser.value.id, { password: resetPwdForm.value.password })
    ElMessage.success('密码已重置')
    resetPwdVisible.value = false
  } finally {
    resettingPwd.value = false
  }
}

async function handleRoleChange(row) {
  await updateUserRole(row.id, { role: row.role })
  ElMessage.success('角色已更新')
  fetchUsers()
}

function handleLogout() {
  userStore.logout()
  router.push('/login')
}
</script>

<style scoped>
.profile {
  max-width: 800px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.profile-header {
  display: flex;
  align-items: center;
  gap: 20px;
}
.profile-info h3 {
  margin: 0 0 4px 0;
  font-size: 18px;
}
.profile-info .email {
  color: #909399;
  margin: 4px 0 0 0;
  font-size: 14px;
}
.role-tag { margin: 0; }
</style>
