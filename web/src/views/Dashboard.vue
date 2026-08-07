<template>
  <div class="dashboard">
    <el-row :gutter="20" class="stats-row">
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: #ecf5ff">
            <el-icon :size="32" color="#409EFF"><Connection /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.total }}</div>
            <div class="stat-label">域名总数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: #f0f9eb">
            <el-icon :size="32" color="#67C23A"><CircleCheck /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.active }}</div>
            <div class="stat-label">正常域名</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: #fdf6ec">
            <el-icon :size="32" color="#E6A23C"><WarningFilled /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.expiring_soon }}</div>
            <div class="stat-label">即将到期</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px">
      <el-col :span="14">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>最近添加的域名</span>
              <el-button text type="primary" @click="$router.push('/domains')">查看全部</el-button>
            </div>
          </template>
          <el-table :data="recentDomains" style="width: 100%" empty-text="暂无域名">
            <el-table-column prop="name" label="域名" min-width="200" />
            <el-table-column prop="registrar" label="注册商" width="120" />
            <el-table-column label="到期时间" width="130">
              <template #default="{ row }">
                {{ row.expiry_date ? row.expiry_date.split('T')[0] : '-' }}
              </template>
            </el-table-column>
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="domainStatusInfo(row).type" size="small">
                  {{ domainStatusInfo(row).text }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="10">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>快捷操作</span>
            </div>
          </template>
          <div class="quick-actions">
            <el-button type="primary" size="large" @click="$router.push('/domains')">
              <el-icon><Plus /></el-icon>添加域名
            </el-button>
            <el-button type="success" size="large" @click="$router.push('/price')">
              <el-icon><Money /></el-icon>域名比价
            </el-button>
          </div>
          <el-divider />
          <div class="tips">
            <h4>使用提示</h4>
            <ul>
              <li>添加您的域名，集中管理所有域名</li>
              <li>使用比价功能找到最优惠的注册商</li>
              <li>设置到期提醒，避免域名过期</li>
            </ul>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getDomainStats, getDomains } from '../api/domain'

const stats = ref({ total: 0, active: 0, expiring_soon: 0 })
const recentDomains = ref([])

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

  const whoisStatus = String(d.whois_status || '').trim().toLowerCase()
  if (whoisStatus) {
    if (whoisStatus.includes('active') || whoisStatus.split(/[,;]/)[0].trim() === 'ok') return { type: 'success', text: '正常' }
    if (whoisStatus.includes('expired') || whoisStatus.includes('redemption') || whoisStatus.includes('pendingdelete')) return { type: 'danger', text: '已过期' }
    if (whoisStatus.includes('hold')) return { type: 'warning', text: '暂停' }
    const first = whoisStatus.split(/[,;]/)[0].trim()
    if (first) return { type: 'info', text: first }
  }

  if (d.expiry_date) {
    const exp = new Date(d.expiry_date)
    if (!isNaN(exp.getTime())) {
      return exp.getTime() < Date.now() ? { type: 'danger', text: '已过期' } : { type: 'success', text: '正常' }
    }
  }

  if (status) return { type: 'info', text: status }
  return { type: 'info', text: '未知' }
}

onMounted(async () => {
  try {
    const [statsRes, domainsRes] = await Promise.all([
      getDomainStats(),
      getDomains({ page: 1 }),
    ])
    stats.value = statsRes
    recentDomains.value = (domainsRes.data || []).slice(0, 5)
  } catch {
    // handled by interceptor
  }
})
</script>

<style scoped>
.stats-row {
  margin-bottom: 10px;
}

.stat-card {
  cursor: pointer;
}

.stat-card :deep(.el-card__body) {
  display: flex;
  align-items: center;
  gap: 20px;
}

.stat-icon {
  width: 64px;
  height: 64px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #303133;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-top: 4px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-weight: 600;
}

.quick-actions {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.quick-actions .el-button {
  width: 100%;
}

.tips h4 {
  margin-bottom: 10px;
  color: #606266;
}

.tips ul {
  padding-left: 18px;
  color: #909399;
  font-size: 14px;
  line-height: 2;
}
</style>
