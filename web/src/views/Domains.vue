<template>
  <div class="domains">
    <el-card>
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <el-input
              v-model="keyword"
              placeholder="搜索域名..."
              prefix-icon="Search"
              clearable
              style="width: 240px; margin-right: 12px"
              @keyup.enter="fetchDomains"
              @clear="fetchDomains"
            />
            <el-select v-model="statusFilter" placeholder="状态筛选" clearable style="width: 150px" @change="fetchDomains">
              <el-option label="正常" value="active" />
              <el-option label="已过期" value="expired" />
              <el-option label="30天内到期" value="expiring_30" />
              <el-option label="已备案" value="icp_registered" />
              <el-option label="未备案/无法备案" value="icp_not_registered" />
            </el-select>
          </div>
          <div class="header-right">
            <el-dropdown trigger="click" style="margin-right: 8px">
              <el-button size="small">
                <el-icon><Setting /></el-icon>列控制
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item v-for="col in allColumns" :key="col.key">
                    <el-checkbox v-model="col.visible" @change="saveColumnSettings">{{ col.label }}</el-checkbox>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            <el-button size="small" @click="handleExportDomains">
              <el-icon><Download /></el-icon>导出
            </el-button>
            <el-upload
              ref="domainUploadRef"
              :auto-upload="false"
              :show-file-list="false"
              accept=".csv"
              :on-change="handleImportDomainsFile"
            >
              <el-button size="small">
                <el-icon><Upload /></el-icon>导入
              </el-button>
            </el-upload>
            <el-button type="primary" @click="showDialog()">
              <el-icon><Plus /></el-icon>添加域名
            </el-button>
          </div>
        </div>
      </template>

      <!-- Batch Action Bar -->
      <div class="batch-bar" v-if="selectedIds.length > 0">
        <span class="batch-count">已选 {{ selectedIds.length }} 个域名</span>
        <el-button type="warning" size="small" :loading="batchLoading" @click="handleBatchRefresh">
          <el-icon><Refresh /></el-icon>批量刷新
        </el-button>
        <el-button type="primary" size="small" :loading="batchLoading" @click="handleBatchPrice">
          <el-icon><Money /></el-icon>批量查价
        </el-button>
        <el-button type="success" size="small" :loading="batchLoading" @click="handleBatchToggle('auto_update', true)">
          <el-icon><Check /></el-icon>自动更新开
        </el-button>
        <el-button type="info" size="small" :loading="batchLoading" @click="handleBatchToggle('auto_update', false)">
          <el-icon><Close /></el-icon>自动更新关
        </el-button>
        <el-button type="primary" size="small" :loading="batchLoading" @click="handleBatchToggle('update_icp', true)">
          <el-icon><Check /></el-icon>ICP更新开
        </el-button>
        <el-button type="info" size="small" :loading="batchLoading" @click="handleBatchToggle('update_icp', false)">
          <el-icon><Close /></el-icon>ICP更新关
        </el-button>
        <el-popconfirm title="确定删除选中的域名？此操作不可恢复。" @confirm="handleBatchDelete">
          <template #reference>
            <el-button type="danger" size="small" :loading="batchLoading">
              <el-icon><Delete /></el-icon>批量删除
            </el-button>
          </template>
        </el-popconfirm>
      </div>

      <el-table
        ref="tableRef"
        :data="domains"
        style="width: 100%"
        v-loading="loading"
        empty-text="暂无域名数据"
        size="small"
        @selection-change="handleSelectionChange"
        @sort-change="handleSortChange"
      >
        <el-table-column type="selection" width="45" />

        <el-table-column prop="name" label="域名" min-width="180" fixed v-if="isColVisible('name')">
          <template #default="{ row }">
            <span style="font-weight:600;cursor:pointer;color:#409EFF;text-decoration:underline" @click="showDetail(row)">{{ row.name }}</span>
          </template>
        </el-table-column>

        <el-table-column label="域名天数" width="90" align="center" v-if="isColVisible('days')" sortable="custom" :sort-orders="['ascending', 'descending']" prop="__days__">
          <template #default="{ row }">
            <span v-if="row.expiry_date" :style="{ color: getDaysColor(row.expiry_date) }">
              {{ calcDays(row.expiry_date) }}天
            </span>
            <span v-else>-</span>
          </template>
        </el-table-column>

        <el-table-column label="证书数量" width="90" align="center" v-if="isColVisible('cert')">
          <template #default="{ row }">
            <el-tag v-if="row.cert_count > 0" type="success" size="small">{{ row.cert_count }}</el-tag>
            <span v-else>0</span>
          </template>
        </el-table-column>

        <el-table-column label="到期时间" width="110" v-if="isColVisible('expiry')" sortable="custom" :sort-orders="['ascending', 'descending']" prop="expiry_date">
          <template #default="{ row }">
            <span :class="{ 'expiring': isExpiringSoon(row.expiry_date) }">
              {{ row.expiry_date ? row.expiry_date.split('T')[0] : '-' }}
            </span>
          </template>
        </el-table-column>

        <el-table-column label="分组" width="100" show-overflow-tooltip v-if="isColVisible('group')">
          <template #default="{ row }">
            <el-tag v-if="row.group" size="small" type="info">{{ row.group }}</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>

        <el-table-column label="标签" width="120" show-overflow-tooltip v-if="isColVisible('tags')">
          <template #default="{ row }">
            <template v-if="row.tags">
              <el-tag v-for="t in row.tags.split(',')" :key="t" size="small" style="margin:1px" type="warning">
                {{ t.trim() }}
              </el-tag>
            </template>
            <span v-else>-</span>
          </template>
        </el-table-column>

        <el-table-column label="备注" width="120" show-overflow-tooltip v-if="isColVisible('note')">
          <template #default="{ row }">{{ row.note || '-' }}</template>
        </el-table-column>

        <el-table-column label="主办单位" width="140" show-overflow-tooltip v-if="isColVisible('org')">
          <template #default="{ row }">{{ row.registrant_org || '-' }}</template>
        </el-table-column>

        <el-table-column label="ICP备案" width="120" show-overflow-tooltip v-if="isColVisible('icp')">
          <template #default="{ row }">
            <span v-if="row.icp_number" style="color:#67C23A">{{ row.icp_number }}</span>
            <span v-else-if="row.icp_status === 'failed'" style="color:#F56C6C">无法备案</span>
            <span v-else-if="row.icp_status === 'not_found'" style="color:#F56C6C">未备案</span>
            <span v-else style="color:#C0C4CC">-</span>
          </template>
        </el-table-column>

        <el-table-column label="更新ICP" width="80" align="center" v-if="isColVisible('update_icp')">
          <template #default="{ row }">
            <el-switch v-model="row.update_icp" size="small" @change="toggleField(row, 'update_icp', row.update_icp)" />
          </template>
        </el-table-column>

        <el-table-column label="更新时间" width="110" v-if="isColVisible('updated_at')">
          <template #default="{ row }">
            <span v-if="row.whois_updated_at" style="font-size:12px;color:#909399">
              {{ formatTime(row.whois_updated_at) }}
            </span>
            <span v-else style="color:#C0C4CC">未更新</span>
          </template>
        </el-table-column>

        <el-table-column label="自动更新" width="80" align="center" v-if="isColVisible('auto_update')">
          <template #default="{ row }">
            <el-switch v-model="row.auto_update" size="small" @change="toggleField(row, 'auto_update', row.auto_update)" />
          </template>
        </el-table-column>

        <el-table-column label="到期提醒" width="80" align="center" v-if="isColVisible('expiry_reminder')">
          <template #default="{ row }">
            <el-switch v-model="row.expiry_reminder" size="small" @change="toggleField(row, 'expiry_reminder', row.expiry_reminder)" />
          </template>
        </el-table-column>

        <el-table-column label="续费价格" width="100" align="center" v-if="isColVisible('price')" sortable="custom" :sort-orders="['ascending', 'descending']" prop="renewal_price">
          <template #default="{ row }">
            <span v-if="row.renewal_price > 0" :style="{ color: row.price_source === 'fallback' ? '#409EFF' : '#E6A23C', fontWeight: 600 }">¥{{ row.renewal_price.toFixed(2) }}</span>
            <el-button v-else text type="primary" size="small" :loading="row._pricing" @click="handleQueryPrice(row)">查询</el-button>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button text type="warning" size="small" :loading="row._refreshing" @click="handleRefresh(row)">
              <el-icon><Refresh /></el-icon>刷新
            </el-button>
            <el-button text type="primary" size="small" @click="showDialog(row)">编辑</el-button>
            <el-button text type="primary" size="small" @click="compareDomain(row.name)">比价</el-button>
            <el-popconfirm title="确定删除此域名？" @confirm="handleDelete(row.id)">
              <template #reference>
                <el-button text type="danger" size="small">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination" v-if="total > 0">
        <el-pagination
          v-model:current-page="page"
          :page-size="20"
          :total="total"
          layout="total, prev, pager, next"
          @current-change="fetchDomains"
        />
      </div>
    </el-card>

    <!-- Add/Edit Dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑域名' : '添加域名'"
      width="560px"
      destroy-on-close
    >
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="90px">
        <el-form-item label="域名" prop="name">
          <el-input v-model="form.name" placeholder="例如: example.com" :disabled="isEdit" />
        </el-form-item>
        <el-form-item label="注册商">
          <el-input v-model="form.registrar" placeholder="例如: Namecheap" />
        </el-form-item>
        <el-form-item label="到期时间">
          <el-date-picker
            v-model="form.expiry_date"
            type="date"
            placeholder="选择到期日期"
            format="YYYY-MM-DD"
            value-format="YYYY-MM-DD"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="分组">
          <el-input v-model="form.group" placeholder="可选分组" />
        </el-form-item>
        <el-form-item label="标签">
          <el-input v-model="form.tags" placeholder="多个标签用逗号分隔" />
        </el-form-item>
        <el-form-item label="证书数量">
          <el-input-number v-model="form.cert_count" :min="0" />
        </el-form-item>
        <el-form-item label="NS服务器">
          <el-input v-model="form.nameservers" placeholder="多个用逗号分隔" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="自动续费">
          <el-switch v-model="form.auto_renew" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.note" type="textarea" :rows="2" placeholder="可选备注" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitLoading" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>

    <!-- Domain Detail Dialog -->
    <el-dialog v-model="detailVisible" :title="detailDomain?.name + ' - 详细信息'" width="800px" destroy-on-close>
      <div v-loading="detailLoading">
        <el-tabs v-model="detailTab">
          <el-tab-pane label="基本信息" name="basic">
            <el-descriptions :column="2" border size="small">
              <el-descriptions-item label="域名">{{ detailDomain?.name }}</el-descriptions-item>
              <el-descriptions-item label="注册商">{{ detailDomain?.registrar || '-' }}</el-descriptions-item>
              <el-descriptions-item label="状态">
                <el-tag :type="domainStatusInfo(detailDomain).type" size="small">
                  {{ domainStatusInfo(detailDomain).text }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="到期时间">{{ detailDomain?.expiry_date ? detailDomain.expiry_date.split('T')[0] : '-' }}</el-descriptions-item>
              <el-descriptions-item label="注册时间">{{ detailDomain?.registration_date ? detailDomain.registration_date.split('T')[0] : '-' }}</el-descriptions-item>
              <el-descriptions-item label="创建时间">{{ detailDomain?.creation_date ? detailDomain.creation_date.split('T')[0] : '-' }}</el-descriptions-item>
              <el-descriptions-item label="更新时间">{{ detailDomain?.updated_date ? detailDomain.updated_date.split('T')[0] : '-' }}</el-descriptions-item>
              <el-descriptions-item label="NS服务器" :span="2">{{ detailDomain?.nameservers || '-' }}</el-descriptions-item>
              <el-descriptions-item label="分组">{{ detailDomain?.group || '-' }}</el-descriptions-item>
              <el-descriptions-item label="标签">
                <template v-if="detailDomain?.tags">
                  <el-tag v-for="t in detailDomain.tags.split(',')" :key="t" size="small" style="margin:1px" type="warning">{{ t.trim() }}</el-tag>
                </template>
                <span v-else>-</span>
              </el-descriptions-item>
              <el-descriptions-item label="备注" :span="2">{{ detailDomain?.note || '-' }}</el-descriptions-item>
              <el-descriptions-item label="自动续费">{{ detailDomain?.auto_renew ? '是' : '否' }}</el-descriptions-item>
              <el-descriptions-item label="续费价格">
                <span v-if="detailDomain?.renewal_price > 0" :style="{ color: detailDomain?.price_source === 'fallback' ? '#409EFF' : '#E6A23C', fontWeight: 600 }">
                  ¥{{ detailDomain?.renewal_price?.toFixed(2) }}
                  <el-tag v-if="detailDomain?.price_source === 'fallback'" size="small" type="info" style="margin-left:4px">参考</el-tag>
                </span>
                <span v-else>-</span>
              </el-descriptions-item>
            </el-descriptions>
          </el-tab-pane>

          <el-tab-pane label="WHOIS 信息" name="whois">
            <template v-if="detailDomain?.whois_updated_at || detailDomain?.registrar_whois || detailDomain?.whois_status || detailDomain?.whois_raw">
              <el-descriptions :column="2" border size="small">
                <el-descriptions-item label="注册人">{{ detailDomain?.registrant_name || '-' }}</el-descriptions-item>
                <el-descriptions-item label="组织">{{ detailDomain?.registrant_org || '-' }}</el-descriptions-item>
                <el-descriptions-item label="邮箱">{{ detailDomain?.registrant_email || '-' }}</el-descriptions-item>
                <el-descriptions-item label="电话">{{ detailDomain?.registrant_phone || '-' }}</el-descriptions-item>
                <el-descriptions-item label="国家">{{ detailDomain?.registrant_country || '-' }}</el-descriptions-item>
                <el-descriptions-item label="DNSSEC">{{ detailDomain?.dnssec || '-' }}</el-descriptions-item>
                <el-descriptions-item label="注册商WHOIS">{{ detailDomain?.registrar_whois || '-' }}</el-descriptions-item>
                <el-descriptions-item label="WHOIS服务器">{{ detailDomain?.whois_server || '-' }}</el-descriptions-item>
                <el-descriptions-item label="WHOIS状态" :span="2">
                  <div v-if="detailDomain?.whois_status" style="line-height:1.8">
                    <div v-for="s in parseWhoisStatus(detailDomain.whois_status)" :key="s.code" style="margin-bottom:2px">
                      <el-tag :type="getWhoisStatusType(s.code)" size="small" style="margin-right:6px">{{ s.code }}</el-tag>
                      <span style="font-size:12px;color:#606266">{{ s.desc }}</span>
                    </div>
                  </div>
                  <span v-else>-</span>
                </el-descriptions-item>
                <el-descriptions-item label="更新时间">{{ detailDomain?.whois_updated_at ? new Date(detailDomain.whois_updated_at).toLocaleString('zh-CN') : '未更新' }}</el-descriptions-item>
              </el-descriptions>
            </template>
            <template v-else>
              <el-empty description="暂无 WHOIS 信息，请点击刷新按钮获取">
                <el-button type="primary" :loading="detailRefreshing" @click="refreshDetailDomain">
                  <el-icon><Refresh /></el-icon>刷新 WHOIS
                </el-button>
              </el-empty>
            </template>
            <div v-if="detailDomain?.whois_raw" style="margin-top:12px">
              <h4 style="margin-bottom:8px">WHOIS 原始数据</h4>
              <pre class="raw-data">{{ detailDomain.whois_raw }}</pre>
            </div>
          </el-tab-pane>

          <el-tab-pane label="ICP 备案" name="icp">
            <template v-if="detailDomain?.icp_number">
              <el-descriptions :column="2" border size="small">
                <el-descriptions-item label="备案号">{{ detailDomain?.icp_number }}</el-descriptions-item>
                <el-descriptions-item label="主办者">{{ detailDomain?.icp_owner_name || '-' }}</el-descriptions-item>
                <el-descriptions-item label="主办者类型">{{ detailDomain?.icp_owner_type || '-' }}</el-descriptions-item>
                <el-descriptions-item label="审核状态">{{ detailDomain?.icp_verify_status || '-' }}</el-descriptions-item>
                <el-descriptions-item label="备案日期">{{ detailDomain?.icp_filing_date ? detailDomain.icp_filing_date.split('T')[0] : '-' }}</el-descriptions-item>
                <el-descriptions-item label="服务名称">{{ detailDomain?.icp_service_name || '-' }}</el-descriptions-item>
                <el-descriptions-item label="服务URL" :span="2">{{ detailDomain?.icp_service_url || '-' }}</el-descriptions-item>
              </el-descriptions>
            </template>
            <template v-else>
              <el-empty description="暂无 ICP 备案信息">
                <el-button v-if="detailDomain?.icp_status === 'not_found'" type="warning" disabled>未备案</el-button>
                <el-button v-else-if="detailDomain?.icp_status === 'failed'" type="danger" disabled>无法查询</el-button>
                <el-button v-else type="info" disabled>暂无数据</el-button>
              </el-empty>
            </template>
            <div v-if="detailDomain?.icp_status === 'not_found'" style="margin-top:8px">
              <el-text type="danger" size="small">该域名未进行 ICP 备案</el-text>
            </div>
            <div v-if="detailDomain?.icp_status === 'failed'" style="margin-top:8px">
              <el-text type="danger" size="small">ICP 备案查询失败</el-text>
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  getDomains, getDomain, createDomain, updateDomain, deleteDomain,
  refreshDomainInfo, queryRenewalPrice, batchQueryRenewalPrice,
  batchDeleteDomains, batchUpdateDomains,
  exportDomains, importDomainsCSV,
} from '../api/domain'

const router = useRouter()
const tableRef = ref(null)
const domainUploadRef = ref(null)
const domains = ref([])
const loading = ref(false)
const keyword = ref('')
const statusFilter = ref('')
const page = ref(1)
const total = ref(0)

const selectedIds = ref([])
const batchLoading = ref(false)

const sortBy = ref('created_at')
const sortOrder = ref('DESC')

const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(null)
const formRef = ref(null)
const submitLoading = ref(false)

// Detail dialog
const detailVisible = ref(false)
const detailDomain = ref(null)
const detailTab = ref('basic')
const detailLoading = ref(false)
const detailRefreshing = ref(false)

const form = reactive({
  name: '',
  registrar: '',
  expiry_date: '',
  nameservers: '',
  auto_renew: false,
  note: '',
  group: '',
  tags: '',
  cert_count: 0,
})

const formRules = {
  name: [{ required: true, message: '请输入域名', trigger: 'blur' }],
}

// --- Column Visibility ---
const allColumns = reactive([
  { key: 'name', label: '域名', visible: true },
  { key: 'days', label: '域名天数', visible: true },
  { key: 'cert', label: '证书数量', visible: true },
  { key: 'expiry', label: '到期时间', visible: true },
  { key: 'group', label: '分组', visible: true },
  { key: 'tags', label: '标签', visible: true },
  { key: 'note', label: '备注', visible: true },
  { key: 'org', label: '主办单位', visible: true },
  { key: 'icp', label: 'ICP备案', visible: true },
  { key: 'update_icp', label: '更新ICP', visible: true },
  { key: 'updated_at', label: '更新时间', visible: true },
  { key: 'auto_update', label: '自动更新', visible: true },
  { key: 'expiry_reminder', label: '到期提醒', visible: true },
  { key: 'price', label: '续费价格', visible: true },
])

function isColVisible(key) {
  const col = allColumns.find(c => c.key === key)
  return col ? col.visible : true
}

function saveColumnSettings() {
  const settings = {}
  allColumns.forEach(c => { settings[c.key] = c.visible })
  localStorage.setItem('domainColumns', JSON.stringify(settings))
}

function loadColumnSettings() {
  try {
    const saved = JSON.parse(localStorage.getItem('domainColumns'))
    if (saved) {
      allColumns.forEach(c => {
        if (saved[c.key] !== undefined) c.visible = saved[c.key]
      })
    }
  } catch {}
}

// --- Helpers ---
function isExpiringSoon(dateStr) {
  if (!dateStr) return false
  const d = new Date(dateStr)
  const now = new Date()
  const diff = (d - now) / (1000 * 60 * 60 * 24)
  return diff > 0 && diff < 30
}

function calcDays(dateStr) {
  if (!dateStr) return 0
  const d = new Date(dateStr)
  const now = new Date()
  return Math.ceil((d - now) / (1000 * 60 * 60 * 24))
}

function getDaysColor(dateStr) {
  const days = calcDays(dateStr)
  if (days < 0) return '#F56C6C'
  if (days < 30) return '#E6A23C'
  if (days < 90) return '#409EFF'
  return '#67C23A'
}

function formatTime(t) {
  if (!t) return '-'
  const d = new Date(t)
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  return `${d.getFullYear()}-${mm}-${dd} ${hh}:${mi}`
}

function handleSelectionChange(rows) {
  selectedIds.value = rows.map(r => r.id)
}

function handleSortChange({ prop, order }) {
  if (!prop || !order) {
    sortBy.value = 'created_at'
    sortOrder.value = 'DESC'
  } else if (prop === '__days__') {
    sortBy.value = 'expiry_date'
    sortOrder.value = order === 'ascending' ? 'ASC' : 'DESC'
  } else {
    sortBy.value = prop
    sortOrder.value = order === 'ascending' ? 'ASC' : 'DESC'
  }
  page.value = 1
  fetchDomains()
}

function showDialog(row) {
  if (row) {
    isEdit.value = true
    editId.value = row.id
    form.name = row.name
    form.registrar = row.registrar || ''
    form.expiry_date = row.expiry_date ? row.expiry_date.split('T')[0] : ''
    form.nameservers = row.nameservers || ''
    form.auto_renew = row.auto_renew
    form.note = row.note || ''
    form.group = row.group || ''
    form.tags = row.tags || ''
    form.cert_count = row.cert_count || 0
  } else {
    isEdit.value = false
    editId.value = null
    form.name = ''
    form.registrar = ''
    form.expiry_date = ''
    form.nameservers = ''
    form.auto_renew = false
    form.note = ''
    form.group = ''
    form.tags = ''
    form.cert_count = 0
  }
  dialogVisible.value = true
}

async function handleRefresh(row) {
  row._refreshing = true
  try {
    const res = await refreshDomainInfo(row.id)
    const d = res.domain || res
    Object.assign(row, d)
    ElMessage.success(`${row.name} 刷新完成`)
  } catch (e) {
    ElMessage.error(`${row.name} 刷新失败: ${e.message || '未知错误'}`)
  } finally {
    row._refreshing = false
  }
}

async function handleQueryPrice(row) {
  row._pricing = true
  try {
    const res = await queryRenewalPrice(row.id)
    if (res.price > 0) {
      row.renewal_price = res.price
      row.price_source = res.source || ''
      ElMessage.success(`${row.name} 续费价格: ¥${res.price.toFixed(2)}${res.source === 'fallback' ? ' (参考价)' : ''}`)
    } else {
      ElMessage.warning(`${row.name} 无法获取续费价格: ${res.error || '未知原因'}`)
    }
  } catch (e) {
    ElMessage.error(`${row.name} 查询失败: ${e.message || '未知错误'}`)
  } finally {
    row._pricing = false
  }
}

async function fetchDomains() {
  loading.value = true
  try {
    const params = { page: page.value, sort_by: sortBy.value, sort_order: sortOrder.value }
    if (keyword.value) params.keyword = keyword.value
    if (statusFilter.value) params.status = statusFilter.value
    const res = await getDomains(params)
    domains.value = (res.data || []).map(d => ({ ...d, _refreshing: false, _pricing: false }))
    total.value = res.total || 0
  } finally {
    loading.value = false
  }
}

async function handleSubmit() {
  await formRef.value.validate()
  submitLoading.value = true
  try {
    if (isEdit.value) {
      await updateDomain(editId.value, form)
      ElMessage.success('更新成功')
    } else {
      await createDomain(form)
      ElMessage.success('添加成功')
    }
    dialogVisible.value = false
    fetchDomains()
  } finally {
    submitLoading.value = false
  }
}

async function handleDelete(id) {
  await deleteDomain(id)
  ElMessage.success('删除成功')
  fetchDomains()
}

async function toggleField(row, field, value) {
  try {
    await updateDomain(row.id, { [field]: value })
  } catch {
    row[field] = !value
  }
}

function compareDomain(name) {
  router.push({ path: '/price', query: { domain: name } })
}

const digitalPlatSuffixes = ['dpdns.org', 'us.kg', 'qzz.io', 'xx.kg', 'qd.je']

function isDigitalPlatDomain(name) {
  const n = String(name || '').trim().toLowerCase()
  return digitalPlatSuffixes.some(s => n === s || n.endsWith('.' + s))
}

async function showDetail(row) {
  detailVisible.value = true
  detailLoading.value = true
  detailTab.value = 'basic'
  detailDomain.value = { ...row }
  try {
    const fresh = await getDomain(row.id)
    detailDomain.value = { ...fresh }
  } catch (e) {
    ElMessage.warning('获取最新详情失败，显示列表数据: ' + (e.message || '未知错误'))
    detailDomain.value = { ...row }
  } finally {
    detailLoading.value = false
  }
  // DigitalPlat 域名优先使用官方 RDAP 服务器获取准确 WHOIS 数据（仅首次）
  if (isDigitalPlatDomain(detailDomain.value?.name) && !detailDomain.value?.whois_updated_at) {
    refreshDetailDomain()
  }
}

const domainStatusMap = {
  'active': { type: 'success', text: '正常' },
  'ok': { type: 'success', text: '正常' },
  'expired': { type: 'danger', text: '已过期' },
  'inactive': { type: 'warning', text: '未激活' },
  'pending': { type: 'warning', text: '处理中' },
  'hold': { type: 'warning', text: '暂停' },
  'clienthold': { type: 'warning', text: '注册商暂停' },
  'serverhold': { type: 'warning', text: '注册局暂停' },
  'redemptionperiod': { type: 'danger', text: '赎回期' },
  'pendingdelete': { type: 'danger', text: '待删除' },
  'pendingtransfer': { type: 'warning', text: '转移中' },
  'pendingrenew': { type: 'warning', text: '待续费' },
  'unknown': { type: 'info', text: '未知' },
}

function domainStatusInfo(d) {
  if (!d) return { type: 'info', text: '-' }
  const status = String(d.status || '').trim().toLowerCase()
  if (domainStatusMap[status]) return domainStatusMap[status]

  // 状态字段未知时，优先依据官方 RDAP/WHOIS 状态
  const whoisStatus = String(d.whois_status || '').trim().toLowerCase()
  if (whoisStatus) {
    if (whoisStatus.includes('active') || whoisStatus.split(/[,;]/)[0].trim() === 'ok') return { type: 'success', text: '正常' }
    if (whoisStatus.includes('expired') || whoisStatus.includes('redemption') || whoisStatus.includes('pendingdelete')) return { type: 'danger', text: '已过期' }
    if (whoisStatus.includes('hold')) return { type: 'warning', text: '暂停' }
    const first = whoisStatus.split(/[,;]/)[0].trim()
    if (first) return { type: 'info', text: first }
  }

  // 其次依据到期时间判断
  if (d.expiry_date) {
    const exp = new Date(d.expiry_date)
    if (!isNaN(exp.getTime())) {
      return exp.getTime() < Date.now() ? { type: 'danger', text: '已过期' } : { type: 'success', text: '正常' }
    }
  }

  if (status) return { type: 'info', text: status }
  return { type: 'info', text: '未知' }
}

const whoisStatusMap = {
  'clientTransferProhibited': '客户端禁止转移（注册商锁定，防止未经授权的域名转移）',
  'clientUpdateProhibited': '客户端禁止更新（锁定域名信息，防止被修改）',
  'clientDeleteProhibited': '客户端禁止删除（防止域名被误删或恶意删除）',
  'serverTransferProhibited': '服务端禁止转移（注册局锁定，最高级别保护）',
  'serverUpdateProhibited': '服务端禁止更新（注册局级别锁定）',
  'serverDeleteProhibited': '服务端禁止删除（注册局级别防删除）',
  'renewPeriod': '续费宽限期（域名已过期，处于续费恢复期）',
  'redemptionPeriod': '赎回期（域名过期后进入赎回阶段，需高价恢复）',
  'pendingDelete': '待删除（域名即将被释放删除）',
  'pendingTransfer': '待转移（域名正在转入过程中）',
  'pendingRenew': '待续费（域名续费处理中）',
  'pendingRestore': '待恢复（域名恢复处理中）',
  'ok': '正常状态（域名可正常使用）',
  'inactive': '未激活（域名未启用DNS解析）',
  'hold': '暂停（域名被暂停解析）',
  'serverHold': '注册局暂停（域名被注册局暂停，通常因实名认证等原因）',
  'clientHold': '注册商暂停（域名被注册商暂停解析）',
  'transferPeriod': '转移中（域名正在转移注册商）',
}

function parseWhoisStatus(statusStr) {
  if (!statusStr) return []
  // WHOIS status can be comma-separated or space-separated
  const parts = statusStr.split(/[,;]/).map(s => s.trim()).filter(Boolean)
  return parts.map(s => {
    let code = s.replace(/^Status:\s*/i, '').trim()
    // Normalize whitespace: "client transfer prohibited" -> "clientTransferProhibited"
    const normalized = code.replace(/\s+(.)/g, (_, c) => c.toUpperCase()).replace(/^\w/, c => c.toLowerCase())
    let desc = whoisStatusMap[normalized] || whoisStatusMap[code]
    if (!desc) {
      desc = `状态: ${code}`
    }
    return { code: normalized || code, desc }
  })
}

function getWhoisStatusType(code) {
  if (code.includes('Prohibited') || code.includes('Hold')) return 'danger'
  if (code.includes('pending') || code.includes('renew') || code.includes('redemption') || code.includes('restore')) return 'warning'
  if (code === 'ok') return 'success'
  return 'info'
}

async function refreshDetailDomain() {
  if (!detailDomain.value) return
  detailRefreshing.value = true
  try {
    const res = await refreshDomainInfo(detailDomain.value.id)
    const d = res.domain || res
    detailDomain.value = { ...detailDomain.value, ...d }
    ElMessage.success('WHOIS 信息已刷新')
  } catch (e) {
    ElMessage.error('刷新失败: ' + (e.message || '未知错误'))
  } finally {
    detailRefreshing.value = false
  }
}

// --- Export / Import ---
async function handleExportDomains() {
  try {
    const res = await exportDomains()
    const blob = new Blob([res], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'domains_export.csv'
    a.click()
    URL.revokeObjectURL(url)
    ElMessage.success('导出成功')
  } catch (e) {
    ElMessage.error('导出失败: ' + (e.message || '未知错误'))
  }
}

async function handleImportDomainsFile(file) {
  const formData = new FormData()
  formData.append('file', file.raw)
  try {
    const res = await importDomainsCSV(formData)
    ElMessage.success(res.message || `导入完成: 新增 ${res.imported}, 更新 ${res.updated}, 跳过 ${res.skipped}`)
    fetchDomains()
  } catch (e) {
    ElMessage.error('导入失败: ' + (e.response?.data?.error || e.message || '未知错误'))
  }
}

// --- Batch Operations ---
async function handleBatchDelete() {
  if (selectedIds.value.length === 0) return
  batchLoading.value = true
  try {
    await batchDeleteDomains(selectedIds.value)
    ElMessage.success(`成功删除 ${selectedIds.value.length} 个域名`)
    selectedIds.value = []
    tableRef.value?.clearSelection()
    fetchDomains()
  } catch (e) {
    ElMessage.error(`批量删除失败: ${e.message || '未知错误'}`)
  } finally {
    batchLoading.value = false
  }
}

async function handleBatchRefresh() {
  if (selectedIds.value.length === 0) return
  batchLoading.value = true
  let success = 0, fail = 0
  for (const id of selectedIds.value) {
    try { await refreshDomainInfo(id); success++ } catch { fail++ }
  }
  batchLoading.value = false
  ElMessage.success(`批量刷新完成: 成功 ${success}, 失败 ${fail}`)
  fetchDomains()
}

async function handleBatchPrice() {
  if (selectedIds.value.length === 0) return
  batchLoading.value = true
  try {
    const res = await batchQueryRenewalPrice(selectedIds.value)
    const data = res.data || []
    const success = data.filter(d => d.price > 0).length
    const fail = data.length - success
    ElMessage.success(`批量查价完成: 成功 ${success}, 失败 ${fail}`)
  } catch (e) {
    ElMessage.error(`批量查价失败: ${e.message || '未知错误'}`)
  } finally {
    batchLoading.value = false
    fetchDomains()
  }
}

async function handleBatchToggle(field, value) {
  if (selectedIds.value.length === 0) return
  batchLoading.value = true
  try {
    await batchUpdateDomains(selectedIds.value, { [field]: value })
    ElMessage.success(`已更新 ${selectedIds.value.length} 个域名`)
    fetchDomains()
  } catch (e) {
    ElMessage.error(`批量更新失败: ${e.message || '未知错误'}`)
  } finally {
    batchLoading.value = false
  }
}

onMounted(() => {
  loadColumnSettings()
  fetchDomains()
})
</script>

<style scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header-left {
  display: flex;
  align-items: center;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.batch-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  margin-bottom: 12px;
  background: #fdf6ec;
  border: 1px solid #e6a23c;
  border-radius: 6px;
}

.batch-count {
  font-weight: 600;
  color: #e6a23c;
  margin-right: 8px;
  white-space: nowrap;
}

.pagination {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.expiring {
  color: #E6A23C;
  font-weight: 600;
}

.raw-data {
  background: #f5f7fa;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  padding: 12px;
  font-family: monospace;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 300px;
  overflow-y: auto;
  line-height: 1.5;
}
</style>
