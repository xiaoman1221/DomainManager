<template>
  <div class="notifications">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>通知管理</span>
          <div class="header-right">
            <el-button size="small" type="warning" @click="handleSendExpiry" :loading="sendingExpiry">
              <el-icon><Bell /></el-icon>发送到期提醒
            </el-button>
            <el-button type="primary" @click="showChannelDialog()">
              <el-icon><Plus /></el-icon>添加通知渠道
            </el-button>
          </div>
        </div>
      </template>

      <el-tabs v-model="activeTab">
        <el-tab-pane label="通知渠道" name="channels">
          <el-table :data="channels" style="width: 100%" v-loading="loading" empty-text="暂无通知渠道">
            <el-table-column prop="name" label="名称" width="180" />
            <el-table-column label="类型" width="140">
              <template #default="{ row }">
                <el-tag :type="typeColor(row.type)" size="small">{{ typeLabel(row.type) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-switch v-model="row.enabled" size="small" @change="handleToggle(row)" />
              </template>
            </el-table-column>
            <el-table-column label="创建时间" width="180">
              <template #default="{ row }">
                {{ new Date(row.created_at).toLocaleString('zh-CN') }}
              </template>
            </el-table-column>
            <el-table-column label="操作" width="220" fixed="right">
              <template #default="{ row }">
                <el-button text type="success" size="small" @click="showTestDialog(row)">测试</el-button>
                <el-button text type="primary" size="small" @click="showChannelDialog(row)">编辑</el-button>
                <el-popconfirm title="确定删除此通知渠道？" @confirm="handleDeleteChannel(row.id)">
                  <template #reference>
                    <el-button text type="danger" size="small">删除</el-button>
                  </template>
                </el-popconfirm>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="发送记录" name="logs">
          <el-table :data="logs" style="width: 100%" v-loading="logsLoading" empty-text="暂无发送记录">
            <el-table-column label="渠道" width="140">
              <template #default="{ row }">
                {{ getChannelName(row.channel_id) }}
              </template>
            </el-table-column>
            <el-table-column prop="title" label="标题" min-width="160" />
            <el-table-column prop="content" label="内容" min-width="250" show-overflow-tooltip />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">
                  {{ row.status === 'success' ? '成功' : '失败' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="error" label="错误信息" width="200" show-overflow-tooltip />
            <el-table-column label="时间" width="180">
              <template #default="{ row }">
                {{ new Date(row.created_at).toLocaleString('zh-CN') }}
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- Add/Edit Channel Dialog -->
    <el-dialog v-model="channelDialogVisible" :title="isEditChannel ? '编辑通知渠道' : '添加通知渠道'" width="550px" destroy-on-close>
      <el-form ref="channelFormRef" :model="channelForm" :rules="channelRules" label-width="100px">
        <el-form-item label="渠道名称" prop="name">
          <el-input v-model="channelForm.name" placeholder="例如：Bark 推送" />
        </el-form-item>
        <el-form-item label="通知类型" prop="type">
          <el-select v-model="channelForm.type" placeholder="选择类型" style="width: 100%" @change="onTypeChange">
            <el-option v-for="t in notificationTypes" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
        </el-form-item>

        <!-- Bark Config -->
        <template v-if="channelForm.type === 'bark'">
          <el-form-item label="服务器">
            <el-input v-model="barkConfig.server" placeholder="https://api.day.app" />
          </el-form-item>
          <el-form-item label="Key" prop="config">
            <el-input v-model="barkConfig.key" placeholder="your-bark-key" />
          </el-form-item>
          <el-form-item label="分组">
            <el-input v-model="barkConfig.group" placeholder="域名管理" />
          </el-form-item>
        </template>

        <!-- Telegram Config -->
        <template v-if="channelForm.type === 'telegram'">
          <el-form-item label="Bot Token" prop="config">
            <el-input v-model="telegramConfig.bot_token" placeholder="123456:ABC-DEF..." />
          </el-form-item>
          <el-form-item label="Chat ID" prop="config">
            <el-input v-model="telegramConfig.chat_id" placeholder="-1001234567890" />
          </el-form-item>
        </template>

        <!-- Email Config -->
        <template v-if="channelForm.type === 'email'">
          <el-form-item label="SMTP 服务器">
            <el-input v-model="emailConfig.smtp_host" placeholder="smtp.gmail.com" />
          </el-form-item>
          <el-form-item label="SMTP 端口">
            <el-input v-model="emailConfig.smtp_port" placeholder="587" />
          </el-form-item>
          <el-form-item label="用户名">
            <el-input v-model="emailConfig.username" placeholder="your-email@gmail.com" />
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="emailConfig.password" type="password" show-password />
          </el-form-item>
          <el-form-item label="发件人">
            <el-input v-model="emailConfig.from" placeholder="your-email@gmail.com" />
          </el-form-item>
          <el-form-item label="收件人">
            <el-input v-model="emailConfig.to" placeholder="多个邮箱用逗号分隔" />
          </el-form-item>
          <el-form-item label="启用 TLS">
            <el-switch v-model="emailConfig.use_tls" />
          </el-form-item>
        </template>

        <!-- Webhook Config -->
        <template v-if="channelForm.type === 'webhook'">
          <el-form-item label="URL" prop="config">
            <el-input v-model="webhookConfig.url" placeholder="https://your-webhook-url.com/notify" />
          </el-form-item>
          <el-form-item label="请求方法">
            <el-select v-model="webhookConfig.method" style="width: 100%">
              <el-option label="POST" value="POST" />
              <el-option label="PUT" value="PUT" />
            </el-select>
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="channelDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSaveChannel" :loading="savingChannel">保存</el-button>
      </template>
    </el-dialog>

    <!-- Test Dialog -->
    <el-dialog v-model="testDialogVisible" title="测试通知" width="450px">
      <el-form :model="testForm" label-width="60px">
        <el-form-item label="标题">
          <el-input v-model="testForm.title" placeholder="测试通知" />
        </el-form-item>
        <el-form-item label="内容">
          <el-input v-model="testForm.content" type="textarea" :rows="3" placeholder="这是一条测试通知" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="testDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleTest" :loading="testing">发送测试</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import {
  getNotificationTypes, getNotificationChannels, createNotificationChannel,
  updateNotificationChannel, deleteNotificationChannel, toggleNotificationChannel,
  testNotificationChannel, getNotificationLogs, sendExpiryNotifications
} from '../api/notification'

const activeTab = ref('channels')
const channels = ref([])
const logs = ref([])
const loading = ref(false)
const logsLoading = ref(false)
const notificationTypes = ref([])

const channelDialogVisible = ref(false)
const isEditChannel = ref(false)
const editChannelId = ref(null)
const savingChannel = ref(false)
const channelFormRef = ref(null)

const testDialogVisible = ref(false)
const testing = ref(false)
const testChannelId = ref(null)

const sendingExpiry = ref(false)

const channelForm = ref({ name: '', type: 'bark', config: '' })
const channelRules = {
  name: [{ required: true, message: '请输入渠道名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择通知类型', trigger: 'change' }]
}

const barkConfig = reactive({ server: '', key: '', group: '域名管理' })
const telegramConfig = reactive({ bot_token: '', chat_id: '' })
const emailConfig = reactive({ smtp_host: '', smtp_port: '587', username: '', password: '', from: '', to: '', use_tls: true })
const webhookConfig = reactive({ url: '', method: 'POST' })

const testForm = ref({ title: '测试通知', content: '这是一条来自 Domain Manager 的测试通知' })

onMounted(() => {
  fetchChannels()
  fetchLogs()
  fetchTypes()
})

async function fetchTypes() {
  try {
    const res = await getNotificationTypes()
    notificationTypes.value = res.data || []
  } catch {}
}

async function fetchChannels() {
  loading.value = true
  try {
    const res = await getNotificationChannels()
    channels.value = res.data || []
  } finally {
    loading.value = false
  }
}

async function fetchLogs() {
  logsLoading.value = true
  try {
    const res = await getNotificationLogs()
    logs.value = res.data || []
  } finally {
    logsLoading.value = false
  }
}

function typeLabel(type) {
  const map = { bark: 'Bark', telegram: 'Telegram', email: '邮件', webhook: 'Webhook' }
  return map[type] || type
}

function typeColor(type) {
  const map = { bark: 'success', telegram: 'primary', email: 'warning', webhook: 'info' }
  return map[type] || 'info'
}

function getChannelName(id) {
  const ch = channels.value.find(c => c.id === id)
  return ch ? ch.name : '未知'
}

function onTypeChange() {
  channelForm.value.config = ''
}

function buildConfig() {
  const type = channelForm.value.type
  if (type === 'bark') return JSON.stringify(barkConfig)
  if (type === 'telegram') return JSON.stringify(telegramConfig)
  if (type === 'email') return JSON.stringify(emailConfig)
  if (type === 'webhook') return JSON.stringify(webhookConfig)
  return ''
}

function parseConfig(configStr, type) {
  try {
    const obj = JSON.parse(configStr || '{}')
    if (type === 'bark') { Object.assign(barkConfig, obj) }
    if (type === 'telegram') { Object.assign(telegramConfig, obj) }
    if (type === 'email') { Object.assign(emailConfig, obj) }
    if (type === 'webhook') { Object.assign(webhookConfig, obj) }
  } catch {}
}

function showChannelDialog(row) {
  isEditChannel.value = !!row
  editChannelId.value = row?.id
  if (row) {
    channelForm.value = { name: row.name, type: row.type, config: row.config || '' }
    parseConfig(row.config, row.type)
  } else {
    channelForm.value = { name: '', type: 'bark', config: '' }
    Object.assign(barkConfig, { server: '', key: '', group: '域名管理' })
    Object.assign(telegramConfig, { bot_token: '', chat_id: '' })
    Object.assign(emailConfig, { smtp_host: '', smtp_port: '587', username: '', password: '', from: '', to: '', use_tls: true })
    Object.assign(webhookConfig, { url: '', method: 'POST' })
  }
  channelDialogVisible.value = true
}

function showTestDialog(row) {
  testChannelId.value = row.id
  testForm.value = { title: '测试通知', content: '这是一条来自 Domain Manager 的测试通知' }
  testDialogVisible.value = true
}

async function handleSaveChannel() {
  try {
    await channelFormRef.value.validate()
  } catch { return }
  savingChannel.value = true
  try {
    const data = { ...channelForm.value, config: buildConfig() }
    if (isEditChannel.value) {
      await updateNotificationChannel(editChannelId.value, data)
      ElMessage.success('更新成功')
    } else {
      await createNotificationChannel(data)
      ElMessage.success('创建成功')
    }
    channelDialogVisible.value = false
    fetchChannels()
  } finally {
    savingChannel.value = false
  }
}

async function handleDeleteChannel(id) {
  await deleteNotificationChannel(id)
  ElMessage.success('删除成功')
  fetchChannels()
}

async function handleToggle(row) {
  await toggleNotificationChannel(row.id)
  ElMessage.success(row.enabled ? '已启用' : '已禁用')
}

async function handleTest() {
  testing.value = true
  try {
    await testNotificationChannel(testChannelId.value, testForm.value)
    ElMessage.success('测试发送成功')
    testDialogVisible.value = false
    fetchLogs()
  } finally {
    testing.value = false
  }
}

async function handleSendExpiry() {
  sendingExpiry.value = true
  try {
    const res = await sendExpiryNotifications()
    ElMessage.success(`到期提醒已发送，成功 ${res.sent || 0} 个渠道`)
    fetchLogs()
  } finally {
    sendingExpiry.value = false
  }
}
</script>

<style scoped>
.notifications { max-width: 100%; }
.card-header { display: flex; align-items: center; justify-content: space-between; }
.header-right { display: flex; align-items: center; gap: 8px; }
</style>
