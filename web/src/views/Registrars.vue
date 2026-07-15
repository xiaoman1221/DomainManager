<template>
  <div class="registrar">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>注册商管理</span>
          <div class="header-actions">
            <el-button size="small" @click="handleExportRegistrars">
              <el-icon><Download /></el-icon>导出
            </el-button>
            <el-upload
              ref="regUploadRef"
              :auto-upload="false"
              :show-file-list="false"
              accept=".csv"
              :on-change="handleImportRegistrarsFile"
            >
              <el-button size="small">
                <el-icon><Upload /></el-icon>导入
              </el-button>
            </el-upload>
            <el-button type="primary" @click="showDialog()">
              <el-icon><Plus /></el-icon>添加注册商
            </el-button>
          </div>
        </div>
      </template>

      <el-table :data="registrars" style="width: 100%" v-loading="loading" empty-text="暂无注册商配置">
        <el-table-column prop="name" label="名称" width="160" />
        <el-table-column label="类型" width="160">
          <template #default="{ row }">
            <el-tag :type="row.type.startsWith('aliyun') || row.type.startsWith('tencent') || row.type.startsWith('huawei') ? 'danger' : 'primary'" size="small">
              {{ typeLabel(row.type) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="地区" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="isCN(row.type)" type="warning" size="small">国内</el-tag>
            <el-tag v-else size="small">国际</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="API配置" min-width="200">
          <template #default="{ row }">
            <span v-if="row.api_key" style="color: #67C23A">
              <el-icon><Check /></el-icon> 已配置
            </span>
            <span v-else style="color: #909399">
              <el-icon><Close /></el-icon> 未配置
            </span>
          </template>
        </el-table-column>
        <el-table-column label="自动同步" width="100" align="center">
          <template #default="{ row }">
            <el-switch v-model="row.sync_enabled" size="small" @change="toggleSync(row)" />
          </template>
        </el-table-column>
        <el-table-column label="启用" width="80" align="center">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" size="small" @change="toggleEnabled(row)" />
          </template>
        </el-table-column>
        <el-table-column label="最后同步" width="170">
          <template #default="{ row }">
            {{ row.last_sync_at ? new Date(row.last_sync_at).toLocaleString('zh-CN') : '从未同步' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button text type="primary" size="small" @click="showImportDialog(row)">导入域名</el-button>
            <el-button text type="primary" size="small" @click="showDialog(row)">编辑</el-button>
            <el-popconfirm title="确定删除此注册商？" @confirm="handleDelete(row.id)">
              <template #reference>
                <el-button text type="danger" size="small">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Add/Edit Dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑注册商' : '添加注册商'"
      width="600px"
      destroy-on-close
    >
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="100px">
        <el-form-item label="注册商类型" prop="type">
          <el-select v-model="form.type" placeholder="选择注册商类型" style="width: 100%" @change="onTypeChange">
            <el-option-group label="国内">
              <el-option v-for="t in cnTypes" :key="t.value" :label="t.label" :value="t.value" />
            </el-option-group>
            <el-option-group label="国际">
              <el-option v-for="t in globalTypes" :key="t.value" :label="t.label" :value="t.value" />
            </el-option-group>
          </el-select>
        </el-form-item>
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="自定义名称" />
        </el-form-item>
        <el-form-item label="API端点">
          <el-input v-model="form.api_endpoint" placeholder="API地址（可选，使用默认）" />
        </el-form-item>
        <el-form-item label="API Key">
          <el-input v-model="form.api_key" placeholder="API Key / AccessKey ID" show-password />
        </el-form-item>
        <el-form-item label="API Secret">
          <el-input v-model="form.api_secret" placeholder="API Secret / AccessKey Secret" show-password />
        </el-form-item>
        <el-form-item label="额外参数">
          <el-input v-model="form.api_extra" placeholder="其他参数（如用户名等）" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <el-form-item label="自动同步">
          <el-switch v-model="form.sync_enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitLoading" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>

    <!-- Import Dialog -->
    <el-dialog
      v-model="importDialogVisible"
      title="导入域名"
      width="600px"
      destroy-on-close
    >
      <el-alert type="info" :closable="false" show-icon style="margin-bottom: 16px">
        从 <strong>{{ currentRegistrar?.name }}</strong> 导入域名到系统中。填写域名列表或留空自动获取。
      </el-alert>
      <el-form label-width="100px">
        <el-form-item label="注册商">
          <el-input :model-value="currentRegistrar?.name" disabled />
        </el-form-item>
        <el-form-item label="域名列表">
          <el-input
            v-model="importDomainsText"
            type="textarea"
            :rows="8"
            placeholder="每行一个域名，例如：&#10;example.com&#10;test.org&#10;my-site.net&#10;&#10;留空则尝试从注册商API自动获取"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="importDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="importLoading" @click="handleImport">
          <el-icon><Download /></el-icon>开始导入
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  getRegistrars,
  createRegistrar,
  updateRegistrar,
  deleteRegistrar,
  getRegistrarTypes,
  importDomains,
  exportRegistrars,
  importRegistrarsCSV,
} from '../api/registrar'

const registrars = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const importDialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(null)
const formRef = ref(null)
const submitLoading = ref(false)
const importLoading = ref(false)
const currentRegistrar = ref(null)
const importDomainsText = ref('')
const regUploadRef = ref(null)

const cnTypes = ref([])
const globalTypes = ref([])

const form = reactive({
  name: '',
  type: '',
  api_endpoint: '',
  api_key: '',
  api_secret: '',
  api_extra: '',
  enabled: true,
  sync_enabled: false,
})

const formRules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }],
}

function typeLabel(type) {
  const all = [...cnTypes.value, ...globalTypes.value]
  const found = all.find((t) => t.value === type)
  return found ? found.label : type
}

function isCN(type) {
  return cnTypes.value.some((t) => t.value === type)
}

function onTypeChange(val) {
  if (!form.name) {
    const all = [...cnTypes.value, ...globalTypes.value]
    const found = all.find((t) => t.value === val)
    if (found) form.name = found.label
  }
}

function showDialog(row) {
  if (row) {
    isEdit.value = true
    editId.value = row.id
    form.name = row.name
    form.type = row.type
    form.api_endpoint = row.api_endpoint || ''
    form.api_key = row.api_key || ''
    form.api_secret = row.api_secret || ''
    form.api_extra = row.api_extra || ''
    form.enabled = row.enabled
    form.sync_enabled = row.sync_enabled
  } else {
    isEdit.value = false
    editId.value = null
    form.name = ''
    form.type = ''
    form.api_endpoint = ''
    form.api_key = ''
    form.api_secret = ''
    form.api_extra = ''
    form.enabled = true
    form.sync_enabled = false
  }
  dialogVisible.value = true
}

function showImportDialog(row) {
  currentRegistrar.value = row
  importDomainsText.value = ''
  importDialogVisible.value = true
}

async function fetchRegistrars() {
  loading.value = true
  try {
    const res = await getRegistrars()
    registrars.value = res.data || []
  } finally {
    loading.value = false
  }
}

async function fetchTypes() {
  try {
    const res = await getRegistrarTypes()
    const all = res.data || []
    cnTypes.value = all.filter((t) => t.region === 'cn')
    globalTypes.value = all.filter((t) => t.region === 'global' || t.region === 'other')
  } catch {
    cnTypes.value = [
      { value: 'aliyun', label: '阿里云（万网）' },
      { value: 'tencent', label: '腾讯云（DNSPod）' },
      { value: 'huawei', label: '华为云' },
    ]
    globalTypes.value = [
      { value: 'aliyun_intl', label: '阿里云（国际）' },
      { value: 'tencent_intl', label: '腾讯云（国际）' },
      { value: 'cloudflare', label: 'Cloudflare' },
      { value: 'godaddy', label: 'GoDaddy' },
      { value: 'namecheap', label: 'Namecheap' },
      { value: 'namesilo', label: 'NameSilo' },
      { value: 'dynadot', label: 'Dynadot' },
      { value: 'porkbun', label: 'Porkbun' },
      { value: 'other', label: '其他（手动导入）' },
    ]
  }
}

async function handleSubmit() {
  await formRef.value.validate()
  submitLoading.value = true
  try {
    if (isEdit.value) {
      await updateRegistrar(editId.value, form)
      ElMessage.success('更新成功')
    } else {
      await createRegistrar(form)
      ElMessage.success('添加成功')
    }
    dialogVisible.value = false
    fetchRegistrars()
  } finally {
    submitLoading.value = false
  }
}

async function handleDelete(id) {
  await deleteRegistrar(id)
  ElMessage.success('删除成功')
  fetchRegistrars()
}

async function toggleSync(row) {
  await updateRegistrar(row.id, { sync_enabled: row.sync_enabled })
}

async function toggleEnabled(row) {
  await updateRegistrar(row.id, { enabled: row.enabled })
}

async function handleImport() {
  importLoading.value = true
  try {
    const res = await importDomains({
      registrar_id: currentRegistrar.value.id,
      domains: importDomainsText.value.trim(),
    })
    if (res.error) {
      ElMessage.error(res.error)
    } else if (res.imported === 0 && res.updated === 0 && res.skipped === 0 && (res.refreshed || 0) === 0) {
      ElMessage.warning('未导入任何域名。支持自动导入的注册商: 阿里云、腾讯云、Cloudflare、GoDaddy、Namecheap、NameSilo、Dynadot。其他类型请手动输入域名列表。')
    } else {
      ElMessage.success(
        res.message || `导入完成: 新增 ${res.imported} 个, 更新 ${res.updated} 个, 刷新 ${res.refreshed || 0} 个, 跳过 ${res.skipped} 个`
      )
    }
    importDialogVisible.value = false
    fetchRegistrars()
  } finally {
    importLoading.value = false
  }
}

// --- Export / Import ---
async function handleExportRegistrars() {
  try {
    const res = await exportRegistrars()
    const blob = new Blob([res], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'registrars_export.csv'
    a.click()
    URL.revokeObjectURL(url)
    ElMessage.success('导出成功')
  } catch (e) {
    ElMessage.error('导出失败: ' + (e.message || '未知错误'))
  }
}

async function handleImportRegistrarsFile(file) {
  const formData = new FormData()
  formData.append('file', file.raw)
  try {
    const res = await importRegistrarsCSV(formData)
    ElMessage.success(res.message || `导入完成: ${res.created} 个注册商`)
    fetchRegistrars()
  } catch (e) {
    ElMessage.error('导入失败: ' + (e.response?.data?.error || e.message || '未知错误'))
  }
}

onMounted(() => {
  fetchRegistrars()
  fetchTypes()
})
</script>

<style scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
