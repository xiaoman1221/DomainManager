import dayjs from 'dayjs'

export const fmtDate = (v) => (v ? dayjs(v).format('YYYY-MM-DD') : '-')
export const fmtDateTime = (v) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm') : '-')
export const fmtUpdated = (v) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm') : '未更新')

// Whole days remaining; negative when past expiry. null when unknown.
export function calcDays(dateStr) {
  if (!dateStr) return null
  return Math.ceil(dayjs(dateStr).diff(dayjs(), 'day', true))
}

export function daysColor(days) {
  if (days === null) return '#a8a29e'
  if (days < 0) return '#dc2626'
  if (days < 30) return '#d97706'
  if (days < 90) return '#2563eb'
  return '#16a34a'
}

export function daysLabel(days) {
  if (days === null) return '-'
  return `${days}天`
}

export const DOMAIN_STATUS_MAP = {
  active: { color: 'success', text: '正常' },
  ok: { color: 'success', text: '正常' },
  expired: { color: 'error', text: '已过期' },
  inactive: { color: 'warning', text: '未激活' },
  pending: { color: 'warning', text: '处理中' },
  hold: { color: 'warning', text: '暂停' },
  clienthold: { color: 'warning', text: '注册商暂停' },
  serverhold: { color: 'warning', text: '注册局暂停' },
  redemptionperiod: { color: 'error', text: '赎回期' },
  pendingdelete: { color: 'error', text: '待删除' },
  pendingtransfer: { color: 'warning', text: '转移中' },
  pendingrenew: { color: 'warning', text: '待续费' },
  unknown: { color: 'default', text: '未知' },
}

export function domainStatusInfo(d) {
  if (!d) return { color: 'default', text: '-' }
  const status = String(d.status || '').trim().toLowerCase()
  if (DOMAIN_STATUS_MAP[status]) return DOMAIN_STATUS_MAP[status]

  const whoisStatus = String(d.whois_status || '').trim().toLowerCase()
  if (whoisStatus) {
    if (whoisStatus.includes('active') || whoisStatus.split(/[,;]/)[0].trim() === 'ok') return { color: 'success', text: '正常' }
    if (whoisStatus.includes('expired') || whoisStatus.includes('redemption') || whoisStatus.includes('pendingdelete')) return { color: 'error', text: '已过期' }
    if (whoisStatus.includes('hold')) return { color: 'warning', text: '暂停' }
    const first = whoisStatus.split(/[,;]/)[0].trim()
    if (first) return { color: 'default', text: first }
  }

  if (d.expiry_date) {
    const days = calcDays(d.expiry_date)
    if (days !== null) return days < 0 ? { color: 'error', text: '已过期' } : { color: 'success', text: '正常' }
  }

  if (status) return { color: 'default', text: status }
  return { color: 'default', text: '未知' }
}
