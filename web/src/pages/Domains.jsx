import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Table, Input, Select, Button, Popover, Checkbox, Space, Tag, Switch,
  Modal, Form, DatePicker, InputNumber, Tabs, Descriptions, Upload,
  Popconfirm, Empty, Tooltip,
} from 'antd'
import {
  SearchOutlined, SettingOutlined, DownloadOutlined, UploadOutlined,
  PlusOutlined, MoneyCollectOutlined, CheckOutlined,
  CloseOutlined, DeleteOutlined, ReloadOutlined, LinkOutlined, EditOutlined,
} from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import dayjs from 'dayjs'
import * as api from '../api/domain'
import { notify } from '../utils/toast'
import { useIsMobile } from '../utils/useIsMobile'
import { downloadBlob } from '../utils/download'
import { useContainerWidth } from '../utils/useContainerWidth'
import { fmtDate, fmtUpdated, calcDays, daysColor, daysLabel } from '../utils/format'
import PageHead from '../components/PageHead'
import StatusTag from '../components/StatusTag'

const FALLBACK_SUFFIXES = ['dpdns.org', 'us.kg', 'qzz.io', 'xx.kg', 'qd.je']

// Single source of truth for column widths (keep in sync with the builder below).
const COL_WIDTHS = {
  days: 80, cert: 70, expiry: 115, group: 90, tags: 130, note: 120,
  org: 140, icp: 130, updateIcp: 80, updatedAt: 130, autoUpdate: 84,
  expiryReminder: 84, price: 110,
}
const ACTIONS_WIDTH = { desktop: 200, mobile: 148 }

const COLUMN_DEFS = [
  { key: 'name', label: '域名', default: true },
  { key: 'days', label: '域名天数', default: true },
  { key: 'cert', label: '证书数量', default: true },
  { key: 'expiry', label: '到期时间', default: true },
  { key: 'group', label: '分组', default: true },
  { key: 'tags', label: '标签', default: true },
  { key: 'note', label: '备注', default: false },
  { key: 'org', label: '主办单位', default: false },
  { key: 'icp', label: 'ICP备案', default: true },
  { key: 'update_icp', label: '更新ICP', default: false },
  { key: 'updated_at', label: '更新时间', default: false },
  { key: 'auto_update', label: '自动更新', default: false },
  { key: 'expiry_reminder', label: '到期提醒', default: true },
  { key: 'price', label: '续费价格', default: true },
]

function loadColumnPrefs() {
  try {
    const saved = JSON.parse(localStorage.getItem('dm_domain_columns') || '{}')
    return COLUMN_DEFS.reduce((acc, c) => {
      acc[c.key] = saved[c.key] !== undefined ? saved[c.key] : c.default
      return acc
    }, {})
  } catch {
    return {}
  }
}

export default function Domains() {
  const navigate = useNavigate()
  const isMobile = useIsMobile()
  const [tableWrapRef, wrapWidth] = useContainerWidth()
  // The domain column adapts to the available space; narrower on mobile.
  const domainWidth = isMobile
    ? Math.max(110, Math.min(170, Math.round((wrapWidth || 360) * 0.32)))
    : Math.max(170, Math.min(480, Math.round((wrapWidth || 1200) * 0.26)))
  const actionsWidth = isMobile ? ACTIONS_WIDTH.mobile : ACTIONS_WIDTH.desktop

  const [loading, setLoading] = useState(false)
  const [domains, setDomains] = useState([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [sortBy, setSortBy] = useState('created_at')
  const [sortOrder, setSortOrder] = useState('DESC')
  const [selectedRowKeys, setSelectedRowKeys] = useState([])
  const [batchLoading, setBatchLoading] = useState(false)
  const [visible, setVisible] = useState(() => loadColumnPrefs())

  // detail modal
  const [detail, setDetail] = useState(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [detailTab, setDetailTab] = useState('basic')
  const [refreshingDetail, setRefreshingDetail] = useState(false)
  const [suffixes, setSuffixes] = useState(FALLBACK_SUFFIXES)

  // edit dialog
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)

  const isDigitalPlat = useCallback(
    (name) => {
      const n = String(name || '').trim().toLowerCase()
      return suffixes.some((s) => n === s || n.endsWith('.' + s))
    },
    [suffixes]
  )

  const fetchDomains = useCallback(
    async (overrides = {}) => {
      setLoading(true)
      try {
        const q = {
          page: overrides.page ?? page,
          page_size: 20,
          sort_by: overrides.sortBy ?? sortBy,
          sort_order: overrides.sortOrder ?? sortOrder,
        }
        const kw = overrides.keyword !== undefined ? overrides.keyword : keyword
        const st = overrides.status !== undefined ? overrides.status : statusFilter
        if (kw) q.keyword = kw
        if (st) q.status = st
        const res = await api.getDomains(q)
        setDomains(res.data || [])
        setTotal(res.total || 0)
      } finally {
        setLoading(false)
      }
    },
    [page, sortBy, sortOrder, keyword, statusFilter]
  )

  useEffect(() => {
    fetchDomains()
  }, [fetchDomains])

  useEffect(() => {
    api.getDigitalPlatSuffixes().then((res) => {
      if (Array.isArray(res.suffixes) && res.suffixes.length) setSuffixes(res.suffixes)
    }).catch(() => {})
  }, [])

  // ---- actions ----
  const handleRefresh = async (row) => {
    try {
      const res = await api.refreshDomainInfo(row.id)
      const d = res.domain || res
      setDomains((list) => list.map((x) => (x.id === row.id ? { ...x, ...d } : x)))
      notify('success', `${row.name} 刷新完成`)
    } catch {
      /* interceptor */
    }
  }

  const handleQueryPrice = async (row) => {
    try {
      const res = await api.queryRenewalPrice(row.id)
      if (res.price > 0) {
        setDomains((list) =>
          list.map((x) => (x.id === row.id ? { ...x, renewal_price: res.price, price_source: res.source || '' } : x))
        )
        notify('success', `${row.name} 续费价 ¥${res.price.toFixed(2)}${res.source === 'fallback' ? '（参考价）' : ''}`)
      } else {
        notify('warning', `${row.name}：${res.error || '无法获取续费价格'}`)
      }
    } catch {
      /* interceptor */
    }
  }

  const toggleField = async (row, field) => {
    const prev = row[field]
    setDomains((list) => list.map((x) => (x.id === row.id ? { ...x, [field]: !prev } : x)))
    try {
      await api.updateDomain(row.id, { [field]: !prev })
    } catch {
      setDomains((list) => list.map((x) => (x.id === row.id ? { ...x, [field]: prev } : x)))
    }
  }

  const handleDelete = async (id) => {
    await api.deleteDomain(id)
    notify('success', '域名已删除')
    fetchDomains()
  }

  // ---- dialog ----
  const openDialog = (row) => {
    setEditing(row || null)
    if (row) {
      form.setFieldsValue({
        name: row.name,
        registrar: row.registrar || '',
        expiry_date: row.expiry_date ? dayjs(row.expiry_date) : null,
        group: row.group || '',
        tags: row.tags || '',
        cert_count: row.cert_count || 0,
        nameservers: row.nameservers || '',
        auto_renew: !!row.auto_renew,
        note: row.note || '',
      })
    } else {
      form.resetFields()
    }
    setDialogOpen(true)
  }

  const handleSubmit = async () => {
    const values = await form.validateFields()
    setSubmitLoading(true)
    try {
      const payload = {
        name: values.name,
        registrar: values.registrar || '',
        expiry_date: values.expiry_date ? values.expiry_date.format('YYYY-MM-DD') : '',
        group: values.group || '',
        tags: values.tags || '',
        cert_count: values.cert_count || 0,
        nameservers: values.nameservers || '',
        auto_renew: !!values.auto_renew,
        note: values.note || '',
      }
      if (editing) {
        await api.updateDomain(editing.id, payload)
        notify('success', '更新成功')
      } else {
        await api.createDomain(payload)
        notify('success', '添加成功')
      }
      setDialogOpen(false)
      fetchDomains()
    } catch {
      /* interceptor */
    } finally {
      setSubmitLoading(false)
    }
  }

  // ---- detail modal ----
  const openDetail = async (row) => {
    setDetail({ ...row })
    setDetailTab('basic')
    setDetailLoading(true)
    try {
      const fresh = await api.getDomain(row.id)
      setDetail(fresh)
    } catch {
      setDetail({ ...row })
    } finally {
      setDetailLoading(false)
    }
  }

  const refreshDetail = async () => {
    if (!detail) return
    setRefreshingDetail(true)
    try {
      const res = await api.refreshDomainInfo(detail.id)
      const d = res.domain || res
      setDetail((prev) => ({ ...prev, ...d }))
      notify('success', 'WHOIS / ICP 已刷新')
    } catch {
      /* interceptor */
    } finally {
      setRefreshingDetail(false)
    }
  }

  // ---- export / import ----
  const handleExport = async () => {
    try {
      const blob = await api.exportDomains()
      downloadBlob(blob, 'domains_export.csv')
      notify('success', '导出成功')
    } catch {
      /* interceptor */
    }
  }

  const handleImportFile = async (file) => {
    const fd = new FormData()
    fd.append('file', file)
    try {
      const res = await api.importDomainsCSV(fd)
      notify('success', res.message || `导入完成：新增 ${res.imported}，更新 ${res.updated}，跳过 ${res.skipped}`)
      fetchDomains()
    } catch {
      /* interceptor */
    }
  }

  // ---- batch ----
  const handleBatchRefresh = async () => {
    setBatchLoading(true)
    let ok = 0
    let fail = 0
    for (const id of selectedRowKeys) {
      try {
        await api.refreshDomainInfo(id)
        ok++
      } catch {
        fail++
      }
    }
    setBatchLoading(false)
    notify('success', `批量刷新完成：成功 ${ok}，失败 ${fail}`)
    fetchDomains()
  }

  const handleBatchPrice = async () => {
    setBatchLoading(true)
    try {
      const res = await api.batchQueryRenewalPrice(selectedRowKeys)
      const data = res.data || []
      const ok = data.filter((d) => d.price > 0).length
      notify('success', `批量查价完成：成功 ${ok}，失败 ${data.length - ok}`)
    } catch {
      /* interceptor */
    } finally {
      setBatchLoading(false)
      fetchDomains()
    }
  }

  const handleBatchToggle = async (field, value) => {
    setBatchLoading(true)
    try {
      await api.batchUpdateDomains(selectedRowKeys, { [field]: value })
      notify('success', `已更新 ${selectedRowKeys.length} 个域名`)
    } catch {
      /* interceptor */
    } finally {
      setBatchLoading(false)
      fetchDomains()
    }
  }

  const handleBatchDelete = async () => {
    setBatchLoading(true)
    try {
      await api.batchDeleteDomains(selectedRowKeys)
      notify('success', `已删除 ${selectedRowKeys.length} 个域名`)
      setSelectedRowKeys([])
    } catch {
      /* interceptor */
    } finally {
      setBatchLoading(false)
      fetchDomains()
    }
  }

  const onTableChange = (pagination, _filters, sorter) => {
    setPage(pagination.current || 1)
    if (sorter && sorter.field) {
      setPage(1)
      setSortBy(sorter.field)
      setSortOrder(sorter.order === 'ascend' ? 'ASC' : 'DESC')
    } else {
      setSortBy('created_at')
      setSortOrder('DESC')
    }
  }

  const savePrefs = (next) => {
    setVisible(next)
    localStorage.setItem('dm_domain_columns', JSON.stringify(next))
  }

  const col = (key, fallback = true) => (visible[key] !== undefined ? visible[key] : fallback)

  // Sum of the columns that are actually rendered, so the table's scroll area
  // hugs the content (no trailing gap from a hard-coded x, and the panel never
  // overflows the viewport).
  const visibleColumnSum = useMemo(() => {
    let s = 0
    if (col('name')) s += domainWidth
    if (col('days')) s += COL_WIDTHS.days
    if (col('expiry')) s += COL_WIDTHS.expiry
    if (col('icp')) s += COL_WIDTHS.icp
    if (col('price')) s += COL_WIDTHS.price
    if (!isMobile) {
      if (col('cert')) s += COL_WIDTHS.cert
      if (col('group')) s += COL_WIDTHS.group
      if (col('tags')) s += COL_WIDTHS.tags
      if (col('note')) s += COL_WIDTHS.note
      if (col('org')) s += COL_WIDTHS.org
      if (col('update_icp')) s += COL_WIDTHS.updateIcp
      if (col('updated_at')) s += COL_WIDTHS.updatedAt
      if (col('auto_update')) s += COL_WIDTHS.autoUpdate
      if (col('expiry_reminder')) s += COL_WIDTHS.expiryReminder
    }
    s += actionsWidth
    return s
  }, [visible, domainWidth, isMobile, actionsWidth])
  const tableScrollX = Math.max(wrapWidth || 1200, visibleColumnSum)

  const columns = useMemo(() => {
    const cols = []
    if (col('name')) {
      cols.push({
        title: '域名',
        dataIndex: 'name',
        key: 'name',
        fixed: 'left',
        width: domainWidth,
        className: 'tbl-first',
        ellipsis: true,
        sorter: true,
        render: (v, r) => <span className="domain-link" onClick={() => openDetail(r)}>{v}</span>,
      })
    }
    if (col('days')) {
      cols.push({
        title: '剩余',
        dataIndex: 'expiry_date',
        key: 'expiry_date',
        width: COL_WIDTHS.days,
        sorter: true,
        render: (v) => {
          const days = calcDays(v)
          return <span style={{ color: daysColor(days), fontWeight: 500 }}>{daysLabel(days)}</span>
        },
      })
    }
    if (col('cert')) {
      cols.push({ title: '证书', dataIndex: 'cert_count', key: 'cert_count', width: COL_WIDTHS.cert, align: 'center', responsive: ['md'], render: (v) => v || 0 })
    }
    if (col('expiry')) {
      cols.push({
        title: '到期时间',
        dataIndex: 'expiry_date',
        key: 'expiry_date2',
        width: COL_WIDTHS.expiry,
        render: (v) => (v ? <span style={{ color: daysColor(calcDays(v)) }}>{fmtDate(v)}</span> : <span className="faint">-</span>),
      })
    }
    if (col('group')) {
      cols.push({
        title: '分组',
        dataIndex: 'group',
        key: 'group',
        width: COL_WIDTHS.group,
        render: (v) => (v ? <Tag style={{ borderRadius: 4 }}>{v}</Tag> : <span className="faint">-</span>),
        responsive: ['md'],
      })
    }
    if (col('tags')) {
      cols.push({
        title: '标签',
        dataIndex: 'tags',
        key: 'tags',
        width: COL_WIDTHS.tags,
        render: (v) =>
          v ? (
            <div className="tag-list">
              {String(v).split(',').filter(Boolean).map((t) => (
                <Tag key={t} style={{ borderRadius: 4 }} color="default">{t.trim()}</Tag>
              ))}
            </div>
          ) : (
            <span className="faint">-</span>
          ),
      })
    }
    if (col('note')) {
      cols.push({ title: '备注', dataIndex: 'note', key: 'note', width: COL_WIDTHS.note, ellipsis: true, responsive: ['md'], render: (v) => v || <span className="faint">-</span> })
    }
    if (col('org')) {
      cols.push({ title: '主办单位', dataIndex: 'registrant_org', key: 'org', width: COL_WIDTHS.org, ellipsis: true, responsive: ['md'], render: (v) => v || <span className="faint">-</span> })
    }
    if (col('icp')) {
      cols.push({
        title: 'ICP备案',
        dataIndex: 'icp_number',
        key: 'icp',
        width: COL_WIDTHS.icp,
        render: (v, r) => {
          if (v) return <span style={{ color: '#16a34a' }}>{v}</span>
          if (r.icp_status === 'failed') return <span style={{ color: '#dc2626' }}>无法备案</span>
          if (r.icp_status === 'not_found') return <span style={{ color: '#dc2626' }}>未备案</span>
          return <span className="faint">-</span>
        },
      })
    }
    if (col('update_icp')) {
      cols.push({
        title: '更新ICP',
        dataIndex: 'update_icp',
        key: 'update_icp',
        width: COL_WIDTHS.updateIcp,
        align: 'center',
        responsive: ['md'],
        render: (v, r) => <Switch size="small" checked={!!v} onChange={() => toggleField(r, 'update_icp')} />,
      })
    }
    if (col('updated_at')) {
      cols.push({ title: '更新时间', dataIndex: 'whois_updated_at', key: 'updated_at', width: COL_WIDTHS.updatedAt, responsive: ['md'], render: fmtUpdated })
    }
    if (col('auto_update')) {
      cols.push({
        title: '自动更新',
        dataIndex: 'auto_update',
        key: 'auto_update',
        width: COL_WIDTHS.autoUpdate,
        align: 'center',
        responsive: ['md'],
        render: (v, r) => <Switch size="small" checked={!!v} onChange={() => toggleField(r, 'auto_update')} />,
      })
    }
    if (col('expiry_reminder')) {
      cols.push({
        title: '到期提醒',
        dataIndex: 'expiry_reminder',
        key: 'expiry_reminder',
        width: COL_WIDTHS.expiryReminder,
        align: 'center',
        responsive: ['md'],
        render: (v, r) => <Switch size="small" checked={!!v} onChange={() => toggleField(r, 'expiry_reminder')} />,
      })
    }
    if (col('price')) {
      cols.push({
        title: '续费价',
        dataIndex: 'renewal_price',
        key: 'renewal_price',
        width: COL_WIDTHS.price,
        sorter: true,
        align: 'right',
        render: (v, r) =>
          v > 0 ? (
            <span className="price-num" style={{ color: r.price_source === 'fallback' ? '#2563eb' : '#d97706' }}>
              ¥{Number(v).toFixed(2)}
            </span>
          ) : (
            <Button size="small" type="text" icon={<MoneyCollectOutlined />} onClick={() => handleQueryPrice(r)}>
              查询
            </Button>
          ),
      })
    }
    cols.push({
      title: '操作',
      key: 'actions',
      fixed: 'right',
      width: actionsWidth,
      render: (_, r) => (
        <Space size={isMobile ? 2 : 0}>
          <Tooltip title="刷新">
            <Button type="text" size="small" icon={<ReloadOutlined />} onClick={() => handleRefresh(r)}>{!isMobile && '刷新'}</Button>
          </Tooltip>
          <Tooltip title="编辑">
            <Button type="text" size="small" icon={<EditOutlined />} onClick={() => openDialog(r)}>{!isMobile && '编辑'}</Button>
          </Tooltip>
          <Tooltip title="比价">
            <Button type="text" size="small" icon={<LinkOutlined />} onClick={() => navigate(`/price?domain=${encodeURIComponent(r.name)}`)}>{!isMobile && '比价'}</Button>
          </Tooltip>
          <Popconfirm title="确定删除此域名？" onConfirm={() => handleDelete(r.id)}>
            <Tooltip title="删除">
              <Button type="text" size="small" danger icon={<DeleteOutlined />}>{!isMobile && '删除'}</Button>
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    })
    return cols
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible, domains, suffixes, navigate, domainWidth, isMobile])

  const columnSettings = (
    <div style={{ width: 190 }}>
      {COLUMN_DEFS.map((c) => (
        <div key={c.key} style={{ padding: '4px 0' }}>
          <Checkbox
            checked={visible[c.key] !== undefined ? visible[c.key] : c.default}
            onChange={(e) => savePrefs({ ...visible, [c.key]: e.target.checked })}
          >
            <span style={{ fontSize: 13 }}>{c.label}</span>
          </Checkbox>
        </div>
      ))}
    </div>
  )

  return (
    <div className="page">
      <PageHead
        title="域名管理"
        sub={`共 ${total} 个域名 · 支持 WHOIS / ICP 刷新与批量操作`}
        actions={<>
          <Input
            allowClear
            prefix={<SearchOutlined style={{ color: '#a8a29e' }} />}
            placeholder="搜索域名"
            style={{ width: 220 }}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onPressEnter={() => setPage(1)}
          />
          <Select
            allowClear
            placeholder="状态筛选"
            style={{ width: 150 }}
            value={statusFilter || undefined}
            onChange={(v) => { setStatusFilter(v || ''); setPage(1) }}
            options={[
              { value: 'active', label: '正常' },
              { value: 'expired', label: '已过期' },
              { value: 'expiring_30', label: '30天内到期' },
              { value: 'icp_registered', label: '已备案' },
              { value: 'icp_not_registered', label: '未备案' },
            ]}
          />
          <Popover content={columnSettings} trigger="click" placement="bottomRight">
            <Button icon={<SettingOutlined />}>列控制</Button>
          </Popover>
          <Button icon={<DownloadOutlined />} onClick={handleExport}>导出</Button>
          <Upload accept=".csv" showUploadList={false} beforeUpload={(file) => { handleImportFile(file); return false }}>
            <Button icon={<UploadOutlined />}>导入</Button>
          </Upload>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => openDialog(null)}>添加域名</Button>
        </>}
      />

      {selectedRowKeys.length > 0 && (
        <div className="panel mb-16 batch-bar" style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '10px 14px', background: '#fefce8', borderColor: '#fde68a' }}>
          <span style={{ fontWeight: 600, fontSize: 13, marginRight: 8 }}>已选 {selectedRowKeys.length} 个域名</span>
          <Button size="small" loading={batchLoading} icon={<ReloadOutlined />} onClick={handleBatchRefresh}>批量刷新</Button>
          <Button size="small" loading={batchLoading} icon={<MoneyCollectOutlined />} onClick={handleBatchPrice}>批量查价</Button>
          <Button size="small" loading={batchLoading} icon={<CheckOutlined />} onClick={() => handleBatchToggle('auto_update', true)}>自动更新开</Button>
          <Button size="small" loading={batchLoading} icon={<CloseOutlined />} onClick={() => handleBatchToggle('auto_update', false)}>自动更新关</Button>
          <Button size="small" loading={batchLoading} icon={<CheckOutlined />} onClick={() => handleBatchToggle('update_icp', true)}>ICP更新开</Button>
          <Button size="small" loading={batchLoading} icon={<CloseOutlined />} onClick={() => handleBatchToggle('update_icp', false)}>ICP更新关</Button>
          <Popconfirm title={`确定删除选中的 ${selectedRowKeys.length} 个域名？此操作不可恢复。`} onConfirm={handleBatchDelete}>
            <Button size="small" danger loading={batchLoading} icon={<DeleteOutlined />}>批量删除</Button>
          </Popconfirm>
        </div>
      )}

      <div className="panel" ref={tableWrapRef}>
        <Table
          rowKey="id"
          columns={columns}
          dataSource={domains}
          loading={loading}
          size="middle"
          scroll={{ x: tableScrollX }}
          rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }}
          onChange={onTableChange}
          pagination={{
            current: page,
            pageSize: 20,
            total,
            showSizeChanger: false,
            showTotal: (t) => `共 ${t} 条`,
          }}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无域名数据" /> }}
        />
      </div>

      {/* Add / Edit */}
      <Modal
        title={editing ? '编辑域名' : '添加域名'}
        open={dialogOpen}
        onOk={handleSubmit}
        onCancel={() => setDialogOpen(false)}
        confirmLoading={submitLoading}
        width={560}
        destroyOnClose
      >
        <Form form={form} layout="vertical" requiredMark={false} style={{ marginTop: 16 }}>
          <Form.Item name="name" label="域名" rules={[{ required: true, message: '请输入域名' }]}>
            <Input placeholder="example.com" disabled={!!editing} />
          </Form.Item>
          <Form.Item name="registrar" label="注册商">
            <Input placeholder="例如：Namecheap" />
          </Form.Item>
          <Form.Item name="expiry_date" label="到期时间">
            <DatePicker style={{ width: '100%' }} placeholder="选择到期日期" />
          </Form.Item>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
            <Form.Item name="group" label="分组">
              <Input placeholder="可选分组" />
            </Form.Item>
            <Form.Item name="tags" label="标签">
              <Input placeholder="多个标签用逗号分隔" />
            </Form.Item>
          </div>
          <Form.Item name="cert_count" label="证书数量">
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="nameservers" label="NS 服务器">
            <Input.TextArea rows={2} placeholder="多个用逗号分隔" />
          </Form.Item>
          <Form.Item name="auto_renew" label="自动续费" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="note" label="备注">
            <Input.TextArea rows={2} placeholder="可选备注" />
          </Form.Item>
        </Form>
      </Modal>

      {/* Detail modal */}
      <Modal
        title={detail ? `${detail.name} — 详细信息` : ''}
        open={!!detail}
        onCancel={() => setDetail(null)}
        footer={null}
        width={isMobile ? '100%' : 920}
        destroyOnClose
        styles={{ body: { maxHeight: '72vh', overflowY: 'auto' } }}
      >
        {detail && (
          <>
            <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
              <Button size="small" icon={<ReloadOutlined />} loading={refreshingDetail} onClick={refreshDetail}>
                刷新 WHOIS / ICP
              </Button>
            </div>
            <Tabs activeKey={detailTab} onChange={setDetailTab} items={[
              {
                key: 'basic',
                label: '基本信息',
                children: (
                  <Descriptions column={isMobile ? 1 : 2} bordered size="small">
                    <Descriptions.Item label="域名">{detail.name}</Descriptions.Item>
                    <Descriptions.Item label="注册商">{detail.registrar || '-'}</Descriptions.Item>
                    <Descriptions.Item label="状态"><StatusTag data={detail} /></Descriptions.Item>
                    <Descriptions.Item label="到期时间">{fmtDate(detail.expiry_date)}</Descriptions.Item>
                    <Descriptions.Item label="注册时间">{fmtDate(detail.registration_date)}</Descriptions.Item>
                    <Descriptions.Item label="创建时间">{fmtDate(detail.creation_date)}</Descriptions.Item>
                    <Descriptions.Item label="更新时间">{fmtDate(detail.updated_date)}</Descriptions.Item>
                    <Descriptions.Item label="分组">{detail.group || '-'}</Descriptions.Item>
                    <Descriptions.Item label="NS 服务器" span={2}>{detail.nameservers || '-'}</Descriptions.Item>
                    <Descriptions.Item label="标签" span={2}>
                      {detail.tags ? <div className="tag-list">{String(detail.tags).split(',').filter(Boolean).map((t) => <Tag key={t}>{t.trim()}</Tag>)}</div> : '-'}
                    </Descriptions.Item>
                    <Descriptions.Item label="备注" span={2}>{detail.note || '-'}</Descriptions.Item>
                    <Descriptions.Item label="自动续费">{detail.auto_renew ? '是' : '否'}</Descriptions.Item>
                    <Descriptions.Item label="续费价格">
                      {detail.renewal_price > 0 ? (
                        <span className="price-num" style={{ color: detail.price_source === 'fallback' ? '#2563eb' : '#d97706' }}>
                          ¥{Number(detail.renewal_price).toFixed(2)}
                          {detail.price_source === 'fallback' && <Tag style={{ marginLeft: 6 }}>参考</Tag>}
                        </span>
                      ) : '-'}
                    </Descriptions.Item>
                  </Descriptions>
                ),
              },
              {
                key: 'whois',
                label: 'WHOIS',
                children: detail.whois_updated_at || detail.whois_raw ? (
                  <Descriptions column={isMobile ? 1 : 2} bordered size="small">
                    <Descriptions.Item label="注册人">{detail.registrant_name || '-'}</Descriptions.Item>
                    <Descriptions.Item label="组织">{detail.registrant_org || '-'}</Descriptions.Item>
                    <Descriptions.Item label="邮箱">{detail.registrant_email || '-'}</Descriptions.Item>
                    <Descriptions.Item label="电话">{detail.registrant_phone || '-'}</Descriptions.Item>
                    <Descriptions.Item label="国家">{detail.registrant_country || '-'}</Descriptions.Item>
                    <Descriptions.Item label="DNSSEC">{detail.dnssec || '-'}</Descriptions.Item>
                    <Descriptions.Item label="注册商(WHOIS)">{detail.registrar_whois || '-'}</Descriptions.Item>
                    <Descriptions.Item label="WHOIS服务器">{detail.whois_server || '-'}</Descriptions.Item>
                    <Descriptions.Item label="WHOIS状态" span={2}>{detail.whois_status || '-'}</Descriptions.Item>
                    <Descriptions.Item label="更新时间" span={2}>{fmtUpdated(detail.whois_updated_at)}</Descriptions.Item>
                  </Descriptions>
                ) : (
                  <Empty description="暂无 WHOIS 信息，点击右上角刷新获取" />
                ),
              },
              {
                key: 'icp',
                label: 'ICP 备案',
                children: (
                  <Descriptions column={isMobile ? 1 : 2} bordered size="small">
                    <Descriptions.Item label="备案号">{detail.icp_number || '-'}</Descriptions.Item>
                    <Descriptions.Item label="状态">
                      {detail.icp_status === 'registered' ? <Tag color="success">已备案</Tag>
                        : detail.icp_status === 'not_found' ? <Tag color="error">未备案</Tag>
                          : detail.icp_status === 'failed' ? <Tag color="error">查询失败</Tag> : <Tag>未知</Tag>}
                    </Descriptions.Item>
                    <Descriptions.Item label="主办单位">{detail.icp_owner_name || '-'}</Descriptions.Item>
                    <Descriptions.Item label="主体类型">{detail.icp_owner_type || '-'}</Descriptions.Item>
                    <Descriptions.Item label="服务名称">{detail.icp_service_name || '-'}</Descriptions.Item>
                    <Descriptions.Item label="服务域名">{detail.icp_service_url || '-'}</Descriptions.Item>
                    <Descriptions.Item label="备案日期">{fmtDate(detail.icp_filing_date)}</Descriptions.Item>
                    <Descriptions.Item label="核验状态">{detail.icp_verify_status || '-'}</Descriptions.Item>
                  </Descriptions>
                ),
              },
              {
                key: 'raw',
                label: '原始数据',
                children: <div className="raw-data">{detail.whois_raw || '暂无原始 WHOIS 数据'}</div>,
              },
            ]} />
          </>
        )}
      </Modal>
    </div>
  )
}
