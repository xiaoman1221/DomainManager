import { useEffect, useState } from 'react'
import { Table, Tabs, Button, Modal, Form, Input, InputNumber, Select, Switch, Tag, Popconfirm, Empty, Alert } from 'antd'
import { PlusOutlined, BellOutlined, DeleteOutlined, PlayCircleOutlined, ClockCircleOutlined } from '@ant-design/icons'
import * as api from '../api/notification'
import { notify } from '../utils/toast'
import PageHead from '../components/PageHead'
import { fmtDateTime } from '../utils/format'

const TYPE_META = {
  bark: { label: 'Bark', color: 'success' },
  telegram: { label: 'Telegram', color: 'blue' },
  email: { label: '邮件', color: 'warning' },
  webhook: { label: 'Webhook', color: 'default' },
}

export default function Notifications() {
  const [channels, setChannels] = useState([])
  const [logs, setLogs] = useState([])
  const [types, setTypes] = useState([])
  const [loading, setLoading] = useState(false)
  const [logsLoading, setLogsLoading] = useState(false)
  const [activeTab, setActiveTab] = useState('channels')

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)

  const [testOpen, setTestOpen] = useState(false)
  const [testChannel, setTestChannel] = useState(null)
  const [testForm] = Form.useForm()
  const [testing, setTesting] = useState(false)

  const [sendingExpiry, setSendingExpiry] = useState(false)

  const [schedules, setSchedules] = useState([])
  const [schedLoading, setSchedLoading] = useState(false)
  const [schedDialogOpen, setSchedDialogOpen] = useState(false)
  const [schedEditing, setSchedEditing] = useState(null)
  const [schedForm] = Form.useForm()
  const [schedSaving, setSchedSaving] = useState(false)
  const [runningId, setRunningId] = useState(null)

  // per-type config sub-forms
  const [bark, setBark] = useState({ server: '', key: '', group: '域名管理' })
  const [telegram, setTelegram] = useState({ bot_token: '', chat_id: '' })
  const [email, setEmail] = useState({ smtp_host: '', smtp_port: '587', username: '', password: '', from: '', to: '', use_tls: false })
  const [webhook, setWebhook] = useState({ url: '', method: 'POST' })

  const fetchChannels = async () => {
    setLoading(true)
    try {
      const res = await api.getNotificationChannels()
      setChannels(res.data || [])
    } finally {
      setLoading(false)
    }
  }

  const fetchLogs = async () => {
    setLogsLoading(true)
    try {
      const res = await api.getNotificationLogs()
      setLogs(res.data || [])
    } finally {
      setLogsLoading(false)
    }
  }

  const fetchSchedules = async () => {
    setSchedLoading(true)
    try {
      const res = await api.getScheduledTasks()
      setSchedules(res.data || [])
    } finally {
      setSchedLoading(false)
    }
  }

  useEffect(() => {
    fetchChannels()
    fetchLogs()
    fetchSchedules()
    api.getNotificationTypes().then((res) => setTypes(res.data || [])).catch(() => {})
  }, [])

  const buildConfig = (type) => {
    if (type === 'bark') return JSON.stringify(bark)
    if (type === 'telegram') return JSON.stringify(telegram)
    if (type === 'email') return JSON.stringify(email)
    if (type === 'webhook') return JSON.stringify(webhook)
    return ''
  }

  const parseConfig = (type, str) => {
    let obj = {}
    try {
      obj = JSON.parse(str || '{}')
    } catch {
      obj = {}
    }
    if (type === 'bark') setBark({ server: '', key: '', group: '域名管理', ...obj })
    if (type === 'telegram') setTelegram({ bot_token: '', chat_id: '', ...obj })
    if (type === 'email') setEmail({ smtp_host: '', smtp_port: '587', username: '', password: '', from: '', to: '', use_tls: false, ...obj })
    if (type === 'webhook') setWebhook({ url: '', method: 'POST', ...obj })
  }

  const openDialog = (row) => {
    setEditing(row || null)
    if (row) {
      form.setFieldsValue({ name: row.name, type: row.type })
      parseConfig(row.type, row.config)
    } else {
      form.resetFields()
      form.setFieldsValue({ type: 'bark' })
      setBark({ server: '', key: '', group: '域名管理' })
      setTelegram({ bot_token: '', chat_id: '' })
      setEmail({ smtp_host: '', smtp_port: '587', username: '', password: '', from: '', to: '', use_tls: false })
      setWebhook({ url: '', method: 'POST' })
    }
    setDialogOpen(true)
  }

  const handleSave = async () => {
    const values = await form.validateFields()
    setSubmitLoading(true)
    try {
      const payload = { ...values, config: buildConfig(values.type) }
      if (editing) {
        await api.updateNotificationChannel(editing.id, payload)
        notify('success', '更新成功')
      } else {
        await api.createNotificationChannel(payload)
        notify('success', '创建成功')
      }
      setDialogOpen(false)
      await fetchChannels()
    } catch {
      /* interceptor */
    } finally {
      setSubmitLoading(false)
    }
  }

  const handleDelete = async (id) => {
    await api.deleteNotificationChannel(id)
    notify('success', '渠道已删除')
    await fetchChannels()
  }

  const handleToggle = async (row) => {
    const prev = row.enabled
    setChannels((list) => list.map((x) => (x.id === row.id ? { ...x, enabled: !prev } : x)))
    try {
      await api.toggleNotificationChannel(row.id)
      notify('success', !prev ? '已启用' : '已禁用')
    } catch {
      setChannels((list) => list.map((x) => (x.id === row.id ? { ...x, enabled: prev } : x)))
    }
  }

  const openTest = (row) => {
    setTestChannel(row)
    testForm.setFieldsValue({ title: '测试通知', content: '这是一条来自 Domain Manager 的测试通知' })
    setTestOpen(true)
  }

  const handleTest = async () => {
    const values = await testForm.validateFields()
    setTesting(true)
    try {
      await api.testNotificationChannel(testChannel.id, values)
      notify('success', '测试发送成功')
      setTestOpen(false)
      await fetchLogs()
    } catch {
      /* interceptor */
    } finally {
      setTesting(false)
    }
  }

  const handleSendExpiry = async () => {
    setSendingExpiry(true)
    try {
      const res = await api.sendExpiryNotifications()
      notify('success', `到期提醒已发送，成功 ${res.sent || 0} 个渠道`)
      await fetchLogs()
    } catch {
      /* interceptor */
    } finally {
      setSendingExpiry(false)
    }
  }

  const openSchedDialog = (row) => {
    setSchedEditing(row || null)
    if (row) {
      schedForm.setFieldsValue({ name: row.name, interval_minutes: row.interval_minutes, enabled: row.enabled })
    } else {
      schedForm.setFieldsValue({ name: '系统信息定时推送', interval_minutes: 1440, enabled: true })
    }
    setSchedDialogOpen(true)
  }

  const handleSaveSched = async () => {
    const values = await schedForm.validateFields()
    setSchedSaving(true)
    try {
      const payload = { ...values, type: 'system_info', enabled: values.enabled !== false }
      if (schedEditing) {
        await api.updateScheduledTask(schedEditing.id, payload)
        notify('success', '更新成功')
      } else {
        await api.createScheduledTask(payload)
        notify('success', '创建成功')
      }
      setSchedDialogOpen(false)
      await fetchSchedules()
    } catch {
      /* interceptor */
    } finally {
      setSchedSaving(false)
    }
  }

  const handleDeleteSched = async (id) => {
    await api.deleteScheduledTask(id)
    notify('success', '定时任务已删除')
    await fetchSchedules()
  }

  const handleToggleSched = async (row) => {
    const prev = row.enabled
    setSchedules((list) => list.map((x) => (x.id === row.id ? { ...x, enabled: !prev } : x)))
    try {
      await api.updateScheduledTask(row.id, { enabled: !prev })
      notify('success', !prev ? '已启用' : '已禁用')
    } catch {
      setSchedules((list) => list.map((x) => (x.id === row.id ? { ...x, enabled: prev } : x)))
    }
  }

  const handleRunSched = async (id) => {
    setRunningId(id)
    try {
      await api.runScheduledTask(id)
      notify('success', '已立即执行')
      await fetchSchedules()
      await fetchLogs()
    } catch {
      /* interceptor */
    } finally {
      setRunningId(null)
    }
  }

  const fmtInterval = (m) => {
    if (!m || m <= 0) return '-'
    if (m % 1440 === 0) return `${m / 1440} 天`
    if (m % 60 === 0) return `${m / 60} 小时`
    return `${m} 分钟`
  }

  const getChannelName = (id) => channels.find((c) => c.id === id)?.name || '未知'

  const channelColumns = [
    { title: '名称', dataIndex: 'name', key: 'name', width: 180, className: 'tbl-first', render: (v) => <span style={{ fontWeight: 500 }}>{v}</span> },
    { title: '类型', dataIndex: 'type', key: 'type', width: 130, render: (v) => <Tag color={TYPE_META[v]?.color || 'default'} style={{ borderRadius: 4 }}>{TYPE_META[v]?.label || v}</Tag> },
    { title: '状态', dataIndex: 'enabled', key: 'enabled', width: 90, align: 'center', render: (v, r) => <Switch size="small" checked={!!v} onChange={() => handleToggle(r)} /> },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170, responsive: ['md'], render: fmtDateTime },
    {
      title: '操作',
      key: 'actions',
      width: 210,
      render: (_, r) => (
        <>
          <Button type="text" size="small" onClick={() => openTest(r)}>测试</Button>
          <Button type="text" size="small" onClick={() => openDialog(r)}>编辑</Button>
          <Popconfirm title="确定删除此通知渠道？" onConfirm={() => handleDelete(r.id)}>
            <Button type="text" size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </>
      ),
    },
  ]

  const logColumns = [
    { title: '渠道', dataIndex: 'channel_id', key: 'channel_id', width: 140, className: 'tbl-first', render: getChannelName },
    { title: '标题', dataIndex: 'title', key: 'title', minWidth: 160, ellipsis: true },
    { title: '内容', dataIndex: 'content', key: 'content', minWidth: 240, ellipsis: true, responsive: ['md'] },
    { title: '状态', dataIndex: 'status', key: 'status', width: 90, render: (v) => <Tag color={v === 'success' ? 'success' : 'error'} style={{ borderRadius: 4 }}>{v === 'success' ? '成功' : '失败'}</Tag> },
    { title: '错误信息', dataIndex: 'error', key: 'error', width: 200, ellipsis: true, responsive: ['md'], render: (v) => v || '-' },
    { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 170, render: fmtDateTime },
  ]

  const scheduleColumns = [
    { title: '名称', dataIndex: 'name', key: 'name', width: 200, className: 'tbl-first', render: (v) => <span style={{ fontWeight: 500 }}>{v}</span> },
    { title: '类型', dataIndex: 'type', key: 'type', width: 120, render: (v) => <Tag color="purple" style={{ borderRadius: 4 }}>{v === 'system_info' ? '系统信息' : v}</Tag> },
    { title: '间隔', dataIndex: 'interval_minutes', key: 'interval_minutes', width: 100, render: fmtInterval },
    { title: '上次执行', dataIndex: 'last_run_at', key: 'last_run_at', width: 170, responsive: ['md'], render: fmtDateTime },
    { title: '启用', dataIndex: 'enabled', key: 'enabled', width: 80, align: 'center', render: (v, r) => <Switch size="small" checked={!!v} onChange={() => handleToggleSched(r)} /> },
    {
      title: '操作',
      key: 'actions',
      width: 220,
      render: (_, r) => (
        <>
          <Button type="text" size="small" icon={<PlayCircleOutlined />} loading={runningId === r.id} onClick={() => handleRunSched(r.id)}>立即执行</Button>
          <Button type="text" size="small" onClick={() => openSchedDialog(r)}>编辑</Button>
          <Popconfirm title="确定删除此定时任务？" onConfirm={() => handleDeleteSched(r.id)}>
            <Button type="text" size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </>
      ),
    },
  ]

  const type = Form.useWatch('type', form) || 'bark'

  return (
    <div className="page">
      <PageHead
        title="通知管理"
        sub="配置推送渠道并查看发送记录"
        actions={<>
          <Button icon={<BellOutlined />} loading={sendingExpiry} onClick={handleSendExpiry}>发送到期提醒</Button>
          <Button icon={<ClockCircleOutlined />} onClick={() => openSchedDialog(null)}>新建定时推送</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => openDialog(null)}>添加通知渠道</Button>
        </>}
      />

      <div className="panel notifications-panel">
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'channels',
              label: `通知渠道`,
              children: (
                <Table rowKey="id" columns={channelColumns} dataSource={channels} loading={loading} size="middle" pagination={false} locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无通知渠道" /> }} />
              ),
            },
            {
              key: 'schedules',
              label: `定时推送`,
              children: (
                <div>
                  <Alert type="info" showIcon style={{ marginBottom: 16 }} message="定时推送系统信息将使用您已启用的通知渠道发送（域名/证书数量、30天内到期、注册商、用户、系统版本）；没有启用渠道时不会发送。" />
                  <Table rowKey="id" columns={scheduleColumns} dataSource={schedules} loading={schedLoading} size="middle" pagination={false} locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无定时推送任务，点击右上角「新建定时推送」创建" /> }} />
                </div>
              ),
            },
            {
              key: 'logs',
              label: `发送记录`,
              children: (
                <Table rowKey="id" columns={logColumns} dataSource={logs} loading={logsLoading} size="middle" pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 条` }} locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无发送记录" /> }} scroll={{ x: 1000 }} />
              ),
            },
          ]}
        />
      </div>

      <Modal title={editing ? '编辑通知渠道' : '添加通知渠道'} open={dialogOpen} onOk={handleSave} onCancel={() => setDialogOpen(false)} confirmLoading={submitLoading} width={560} destroyOnClose>
        <Form form={form} layout="vertical" requiredMark={false} style={{ marginTop: 16 }}>
          <Form.Item name="name" label="渠道名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="例如：Bark 推送" />
          </Form.Item>
          <Form.Item name="type" label="通知类型" rules={[{ required: true, message: '请选择类型' }]}>
            <Select options={types.map((t) => ({ value: t.value, label: `${t.label} — ${t.description || ''}` }))} />
          </Form.Item>

          {type === 'bark' && (
            <>
              <Form.Item label="服务器"><Input value={bark.server} onChange={(e) => setBark({ ...bark, server: e.target.value })} placeholder="https://api.day.app" /></Form.Item>
              <Form.Item label="Key" required><Input value={bark.key} onChange={(e) => setBark({ ...bark, key: e.target.value })} placeholder="your-bark-key" /></Form.Item>
              <Form.Item label="分组"><Input value={bark.group} onChange={(e) => setBark({ ...bark, group: e.target.value })} placeholder="域名管理" /></Form.Item>
            </>
          )}
          {type === 'telegram' && (
            <>
              <Form.Item label="Bot Token" required><Input value={telegram.bot_token} onChange={(e) => setTelegram({ ...telegram, bot_token: e.target.value })} placeholder="123456:ABC-DEF..." /></Form.Item>
              <Form.Item label="Chat ID" required><Input value={telegram.chat_id} onChange={(e) => setTelegram({ ...telegram, chat_id: e.target.value })} placeholder="-1001234567890" /></Form.Item>
            </>
          )}
          {type === 'email' && (
            <>
              <Form.Item label="SMTP 服务器" required><Input value={email.smtp_host} onChange={(e) => setEmail({ ...email, smtp_host: e.target.value })} placeholder="smtp.gmail.com" /></Form.Item>
              <Form.Item label="SMTP 端口" required><Input value={email.smtp_port} onChange={(e) => setEmail({ ...email, smtp_port: e.target.value })} placeholder="587" /></Form.Item>
              <Form.Item label="用户名"><Input value={email.username} onChange={(e) => setEmail({ ...email, username: e.target.value })} placeholder="your-email@gmail.com" /></Form.Item>
              <Form.Item label="密码"><Input.Password value={email.password} onChange={(e) => setEmail({ ...email, password: e.target.value })} placeholder="SMTP 授权码" /></Form.Item>
              <Form.Item label="发件人"><Input value={email.from} onChange={(e) => setEmail({ ...email, from: e.target.value })} placeholder="your-email@gmail.com" /></Form.Item>
              <Form.Item label="收件人" required><Input value={email.to} onChange={(e) => setEmail({ ...email, to: e.target.value })} placeholder="多个邮箱用逗号分隔" /></Form.Item>
              <Form.Item label="SSL(465)" valuePropName="checked">
                <Switch checked={email.use_tls} onChange={(v) => setEmail({ ...email, use_tls: v })} />
                <span className="faint" style={{ marginLeft: 8, fontSize: 12 }}>端口 465 使用隐式 TLS；其他端口自动协商 STARTTLS</span>
              </Form.Item>
            </>
          )}
          {type === 'webhook' && (
            <>
              <Form.Item label="URL" required><Input value={webhook.url} onChange={(e) => setWebhook({ ...webhook, url: e.target.value })} placeholder="https://your-webhook-url.com/notify" /></Form.Item>
              <Form.Item label="请求方法">
                <Select value={webhook.method} onChange={(v) => setWebhook({ ...webhook, method: v })} options={[{ value: 'POST', label: 'POST' }, { value: 'PUT', label: 'PUT' }]} />
              </Form.Item>
            </>
          )}
        </Form>
      </Modal>

      <Modal title={schedEditing ? '编辑定时推送' : '新建定时推送'} open={schedDialogOpen} onOk={handleSaveSched} onCancel={() => setSchedDialogOpen(false)} confirmLoading={schedSaving} width={480} destroyOnClose>
        <Form form={schedForm} layout="vertical" requiredMark={false} style={{ marginTop: 16 }}>
          <Form.Item name="name" label="任务名称" rules={[{ required: true, message: '请输入任务名称' }]}>
            <Input placeholder="例如：每日系统信息推送" />
          </Form.Item>
          <Form.Item name="interval_minutes" label="推送间隔（分钟）" rules={[{ required: true, message: '请输入推送间隔' }]} extra="例如 60=每小时、1440=每天；服务每分钟检查一次">
            <InputNumber min={1} max={525600} style={{ width: '100%' }} placeholder="1440" />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Alert type="info" showIcon message="推送内容包含域名/证书总数与30天内到期数量、注册商、用户数量和系统版本，通过您已启用的通知渠道发送。" />
        </Form>
      </Modal>

      <Modal title="测试通知" open={testOpen} onOk={handleTest} onCancel={() => setTestOpen(false)} confirmLoading={testing} width={440} destroyOnClose>
        <Form form={testForm} layout="vertical" requiredMark={false} style={{ marginTop: 16 }}>
          <Form.Item name="title" label="标题" rules={[{ required: true, message: '请输入标题' }]}><Input /></Form.Item>
          <Form.Item name="content" label="内容" rules={[{ required: true, message: '请输入内容' }]}><Input.TextArea rows={3} /></Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
