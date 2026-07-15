<template>
  <div class="price">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>域名比价</span>
        </div>
      </template>

      <el-form :inline="true" @submit.prevent="handleCompare">
        <el-form-item label="域名">
          <el-input
            v-model="domain"
            placeholder="请输入域名，如 example.com"
            style="width: 360px"
            clearable
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="handleCompare">
            <el-icon><Search /></el-icon>查询比价
          </el-button>
        </el-form-item>
      </el-form>

      <el-alert
        v-if="!loading && prices.length > 0"
        type="success"
        :closable="false"
        show-icon
        style="margin-bottom: 16px"
      >
        已找到 <strong>{{ prices.length }}</strong> 个注册商的报价
      </el-alert>

      <el-table :data="prices" style="width: 100%" v-loading="loading" empty-text="请输入域名进行比价">
        <el-table-column prop="registrar" label="注册商" width="160">
          <template #default="{ row }">
            <div class="registrar-cell">
              <span class="registrar-name">{{ row.registrar }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="tld" label="后缀" width="80" align="center">
          <template #default="{ row }">
            <el-tag size="small">.{{ row.tld }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="注册价" width="140" align="right">
          <template #default="{ row }">
            <span class="price-main">{{ row.currency === 'CNY' ? '¥' : '$' }}{{ row.register_price.toFixed(2) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="续费价" width="140" align="right">
          <template #default="{ row }">
            <span class="price-sub">{{ row.currency === 'CNY' ? '¥' : '$' }}{{ row.renew_price.toFixed(2) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="转入价" width="140" align="right">
          <template #default="{ row }">
            <span class="price-sub">{{ row.currency === 'CNY' ? '¥' : '$' }}{{ row.transfer_price.toFixed(2) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="最低注册价" width="140" align="right">
          <template #default="{ row }">
            <el-tag v-if="isCheapest(row)" type="success" size="small">最低价</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" link @click="goRegister(row.url)">
              前往购买
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card style="margin-top: 20px">
      <template #header>
        <span>支持的域名后缀</span>
      </template>
      <div class="tld-list">
        <el-tag
          v-for="tld in supportedTLDs"
          :key="tld"
          class="tld-tag"
          @click="fillTLD(tld)"
        >
          .{{ tld }}
        </el-tag>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { comparePrices, getSupportedTLDs } from '../api/price'

const route = useRoute()
const domain = ref('')
const prices = ref([])
const loading = ref(false)
const supportedTLDs = ref([])

function isCheapest(row) {
  if (prices.value.length <= 1) return false
  const min = Math.min(...prices.value.map((p) => p.register_price))
  return row.register_price === min
}

function fillTLD(tld) {
  domain.value = domain.value.split('.')[0] + '.' + tld
}

function goRegister(url) {
  if (url) {
    window.open(url, '_blank')
  }
}

async function handleCompare() {
  if (!domain.value.trim()) {
    ElMessage.warning('请输入域名')
    return
  }
  loading.value = true
  try {
    const res = await comparePrices({ domain: domain.value.trim() })
    prices.value = res.prices || []
    if (prices.value.length === 0) {
      ElMessage.info('暂无该域名的比价数据')
    }
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  try {
    const res = await getSupportedTLDs()
    supportedTLDs.value = res.tlds || []
  } catch {
    // handled
  }

  if (route.query.domain) {
    domain.value = route.query.domain
    handleCompare()
  }
})
</script>

<style scoped>
.card-header {
  font-weight: 600;
  font-size: 16px;
}

.registrar-cell {
  display: flex;
  align-items: center;
}

.registrar-name {
  font-weight: 500;
}

.price-main {
  font-size: 16px;
  font-weight: 700;
  color: #f56c6c;
}

.price-sub {
  font-size: 14px;
  color: #909399;
}

.tld-list {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.tld-tag {
  cursor: pointer;
  transition: transform 0.2s;
}

.tld-tag:hover {
  transform: scale(1.1);
}
</style>
