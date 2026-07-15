<template>
  <div class="certificates">
    <el-card>
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <el-input
              v-model="keyword"
              placeholder="搜索域名或颁发者..."
              prefix-icon="Search"
              clearable
              style="width: 240px; margin-right: 12px"
              @keyup.enter="fetchCerts"
              @clear="fetchCerts"
            />
            <el-select v-model="statusFilter" placeholder="状态筛选" clearable style="width: 150px" @change="fetchCerts">
              <el-option label="正常" value="active" />
              <el-option label="已过期" value="expired" />
              <el-option label="30天内到期" value="expiring_30" />
            </el-select>
          </div>
          <div class="header-right">
            <el-button size="small" @click="showCertimateConfig = true">
              <el-icon><Setting /></el-icon>Certimate 配置
            </el-button>
            <el-button size="small" type="success" @click="handleSync" :loading="syncing">
              <el-icon><Refresh /></el-icon>同步证书
            </el-button>
            <el-button type="primary" @click="showDialog()">
              <el-icon><Plus /></el-icon>添加证书
            </el-button>
          </div>
        </div>
      </template>

      <div class="stats-row" v-if="stats">
        <el-row :gutter="16">
          <el-col :span="6">
            <el-statistic title="证书总数" :value="stats.total" />
          </el-col>
          <el-col :span="6">
            <el-statistic title="正常" :value="stats.active" />
          </el-col>
          <el-col :span="6">
            <el-statistic title="已过期" :value="stats.expired" />
          </el-col>
          <el-col :span="6">
            <el-statistic title="即将到期" :value="stats.expiring_soon" />
          </el-col>
        </el-row>
      </div>

      <el-table :data="certs" style="width: 100%" v-loading="loading" empty-text="暂无证书">
        <el-table-column prop="domain" label="域名" min-width="200" />
        <el-table-column prop="issuer" label="颁发者" width="160" />
        <el-table-column label="密钥算法" width="120">
          <template #default="{ row }">
            <el-tag size="small">{{ row.key_algorithm || '-' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="生效时间" width="120">
          <template #default="{ row }">
            {{ row.not_before ? row.not_before.split('T')[0] : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="到期时间" width="120">
          <template #default="{ row }">
            <span :class="getExpiryClass(row)">{{ row.not_after ? row.not_after.split('T')[0] : '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'danger'" size="small">
              {{ row.status === 'active' ? '正常' : '已过期' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="来源" width="100">
          <template #default="{ row }">
            <el-tag :type="row.source === 'certimate' ? 'primary' : 'info'" size="small">
              {{ row.source === 'certimate' ? 'Certimate' : row.source || '手动' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button text type="primary" size="small" @click="showDialog(row)">编辑</el-button>
            <el-button text type="info" size="small" @click="viewDetail(row)">详情</el-button>
            <el-popconfirm title="确定删除此证书？" @confirm="handleDelete(row.id)">
              <template #reference>
                <el-button text type="danger" size="small">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Add/Edit Dialog -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑证书' : '添加证书'" width="650px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="100px">
        <el-form-item label="域名" prop="domain">
          <el-input v-model="form.domain" placeholder="example.com" />
        </el-form-item>
        <el-form-item label="颁发者">
          <el-input v-model="form.issuer" placeholder="Let's Encrypt" />
        </el-form-item>
        <el-form-item label="密钥算法">
          <el-select v-model="form.key_algorithm" placeholder="选择算法" style="width: 100%">
            <el-option label="RSA 2048" value="RSA 2048" />
            <el-option label="RSA 4096" value="RSA 4096" />
            <el-option label="ECDSA P-256" value="ECDSA P-256" />
            <el-option label="ECDSA P-384" value="ECDSA P-384" />
          </el-select>
        </el-form-item>
        <el-form-item label="生效时间">
          <el-date-picker v-model="form.not_before" type="date" placeholder="选择日期" value-format="YYYY-MM-DD" style="width: 100%" />
        </el-form-item>
        <el-form-item label="到期时间">
          <el-date-picker v-model="form.not_after" type="date" placeholder="选择日期" value-format="YYYY-MM-DD" style="width: 100%" />
        </el-form-item>
        <el-form-item label="SAN 域名">
          <el-input v-model="form.subject_alt_names" placeholder="多个域名用分号分隔" />
        </el-form-item>
        <el-form-item label="来源">
          <el-select v-model="form.source" style="width: 100%">
            <el-option label="Certimate" value="certimate" />
            <el-option label="手动添加" value="manual" />
            <el-option label="Let's Encrypt" value="letsencrypt" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.note" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave" :loading="saving">保存</el-button>
      </template>
    </el-dialog>

    <!-- Detail Dialog -->
    <el-dialog v-model="detailVisible" title="证书详情" width="650px">
      <el-descriptions :column="2" border v-if="detailCert">
        <el-descriptions-item label="域名" :span="2">{{ detailCert.domain }}</el-descriptions-item>
        <el-descriptions-item label="颁发者">{{ detailCert.issuer }}</el-descriptions-item>
        <el-descriptions-item label="密钥算法">{{ detailCert.key_algorithm }}</el-descriptions-item>
        <el-descriptions-item label="生效时间">{{ detailCert.not_before ? detailCert.not_before.split('T')[0] : '-' }}</el-descriptions-item>
        <el-descriptions-item label="到期时间">{{ detailCert.not_after ? detailCert.not_after.split('T')[0] : '-' }}</el-descriptions-item>
        <el-descriptions-item label="SAN 域名" :span="2">{{ detailCert.subject_alt_names || '-' }}</el-descriptions-item>
        <el-descriptions-item label="序列号" :span="2">{{ detailCert.serial_number || '-' }}</el-descriptions-item>
        <el-descriptions-item label="来源">{{ detailCert.source }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="detailCert.status === 'active' ? 'success' : 'danger'" size="small">
            {{ detailCert.status === 'active' ? '正常' : '已过期' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="备注" :span="2">{{ detailCert.note || '-' }}</el-descriptions-item>
        <el-descriptions-item label="创建时间" :span="2">{{ new Date(detailCert.created_at).toLocaleString('zh-CN') }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>

    <!-- Certimate Config Dialog -->
    <el-dialog v-model="showCertimateConfig" title="Certimate 配置" width="500px">
      <el-form :model="certimateForm" label-width="80px">
        <el-form-item label="API 地址">
          <el-input v-model="certimateForm.url" placeholder="http://127.0.0.1:8090" />
        </el-form-item>
        <el-form-item label="Token">
          <el-input v-model="certimateForm.token" type="password" show-password placeholder="PocketBase superuser token" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCertimateConfig = false">取消</el-button>
        <el-button type="primary" @click="saveCertimateForm" :loading="savingConfig">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  getCertificates, createCertificate, updateCertificate, deleteCertificate,
  getCertificateStats, getCertimateConfig, saveCertimateConfig, syncCertimateCertificates
} from '../api/certificate'

const certs = ref([])
const stats = ref(null)
const loading = ref(false)
const keyword = ref('')
const statusFilter = ref('')
const dialogVisible = ref(false)
const detailVisible = ref(false)
const detailCert = ref(null)
const isEdit = ref(false)
const editId = ref(null)
const saving = ref(false)
const syncing = ref(false)
const showCertimateConfig = ref(false)
const savingConfig = ref(false)
const formRef = ref(null)

const certimateForm = ref({ url: '', token: '' })

const form = ref({
  domain: '', issuer: '', key_algorithm: '', not_before: '', not_after: '',
  subject_alt_names: '', source: 'certimate', note: ''
})

const formRules = {
  domain: [{ required: true, message: '请输入域名', trigger: 'blur' }]
}

onMounted(() => {
  fetchCerts()
  fetchStats()
  fetchCertimateConfig()
})

async function fetchCerts() {
  loading.value = true
  try {
    const res = await getCertificates({ keyword: keyword.value, status: statusFilter.value })
    certs.value = res.data || []
  } finally {
    loading.value = false
  }
}

async function fetchStats() {
  try {
    stats.value = await getCertificateStats()
  } catch {}
}

async function fetchCertimateConfig() {
  try {
    const res = await getCertimateConfig()
    certimateForm.value = { url: res.url || '', token: res.token || '' }
  } catch {}
}

function getExpiryClass(row) {
  if (!row.not_after || row.status === 'expired') return 'text-danger'
  const d = new Date(row.not_after)
  const now = new Date()
  const diff = (d - now) / (1000 * 60 * 60 * 24)
  if (diff < 30) return 'text-warning'
  return ''
}

function showDialog(row) {
  isEdit.value = !!row
  editId.value = row?.id
  if (row) {
    form.value = {
      domain: row.domain || '', issuer: row.issuer || '',
      key_algorithm: row.key_algorithm || '',
      not_before: row.not_before ? row.not_before.split('T')[0] : '',
      not_after: row.not_after ? row.not_after.split('T')[0] : '',
      subject_alt_names: row.subject_alt_names || '', source: row.source || 'certimate',
      note: row.note || ''
    }
  } else {
    form.value = { domain: '', issuer: '', key_algorithm: '', not_before: '', not_after: '', subject_alt_names: '', source: 'certimate', note: '' }
  }
  dialogVisible.value = true
}

function viewDetail(row) {
  detailCert.value = row
  detailVisible.value = true
}

async function handleSave() {
  try {
    await formRef.value.validate()
  } catch { return }
  saving.value = true
  try {
    if (isEdit.value) {
      await updateCertificate(editId.value, form.value)
      ElMessage.success('更新成功')
    } else {
      await createCertificate(form.value)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    fetchCerts()
    fetchStats()
  } finally {
    saving.value = false
  }
}

async function handleDelete(id) {
  await deleteCertificate(id)
  ElMessage.success('删除成功')
  fetchCerts()
  fetchStats()
}

async function handleSync() {
  syncing.value = true
  try {
    const res = await syncCertimateCertificates()
    ElMessage.success(`同步完成，共 ${res.total || 0} 张证书`)
    fetchCerts()
    fetchStats()
  } finally {
    syncing.value = false
  }
}

async function saveCertimateForm() {
  savingConfig.value = true
  try {
    await saveCertimateConfig(certimateForm.value)
    ElMessage.success('配置已保存')
    showCertimateConfig.value = false
  } finally {
    savingConfig.value = false
  }
}
</script>

<style scoped>
.certificates { max-width: 100%; }
.card-header { display: flex; align-items: center; justify-content: space-between; }
.header-left { display: flex; align-items: center; }
.header-right { display: flex; align-items: center; gap: 8px; }
.stats-row { margin-bottom: 16px; padding: 12px; background: #f5f7fa; border-radius: 8px; }
.text-danger { color: #F56C6C; font-weight: 600; }
.text-warning { color: #E6A23C; font-weight: 600; }
</style>
