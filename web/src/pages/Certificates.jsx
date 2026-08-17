import { useEffect, useState } from 'react'
import { Table, Button, Modal, Form, Input, Select, Tag, Descriptions, Popconfirm, Empty, Statistic, Alert } from 'antd'
import { PlusOutlined, SettingOutlined, SyncOutlined, DeleteOutlined, EyeOutlined } from '@ant-design/icons'
import * as api from '../api/certificate'
import { useAuth } from '../context/AuthContext'
import { notify } from '../utils/toast'
import PageHead from '../components/PageHead'
import { fmtDate, fmtDateTime, calcDays } from '../utils/format'
import { useIsMobile } from '../utils/useIsMobile'

export default function Certificates() {
  const { user } = useAuth()
  const isAdmin = user?.role_group === 'admin'
  const isMobile = useIsMobile()

  const [certs, setCerts] = useState([])
  const [stats, setStats] = useState(null)
  const [loading, setLoading] = useState(false)
  const [kwInput, setKwInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState('')

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)
  const [detail, setDetail] = useState(null)

  const [configOpen, setConfigOpen] = useState(false)
  const [configForm] = Form.useForm()
  const [savingConfig, setSavingConfig] = useState(false)
  const [syncing, setSyncing] = useState(false)

  const fetchCerts = async () => {
    setLoading(true)
    try {
      const res = await api.getCertificates({ keyword, status: statusFilter })
      setCerts(res.data || [])
    } finally {
      setLoading(false)
    }
  }

  const fetchStats = () => api.getCertificateStats().then(setStats).catch(() => {})

  useEffect(() => {
    fetchCerts()
    fetchStats()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [keyword, statusFilter])

  const openDialog = (row) => {
    setEditing(row || null)
    if (row) {
      form.setFieldsValue({
        domain: row.domain,
        issuer: row.issuer || '',
        key_algorithm: row.key_algorithm || '',
        not_before: row.not_before || '',
        not_after: row.not_after || '',
        subject_alt_names: row.subject_alt_names || '',
        source: row.source || 'certimate',
        note: row.note || '',
      })
    } else {
      form.resetFields()
      form.setFieldsValue({ source: 'certimate' })
    }
    setDialogOpen(true)
  }

  const handleSave = async () => {
    const values = await form.validateFields()
    setSubmitLoading(true)
    try {
      const payload = {
        ...values,
        not_before: values.not_before ? values.not_before.split('T')[0] : '',
        not_after: values.not_after ? values.not_after.split('T')[0] : '',
      }
      if (editing) {
        await api.updateCertificate(editing.id, payload)
        notify('success', '更新成功')
      } else {
        await api.createCertificate(payload)
        notify('success', '创建成功')
      }
      setDialogOpen(false)
      fetchCerts()
      fetchStats()
    } catch {
      /* interceptor */
    } finally {
      setSubmitLoading(false)
    }
  }

  const handleDelete = async (id) => {
    await api.deleteCertificate(id)
    notify('success', '证书已删除')
    fetchCerts()
    fetchStats()
  }

  const handleSync = async () => {
    setSyncing(true)
    try {
      const res = await api.syncCertimateCertificates()
      notify('success', `同步完成，共 ${res.total || 0} 张证书`)
      fetchCerts()
      fetchStats()
    } catch {
      /* interceptor */
    } finally {
      setSyncing(false)
    }
  }

  const handleSaveConfig = async () => {
    const values = await configForm.validateFields()
    setSavingConfig(true)
    try {
      await api.saveCertimateConfig(values)
      notify('success', '配置已保存')
      setConfigOpen(false)
    } catch {
      /* interceptor */
    } finally {
      setSavingConfig(false)
    }
  }

  const openConfig = async () => {
    try {
      const res = await api.getCertimateConfig()
      configForm.setFieldsValue({ url: res.url || '', username: res.username || '', password: '' })
    } catch {
      configForm.setFieldsValue({ url: '', username: '', password: '' })
    }
    setConfigOpen(true)
  }

  const expiryClass = (row) => {
    if (!row.not_after || row.status === 'expired') return { color: '#dc2626', fontWeight: 600 }
    const days = calcDays(row.not_after)
    if (days < 30) return { color: '#d97706', fontWeight: 600 }
    return undefined
  }

  const columns = [
    { title: '域名', dataIndex: 'domain', key: 'domain', width: 210, className: 'tbl-first', render: (v) => <span style={{ fontWeight: 500 }}>{v}</span> },
    { title: '颁发者', dataIndex: 'issuer', key: 'issuer', width: 160, responsive: ['md'], render: (v) => v || <span className="faint">-</span> },
    { title: '密钥算法', dataIndex: 'key_algorithm', key: 'key_algorithm', width: 110, responsive: ['md'], render: (v) => <Tag style={{ borderRadius: 4 }}>{v || '-'}</Tag> },
    { title: '生效时间', dataIndex: 'not_before', key: 'not_before', width: 115, responsive: ['md'], render: fmtDate },
    { title: '到期时间', dataIndex: 'not_after', key: 'not_after', width: 115, render: (v, r) => <span style={expiryClass(r)}>{fmtDate(v)}</span> },
    { title: '状态', dataIndex: 'status', key: 'status', width: 90, render: (v) => <Tag color={v === 'active' ? 'success' : 'error'} style={{ borderRadius: 4 }}>{v === 'active' ? '正常' : '已过期'}</Tag> },
    { title: '来源', dataIndex: 'source', key: 'source', width: 100, responsive: ['md'], render: (v) => <Tag color={v === 'certimate' ? 'blue' : 'default'} style={{ borderRadius: 4 }}>{v === 'certimate' ? 'Certimate' : v || '手动'}</Tag> },
    {
      title: '操作',
      key: 'actions',
      fixed: 'right',
      width: 170,
      render: (_, r) => (
        <>
          <Button type="text" size="small" onClick={() => openDialog(r)}>编辑</Button>
          <Button type="text" size="small" icon={<EyeOutlined />} onClick={() => setDetail(r)}>详情</Button>
          <Popconfirm title="确定删除此证书？" onConfirm={() => handleDelete(r.id)}>
            <Button type="text" size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </>
      ),
    },
  ]

  return (
    <div className="page">
      <PageHead
        title="证书管理"
        sub="跟踪 SSL 证书生命周期，支持与 Certimate 同步"
        actions={<>
          <Input.Search allowClear placeholder="搜索域名或颁发者" style={{ width: 240 }} value={kwInput} onChange={(e) => setKwInput(e.target.value)} onSearch={(v) => setKeyword(v)} />
          <Select
            allowClear placeholder="状态筛选" style={{ width: 140 }} value={statusFilter || undefined}
            onChange={(v) => setStatusFilter(v || '')}
            options={[
              { value: 'active', label: '正常' },
              { value: 'expired', label: '已过期' },
              { value: 'expiring_30', label: '30天内到期' },
            ]}
          />
          {isAdmin && (
            <>
              <Button icon={<SettingOutlined />} onClick={openConfig}>Certimate 配置</Button>
              <Button icon={<SyncOutlined />} loading={syncing} onClick={handleSync}>同步证书</Button>
            </>
          )}
          <Button type="primary" icon={<PlusOutlined />} onClick={() => openDialog(null)}>添加证书</Button>
        </>}
      />

      {stats && (
        <div className="panel mb-16">
          <div className="stats-grid-4">
            <div style={{ padding: '18px 24px', borderRight: '1px solid var(--border)' }}><Statistic title="证书总数" value={stats.total} /></div>
            <div style={{ padding: '18px 24px', borderRight: '1px solid var(--border)' }}><Statistic title="正常" value={stats.active} valueStyle={{ color: '#16a34a' }} /></div>
            <div style={{ padding: '18px 24px', borderRight: '1px solid var(--border)' }}><Statistic title="已过期" value={stats.expired} valueStyle={{ color: '#dc2626' }} /></div>
            <div style={{ padding: '18px 24px' }}><Statistic title="即将到期" value={stats.expiring_soon} valueStyle={{ color: '#d97706' }} /></div>
          </div>
        </div>
      )}

      <div className="panel">
        <Table rowKey="id" columns={columns} dataSource={certs} loading={loading} size="middle" scroll={{ x: 1100 }} pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 条` }} locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无证书" /> }} />
      </div>

      <Modal title={editing ? '编辑证书' : '添加证书'} open={dialogOpen} onOk={handleSave} onCancel={() => setDialogOpen(false)} confirmLoading={submitLoading} width={620} destroyOnClose>
        <Form form={form} layout="vertical" requiredMark={false} style={{ marginTop: 16 }}>
          <Form.Item name="domain" label="域名" rules={[{ required: true, message: '请输入域名' }]}>
            <Input placeholder="example.com" />
          </Form.Item>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
            <Form.Item name="issuer" label="颁发者"><Input placeholder="Let's Encrypt" /></Form.Item>
            <Form.Item name="key_algorithm" label="密钥算法">
              <Select options={[{ value: 'RSA 2048', label: 'RSA 2048' }, { value: 'RSA 4096', label: 'RSA 4096' }, { value: 'ECC P-256', label: 'ECC P-256' }, { value: 'ECC P-384', label: 'ECC P-384' }]} placeholder="选择算法" />
            </Form.Item>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
            <Form.Item name="not_before" label="生效时间"><Input placeholder="YYYY-MM-DD" /></Form.Item>
            <Form.Item name="not_after" label="到期时间"><Input placeholder="YYYY-MM-DD" /></Form.Item>
          </div>
          <Form.Item name="subject_alt_names" label="Subject Alt Names"><Input.TextArea rows={2} placeholder="example.com,www.example.com" /></Form.Item>
          <Form.Item name="source" label="来源">
            <Select options={[{ value: 'certimate', label: 'Certimate' }, { value: 'manual', label: '手动' }, { value: 'other', label: '其他' }]} />
          </Form.Item>
          <Form.Item name="note" label="备注"><Input.TextArea rows={2} /></Form.Item>
        </Form>
      </Modal>

      <Modal
        title={detail ? `${detail.domain} — 证书详情` : ''}
        open={!!detail}
        onCancel={() => setDetail(null)}
        footer={null}
        width={isMobile ? '100%' : 560}
        destroyOnClose
      >
        {detail && (
          <Descriptions column={isMobile ? 1 : 2} bordered size="small">
            <Descriptions.Item label="域名">{detail.domain}</Descriptions.Item>
            <Descriptions.Item label="颁发者">{detail.issuer || '-'}</Descriptions.Item>
            <Descriptions.Item label="序列号">{detail.serial_number || '-'}</Descriptions.Item>
            <Descriptions.Item label="生效时间">{fmtDateTime(detail.not_before)}</Descriptions.Item>
            <Descriptions.Item label="到期时间">{fmtDateTime(detail.not_after)}</Descriptions.Item>
            <Descriptions.Item label="Subject Alt Names">{detail.subject_alt_names || '-'}</Descriptions.Item>
            <Descriptions.Item label="密钥算法">{detail.key_algorithm || '-'}</Descriptions.Item>
            <Descriptions.Item label="签名算法">{detail.signature_algorithm || '-'}</Descriptions.Item>
            <Descriptions.Item label="状态"><Tag color={detail.status === 'active' ? 'success' : 'error'}>{detail.status === 'active' ? '正常' : '已过期'}</Tag></Descriptions.Item>
            <Descriptions.Item label="来源">{detail.source || '-'}</Descriptions.Item>
            <Descriptions.Item label="备注">{detail.note || '-'}</Descriptions.Item>
          </Descriptions>
        )}
      </Modal>

      <Modal title="Certimate 配置" open={configOpen} onOk={handleSaveConfig} onCancel={() => setConfigOpen(false)} confirmLoading={savingConfig} width={480} destroyOnClose>
        <Alert type="info" showIcon style={{ marginBottom: 16 }} message="填写 Certimate 后台登录账号密码，系统会自动换取 API Token（密码仅保存用于登录，不会回显）。" />
        <Form form={configForm} layout="vertical" requiredMark={false}>
          <Form.Item name="url" label="API 地址" rules={[{ required: true, message: '请输入 API 地址' }]}>
            <Input placeholder="http://127.0.0.1:8090" />
          </Form.Item>
          <Form.Item name="username" label="登录账号" rules={[{ required: true, message: '请输入登录账号' }]}>
            <Input placeholder="Certimate 后台登录账号" autoComplete="off" />
          </Form.Item>
          <Form.Item name="password" label="登录密码" rules={[{ required: true, message: '请输入登录密码' }]}>
            <Input.Password placeholder="Certimate 后台登录密码" autoComplete="new-password" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

