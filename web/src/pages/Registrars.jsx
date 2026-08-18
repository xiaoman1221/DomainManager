import { useEffect, useMemo, useState } from 'react'
import { Table, Button, Modal, Form, Select, Input, Switch, Tag, Upload, Popconfirm, Empty, Alert } from 'antd'
import { PlusOutlined, DownloadOutlined, UploadOutlined, ImportOutlined, DeleteOutlined, CheckOutlined, CloseOutlined } from '@ant-design/icons'
import * as api from '../api/registrar'
import { notify } from '../utils/toast'
import PageHead from '../components/PageHead'
import { downloadBlob } from '../utils/download'
import { fmtDateTime } from '../utils/format'

// Per-type credential field labels using each registrar's official naming.
// key -> APIKey, secret -> APISecret, extra -> APIExtra (all optional).
const REGISTRAR_FIELDS = {
  aliyun: { key: 'AccessKey ID', secret: 'AccessKey Secret' },
  aliyun_intl: { key: 'AccessKey ID', secret: 'AccessKey Secret' },
  tencent: { key: 'SecretId', secret: 'SecretKey' },
  tencent_intl: { key: 'SecretId', secret: 'SecretKey' },
  huawei: { key: 'Access Key ID', secret: 'Secret Access Key' },
  cloudflare: { key: '账号邮箱 (Email)', secret: '全局 API 密钥 (Global API Key)' },
  godaddy: { key: 'API Key', secret: 'API Secret' },
  namecheap: { key: 'ApiUser（账户名）', secret: 'API Key', extra: 'UserName（需管理的账户，通常与 ApiUser 相同）' },
  namesilo: { key: 'API Key' },
  dynadot: { key: 'API Key' },
  digitalplat: { key: 'API Key' },
  porkbun: { key: 'API Key', secret: 'Secret Key' },
  amazon: { key: 'Access Key ID', secret: 'Secret Access Key' },
  other: {},
}

// One-line setup documentation shown in the edit dialog per registrar type.
const REGISTRAR_FIELD_DOCS = {
  aliyun: '在阿里云「RAM 访问控制 → 用户 → 创建 AccessKey」获取 AccessKey ID / AccessKey Secret。',
  aliyun_intl: '在阿里云国际版「RAM 访问控制」中创建 AccessKey，用于调用域名 API。',
  tencent: '在腾讯云「访问管理 CAM → API 密钥管理」中创建 SecretId / SecretKey。',
  tencent_intl: '在腾讯云国际版「访问管理」中创建 SecretId / SecretKey。',
  huawei: '在华为云「我的凭证 → 访问密钥」中创建 AK/SK（Access Key ID / Secret Access Key）。',
  cloudflare: '在 Cloudflare「我的个人资料 → API 令牌」中获取 Global API Key，配合账号邮箱使用。',
  godaddy: '在 GoDaddy 开发者门户 developer.godaddy.com 创建 Production 环境 API Key / API Secret。',
  namecheap: '在 Namecheap「账户设置 → API」中开启 API 并创建 API Key；ApiUser 填账户名，UserName 填需管理的账户（通常与 ApiUser 相同）。',
  namesilo: '在 NameSilo「Account → API Management」中创建 API Key。',
  dynadot: '在 Dynadot「My Account → API」中获取 API Key。',
  digitalplat: '在 DigitalPlat 控制台 dash.domain.digitalplat.org 的 API Keys 页面创建 API Key。',
  porkbun: '在 Porkbun「Account → API Access」中创建 API Key / Secret Key。',
  amazon: '在 AWS IAM 中创建访问密钥（Access Key ID / Secret Access Key），并授予 Route53 只读权限。',
  other: '该类型不支持自动导入，请在「导入域名」弹窗中手动输入域名列表。',
}

export default function Registrars() {
  const [registrars, setRegistrars] = useState([])
  const [loading, setLoading] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState(null)
  const [form] = Form.useForm()
  const selectedType = Form.useWatch('type', form)
  const [submitLoading, setSubmitLoading] = useState(false)
  const [importOpen, setImportOpen] = useState(false)
  const [importRegistrar, setImportRegistrar] = useState(null)
  const [importText, setImportText] = useState('')
  const [importLoading, setImportLoading] = useState(false)
  const [cnTypes, setCnTypes] = useState([])
  const [globalTypes, setGlobalTypes] = useState([])

  const allTypes = useMemo(() => [...cnTypes, ...globalTypes], [cnTypes, globalTypes])

  const fetchRegistrars = async () => {
    setLoading(true)
    try {
      const res = await api.getRegistrars()
      setRegistrars(res.data || [])
    } finally {
      setLoading(false)
    }
  }

  const fetchTypes = async () => {
    try {
      const res = await api.getRegistrarTypes()
      const all = res.data || []
      setCnTypes(all.filter((t) => t.region === 'cn'))
      setGlobalTypes(all.filter((t) => t.region === 'global' || t.region === 'other'))
    } catch {
      setCnTypes([{ value: 'aliyun', label: '阿里云（万网）' }, { value: 'tencent', label: '腾讯云（DNSPod）' }, { value: 'huawei', label: '华为云' }])
      setGlobalTypes([{ value: 'aliyun_intl', label: '阿里云（国际）' }, { value: 'tencent_intl', label: '腾讯云（国际）' }, { value: 'cloudflare', label: 'Cloudflare' }, { value: 'godaddy', label: 'GoDaddy' }, { value: 'namecheap', label: 'Namecheap' }, { value: 'namesilo', label: 'NameSilo' }, { value: 'dynadot', label: 'Dynadot' }, { value: 'digitalplat', label: 'DigitalPlat（免费域名）' }, { value: 'porkbun', label: 'Porkbun' }, { value: 'amazon', label: 'AWS Route53' }, { value: 'other', label: '其他（手动导入）' }])
    }
  }

  useEffect(() => {
    fetchRegistrars()
    fetchTypes()
  }, [])

  const typeLabel = (type) => allTypes.find((t) => t.value === type)?.label || type
  const isCN = (type) => cnTypes.some((t) => t.value === type)

  const openDialog = (row) => {
    setEditing(row || null)
    if (row) {
      form.setFieldsValue({
        name: row.name,
        type: row.type,
        api_endpoint: row.api_endpoint || '',
        api_key: row.api_key || '',
        api_secret: row.api_secret || '',
        api_extra: row.api_extra || '',
        enabled: row.enabled,
        sync_enabled: row.sync_enabled,
        use_proxy: row.use_proxy || false,
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
      if (editing) {
        await api.updateRegistrar(editing.id, values)
        notify('success', '更新成功')
      } else {
        await api.createRegistrar(values)
        notify('success', '添加成功')
      }
      setDialogOpen(false)
      fetchRegistrars()
    } catch {
      /* interceptor */
    } finally {
      setSubmitLoading(false)
    }
  }

  const handleDelete = async (id) => {
    await api.deleteRegistrar(id)
    notify('success', '注册商已删除')
    fetchRegistrars()
  }

  const toggleField = async (row, field) => {
    const prev = row[field]
    setRegistrars((list) => list.map((x) => (x.id === row.id ? { ...x, [field]: !prev } : x)))
    try {
      await api.updateRegistrar(row.id, { [field]: !prev })
    } catch {
      setRegistrars((list) => list.map((x) => (x.id === row.id ? { ...x, [field]: prev } : x)))
    }
  }

  const openImport = (row) => {
    setImportRegistrar(row)
    setImportText('')
    setImportOpen(true)
  }

  const handleImport = async () => {
    setImportLoading(true)
    try {
      const res = await api.importDomains({ registrar_id: importRegistrar.id, domains: importText.trim() })
      if (res.error) {
        notify('error', res.error)
      } else if (res.imported === 0 && res.skipped === 0 && (res.refreshed || 0) === 0) {
        notify('warning', '未导入任何域名。支持自动导入的注册商：阿里云、腾讯云、华为云、Cloudflare、GoDaddy、Namecheap、NameSilo、Dynadot、Porkbun、AWS Route53、DigitalPlat。其他类型请手动输入域名列表。')
      } else {
        notify('success', res.message || `导入完成：新增 ${res.imported}，刷新 ${res.refreshed || 0}，跳过 ${res.skipped}`)
      }
      setImportOpen(false)
    } catch {
      /* interceptor */
    } finally {
      setImportLoading(false)
    }
  }

  const handleExport = async () => {
    try {
      const blob = await api.exportRegistrars()
      downloadBlob(blob, 'registrars_export.csv')
      notify('success', '导出成功')
    } catch {
      /* interceptor */
    }
  }

  const handleImportCSV = async (file) => {
    const fd = new FormData()
    fd.append('file', file)
    try {
      const res = await api.importRegistrarsCSV(fd)
      notify('success', res.message || `导入完成：新建 ${res.created}，更新 ${res.updated}，跳过 ${res.skipped}`)
      fetchRegistrars()
    } catch {
      /* interceptor */
    }
  }

  const columns = [
    { title: '名称', dataIndex: 'name', key: 'name', width: 150, className: 'tbl-first', render: (v) => <span style={{ fontWeight: 500 }}>{v}</span> },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 170,
      render: (v) => (
        <Tag color={v.startsWith('aliyun') || v.startsWith('tencent') || v.startsWith('huawei') ? 'red' : 'blue'} style={{ borderRadius: 4 }}>
          {typeLabel(v)}
        </Tag>
      ),
    },
    { title: '地区', dataIndex: 'type', key: 'region', width: 70, align: 'center', responsive: ['md'], render: (v) => (isCN(v) ? <Tag style={{ borderRadius: 4 }}>国内</Tag> : <Tag style={{ borderRadius: 4 }}>国际</Tag>) },
    {
      title: 'API 配置',
      key: 'api',
      width: 110,
      responsive: ['md'],
      render: (_, r) => {
        const fields = REGISTRAR_FIELDS[r.type] || { key: 'API Key', secret: 'API Secret' }
        const configured = !!fields.key && !!r.api_key && (!fields.secret || !!r.api_secret)
        return configured ? (
          <span style={{ color: '#16a34a' }}><CheckOutlined /> 已配置</span>
        ) : (
          <span className="faint"><CloseOutlined /> 未配置</span>
        )
      },
    },
    { title: '自动同步', dataIndex: 'sync_enabled', key: 'sync_enabled', width: 90, align: 'center', responsive: ['md'], render: (v, r) => <Switch size="small" checked={!!v} onChange={() => toggleField(r, 'sync_enabled')} /> },
    { title: '启用', dataIndex: 'enabled', key: 'enabled', width: 70, align: 'center', render: (v, r) => <Switch size="small" checked={!!v} onChange={() => toggleField(r, 'enabled')} /> },
    { title: '最后同步', dataIndex: 'last_sync_at', key: 'last_sync_at', width: 160, responsive: ['md'], render: (v) => fmtDateTime(v) },
    {
      title: '操作',
      key: 'actions',
      fixed: 'right',
      width: 200,
      render: (_, r) => (
        <>
          <Button type="text" size="small" icon={<ImportOutlined />} onClick={() => openImport(r)}>导入域名</Button>
          <Button type="text" size="small" onClick={() => openDialog(r)}>编辑</Button>
          <Popconfirm title="确定删除此注册商？" onConfirm={() => handleDelete(r.id)}>
            <Button type="text" size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </>
      ),
    },
  ]

  return (
    <div className="page">
      <PageHead
        title="注册商管理"
        sub="维护注册商 API 凭据并从注册商批量导入域名"
        actions={<>
          <Button icon={<DownloadOutlined />} onClick={handleExport}>导出</Button>
          <Upload accept=".csv" showUploadList={false} beforeUpload={(file) => { handleImportCSV(file); return false }}>
            <Button icon={<UploadOutlined />}>导入</Button>
          </Upload>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => openDialog(null)}>添加注册商</Button>
        </>}
      />

      <div className="panel">
        <Table rowKey="id" columns={columns} dataSource={registrars} loading={loading} size="middle" scroll={{ x: 1000 }} locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无注册商配置" /> }} />
      </div>

      <Modal title={editing ? '编辑注册商' : '添加注册商'} open={dialogOpen} onOk={handleSubmit} onCancel={() => setDialogOpen(false)} confirmLoading={submitLoading} width={580} destroyOnClose>
        <Form form={form} layout="vertical" requiredMark={false} style={{ marginTop: 16 }}>
          <Form.Item name="type" label="注册商类型" rules={[{ required: true, message: '请选择类型' }]}>
            <Select placeholder="选择注册商类型" options={[
              { label: '国内', options: cnTypes },
              { label: '国际', options: globalTypes },
            ]} />
          </Form.Item>
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="自定义名称" />
          </Form.Item>
          <Form.Item name="api_endpoint" label="API 端点">
            <Input placeholder="API 地址（可选，使用默认）" />
          </Form.Item>
          <Alert type="info" showIcon style={{ marginBottom: 16 }} message={REGISTRAR_FIELD_DOCS[selectedType] || '填写该注册商 API 凭据，用于自动导入域名'} />
          {(() => {
            const fields = REGISTRAR_FIELDS[selectedType] || { key: 'API Key', secret: 'API Secret' }
            return (
              <>
                {fields.key && (
                  <Form.Item name="api_key" label={fields.key}>
                    <Input placeholder={fields.key} autoComplete="off" />
                  </Form.Item>
                )}
                {fields.secret && (
                  <Form.Item name="api_secret" label={fields.secret}>
                    <Input.Password placeholder={fields.secret} autoComplete="new-password" />
                  </Form.Item>
                )}
                {fields.extra && (
                  <Form.Item name="api_extra" label={fields.extra}>
                    <Input placeholder={fields.extra} autoComplete="off" />
                  </Form.Item>
                )}
              </>
            )
          })()}
          <div style={{ display: 'flex', gap: 32 }}>
            <Form.Item name="enabled" label="启用" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item name="sync_enabled" label="自动同步" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item name="use_proxy" label="使用代理" valuePropName="checked" extra="开启后该注册商的 API 请求走系统设置的 HTTP 代理（PROXY_URL）">
              <Switch />
            </Form.Item>
          </div>
        </Form>
      </Modal>

      <Modal title={`从「${importRegistrar?.name || ''}」导入域名`} open={importOpen} onOk={handleImport} onCancel={() => setImportOpen(false)} confirmLoading={importLoading} width={560} destroyOnClose>
        <Alert type="info" showIcon style={{ marginBottom: 16 }} message="填写域名列表或留空自动获取。" />
        <Input.TextArea
          rows={8}
          value={importText}
          onChange={(e) => setImportText(e.target.value)}
          placeholder={'每行一个域名，例如：\nexample.com\ntest.org\n\n留空则尝试从注册商 API 自动获取'}
        />
      </Modal>
    </div>
  )
}
