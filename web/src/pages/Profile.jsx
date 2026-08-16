import { useEffect, useState } from 'react'
import { Button, Form, Input, Table, Select, Modal, Tag, Descriptions, Divider, Empty, Popconfirm } from 'antd'
import { SaveOutlined, KeyOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import AppAvatar from '../components/AppAvatar'
import * as api from '../api/settings'
import { notify } from '../utils/toast'
import PageHead from '../components/PageHead'
import { fmtDateTime } from '../utils/format'
import { useIsMobile } from '../utils/useIsMobile'

export default function Profile() {
  const { user, logout, fetchProfile } = useAuth()
  const navigate = useNavigate()
  const isAdmin = user?.role === 'admin'
  const isMobile = useIsMobile()

  const [profileForm] = Form.useForm()
  const [pwdForm] = Form.useForm()
  const [savingProfile, setSavingProfile] = useState(false)
  const [savingPwd, setSavingPwd] = useState(false)

  const [users, setUsers] = useState([])
  const [usersLoading, setUsersLoading] = useState(false)
  const [sysInfo, setSysInfo] = useState(null)

  const [resetOpen, setResetOpen] = useState(false)
  const [resetUser, setResetUser] = useState(null)
  const [resetForm] = Form.useForm()
  const [resetting, setResetting] = useState(false)

  useEffect(() => {
    if (user) {
      profileForm.setFieldsValue({ nickname: user.nickname || '', email: user.email || '' })
    }
  }, [user, profileForm])

  const fetchUsers = async () => {
    setUsersLoading(true)
    try {
      const res = await api.getUsers()
      setUsers(res.data || [])
    } finally {
      setUsersLoading(false)
    }
  }

  useEffect(() => {
    if (isAdmin) {
      fetchUsers()
      api.getSystemInfo().then(setSysInfo).catch(() => {})
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAdmin])

  const handleSaveProfile = async () => {
    const values = await profileForm.validateFields()
    setSavingProfile(true)
    try {
      await api.updateUserProfile(values)
      await fetchProfile()
      notify('success', '资料已更新')
    } catch {
      /* interceptor */
    } finally {
      setSavingProfile(false)
    }
  }

  const handleChangePwd = async () => {
    const values = await pwdForm.validateFields()
    setSavingPwd(true)
    try {
      await api.changePassword({ old_password: values.old_password, new_password: values.new_password })
      notify('success', '密码已修改')
      pwdForm.resetFields()
    } catch {
      /* interceptor */
    } finally {
      setSavingPwd(false)
    }
  }

  const handleRoleChange = async (row, role) => {
    try {
      await api.updateUserRole(row.id, { role })
      notify('success', '角色已更新')
      fetchUsers()
    } catch {
      /* interceptor */
    }
  }

  const openReset = (row) => {
    setResetUser(row)
    resetForm.resetFields()
    setResetOpen(true)
  }

  const handleReset = async () => {
    const values = await resetForm.validateFields()
    setResetting(true)
    try {
      await api.adminUpdatePassword(resetUser.id, { password: values.password })
      notify('success', '密码已重置')
      setResetOpen(false)
    } catch {
      /* interceptor */
    } finally {
      setResetting(false)
    }
  }

  const userColumns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60, responsive: ['md'] },
    { title: '用户名', dataIndex: 'username', key: 'username', width: 140, className: 'tbl-first', render: (v) => <span style={{ fontWeight: 500 }}>{v}</span> },
    { title: '邮箱', dataIndex: 'email', key: 'email', width: 220, responsive: ['md'] },
    { title: '昵称', dataIndex: 'nickname', key: 'nickname', width: 120, responsive: ['md'], render: (v) => v || '-' },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      width: 130,
      render: (v, r) => (
        <Select
          size="small"
          value={v}
          disabled={r.username === 'admin'}
          onChange={(nv) => handleRoleChange(r, nv)}
          options={[{ value: 'admin', label: '管理员' }, { value: 'user', label: '普通用户' }]}
          style={{ width: 110 }}
        />
      ),
    },
    { title: '注册时间', dataIndex: 'created_at', key: 'created_at', width: 170, responsive: ['md'], render: fmtDateTime },
    {
      title: '操作',
      key: 'actions',
      width: 110,
      render: (_, r) => (
        <Popconfirm title={`确定重置 ${r.username} 的密码？`} onConfirm={() => openReset(r)}>
          <Button type="text" size="small" icon={<KeyOutlined />}>重置密码</Button>
        </Popconfirm>
      ),
    },
  ]

  return (
    <div className="page" style={{ maxWidth: 1000 }}>
      <PageHead title="个人设置" sub="头像支持 QQ 邮箱自动识别与 Gravatar" />

      <div className="panel mb-16">
        <div style={{ display: 'flex', gap: 24, padding: 24 }}>
          <AppAvatar email={user?.email} name={user?.nickname || user?.username} size={72} />
          <div style={{ flex: 1 }}>
            <h3 style={{ margin: '0 0 6px', fontSize: 20, fontWeight: 500, letterSpacing: '-0.01em' }}>{user?.nickname || user?.username}</h3>
            <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 12 }}>
              <Tag color={isAdmin ? 'red' : 'default'} style={{ borderRadius: 4 }}>{isAdmin ? '管理员' : '普通用户'}</Tag>
              <span className="muted" style={{ fontSize: 13 }}>{user?.email}</span>
            </div>
            <div className="faint" style={{ fontSize: 12 }}>注册于 {fmtDateTime(user?.created_at)}</div>
          </div>
        </div>
      </div>

      <div className="two-col">
        <div className="panel">
          <div className="panel-head"><h3 className="panel-title">基本资料</h3></div>
          <div className="panel-body">
            <Form form={profileForm} layout="vertical" requiredMark={false}>
              <Form.Item label="用户名"><Input value={user?.username} disabled /></Form.Item>
              <Form.Item name="nickname" label="昵称"><Input placeholder="设置昵称" /></Form.Item>
              <Form.Item name="email" label="邮箱" rules={[{ required: true, message: '请输入邮箱' }, { type: 'email', message: '邮箱格式不正确' }]}><Input placeholder="邮箱地址" /></Form.Item>
              <Button type="primary" icon={<SaveOutlined />} loading={savingProfile} onClick={handleSaveProfile}>保存资料</Button>
            </Form>
          </div>
        </div>

        <div className="panel">
          <div className="panel-head"><h3 className="panel-title">修改密码</h3></div>
          <div className="panel-body">
            <Form form={pwdForm} layout="vertical" requiredMark={false}>
              <Form.Item name="old_password" label="当前密码" rules={[{ required: true, message: '请输入当前密码' }]}>
                <Input.Password autoComplete="current-password" />
              </Form.Item>
              <Form.Item name="new_password" label="新密码" rules={[{ required: true, message: '请输入新密码' }, { min: 6, message: '密码至少 6 位' }]}>
                <Input.Password autoComplete="new-password" />
              </Form.Item>
              <Form.Item
                name="confirm"
                label="确认新密码"
                dependencies={['new_password']}
                rules={[
                  { required: true, message: '请确认新密码' },
                  ({ getFieldValue }) => ({
                    validator(_, value) {
                      if (!value || getFieldValue('new_password') === value) return Promise.resolve()
                      return Promise.reject(new Error('两次输入的密码不一致'))
                    },
                  }),
                ]}
              >
                <Input.Password autoComplete="new-password" />
              </Form.Item>
              <Button icon={<KeyOutlined />} loading={savingPwd} onClick={handleChangePwd}>修改密码</Button>
            </Form>
          </div>
        </div>
      </div>

      <div className="panel mt-16" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '14px 20px' }}>
        <span className="muted" style={{ fontSize: 13 }}>退出当前账号</span>
        <Button danger onClick={() => { logout(); navigate('/login') }}>退出登录</Button>
      </div>

      {isAdmin && (
        <div className="panel mt-16">
          <div className="panel-head"><h3 className="panel-title">系统管理</h3></div>
          <div style={{ padding: 20 }}>
            <h4 style={{ margin: '0 0 12px', fontSize: 14, fontWeight: 600 }}>用户管理</h4>
            <Table rowKey="id" columns={userColumns} dataSource={users} loading={usersLoading} size="middle" pagination={false} scroll={{ x: 900 }} locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无用户" /> }} />

            <Divider />

            <h4 style={{ margin: '0 0 12px', fontSize: 14, fontWeight: 600 }}>系统信息</h4>
            {sysInfo ? (
              <Descriptions column={isMobile ? 1 : 3} size="small" bordered>
                <Descriptions.Item label="域名总数">{sysInfo.domains}</Descriptions.Item>
                <Descriptions.Item label="证书总数">{sysInfo.certificates}</Descriptions.Item>
                <Descriptions.Item label="注册商数量">{sysInfo.registrars}</Descriptions.Item>
                <Descriptions.Item label="用户数量">{sysInfo.users}</Descriptions.Item>
                <Descriptions.Item label="系统版本">{sysInfo.version}</Descriptions.Item>
              </Descriptions>
            ) : null}
          </div>
        </div>
      )}

      <Modal title={`重置 ${resetUser?.username || ''} 的密码`} open={resetOpen} onOk={handleReset} onCancel={() => setResetOpen(false)} confirmLoading={resetting} width={420} destroyOnClose>
        <Form form={resetForm} layout="vertical" requiredMark={false} style={{ marginTop: 16 }}>
          <Form.Item name="password" label="新密码" rules={[{ required: true, message: '请输入新密码' }, { min: 6, message: '密码至少 6 位' }]}>
            <Input.Password autoComplete="new-password" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
