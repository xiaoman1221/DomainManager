import { useCallback, useEffect, useState } from 'react'
import {
  Button, Table, Modal, Form, Input, Select, Tag, Popconfirm, Drawer, Descriptions, Empty, Space,
} from 'antd'
import { EyeOutlined, EditOutlined, DeleteOutlined, KeyOutlined, ReloadOutlined } from '@ant-design/icons'
import { useAuth } from '../context/AuthContext'
import AppAvatar from '../components/AppAvatar'
import * as api from '../api/settings'
import { notify } from '../utils/toast'
import PageHead from '../components/PageHead'
import { fmtDateTime } from '../utils/format'
import { useIsMobile } from '../utils/useIsMobile'

export default function Users() {
  const { user: me } = useAuth()
  const isMobile = useIsMobile()

  const [users, setUsers] = useState([])
  const [loading, setLoading] = useState(false)

  const [viewUser, setViewUser] = useState(null)
  const [editUser, setEditUser] = useState(null)
  const [editForm] = Form.useForm()
  const [saving, setSaving] = useState(false)
  const [resetOpen, setResetOpen] = useState(false)
  const [resetForm] = Form.useForm()
  const [resetting, setResetting] = useState(false)

  const fetchUsers = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.getUsers()
      setUsers(res.data || [])
    } catch {
      /* interceptor */
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchUsers()
  }, [fetchUsers])

  const openEdit = (row) => {
    setEditUser(row)
    editForm.setFieldsValue({
      nickname: row.nickname || '',
      email: row.email || '',
      role_group: row.role_group || 'user',
      user_group: row.user_group || '',
    })
  }

  const handleSave = async () => {
    const values = await editForm.validateFields()
    setSaving(true)
    try {
      await api.updateUser(editUser.id, values)
      notify('success', '用户已更新')
      setEditUser(null)
      fetchUsers()
    } catch {
      /* interceptor */
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (row) => {
    try {
      await api.deleteUser(row.id)
      notify('success', '用户已删除')
      fetchUsers()
    } catch {
      /* interceptor */
    }
  }

  const openReset = () => {
    resetForm.resetFields()
    setResetOpen(true)
  }

  const handleReset = async () => {
    const values = await resetForm.validateFields()
    setResetting(true)
    try {
      await api.adminUpdatePassword(editUser.id, { password: values.password })
      notify('success', '密码已重置')
      setResetOpen(false)
    } catch {
      /* interceptor */
    } finally {
      setResetting(false)
    }
  }

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 64, responsive: ['md'] },
    {
      title: '用户',
      dataIndex: 'username',
      key: 'username',
      width: 200,
      className: 'tbl-first',
      render: (v, r) => (
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <AppAvatar email={r.email} name={r.nickname || r.username} avatar={r.oauth_avatar} size={30} />
          <div style={{ minWidth: 0 }}>
            <div style={{ fontWeight: 500, lineHeight: 1.3 }}>{v}</div>
            <div className="faint" style={{ fontSize: 12 }}>{r.nickname || '—'}</div>
          </div>
        </div>
      ),
    },
    { title: '邮箱', dataIndex: 'email', key: 'email', width: 220, render: (v) => <span className="muted">{v}</span> },
    {
      title: '角色组',
      dataIndex: 'role_group',
      key: 'role_group',
      width: 110,
      render: (v) => <Tag color={v === 'admin' ? 'red' : 'default'} style={{ borderRadius: 4 }}>{v === 'admin' ? '管理员' : '普通用户'}</Tag>,
    },
    { title: '用户组', dataIndex: 'user_group', key: 'user_group', width: 120, responsive: ['md'], render: (v) => v ? <Tag style={{ borderRadius: 4 }}>{v}</Tag> : <span className="faint">—</span> },
    { title: '注册时间', dataIndex: 'created_at', key: 'created_at', width: 160, responsive: ['md'], render: fmtDateTime },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      render: (_, r) => (
        <Space size={0}>
          <Button type="text" size="small" icon={<EyeOutlined />} onClick={() => setViewUser(r)}>查看</Button>
          <Button type="text" size="small" icon={<EditOutlined />} onClick={() => openEdit(r)}>编辑</Button>
          <Popconfirm title={`确定删除用户 ${r.username}？`} description="删除后该用户将无法登录" onConfirm={() => handleDelete(r)}>
            <Button
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              disabled={r.id === me?.id}
            >
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div className="page">
      <PageHead title="用户管理" sub="查看、编辑与删除系统用户（仅管理员可用）" />

      <div className="panel">
        <div className="panel-head" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <h3 className="panel-title">全部用户</h3>
          <Button size="small" icon={<ReloadOutlined />} onClick={fetchUsers}>刷新</Button>
        </div>
        <div style={{ padding: '0 20px 20px' }}>
          <Table
            rowKey="id"
            columns={columns}
            dataSource={users}
            loading={loading}
            size="middle"
            pagination={{ pageSize: 10, showSizeChanger: false }}
            scroll={{ x: 900 }}
            locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无用户" /> }}
          />
        </div>
      </div>

      {/* 查看详情 */}
      <Drawer
        title="用户详情"
        open={!!viewUser}
        onClose={() => setViewUser(null)}
        width={isMobile ? '100%' : 420}
      >
        {viewUser && (
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 14, marginBottom: 20 }}>
              <AppAvatar email={viewUser.email} name={viewUser.nickname || viewUser.username} avatar={viewUser.oauth_avatar} size={56} />
              <div>
                <div style={{ fontSize: 17, fontWeight: 600 }}>{viewUser.nickname || viewUser.username}</div>
                <div className="muted" style={{ fontSize: 13 }}>@{viewUser.username}</div>
              </div>
            </div>
            <Descriptions column={1} size="small" bordered>
              <Descriptions.Item label="ID">{viewUser.id}</Descriptions.Item>
              <Descriptions.Item label="邮箱">{viewUser.email}</Descriptions.Item>
              <Descriptions.Item label="角色组">
                {viewUser.role_group === 'admin' ? '管理员' : '普通用户'}
              </Descriptions.Item>
              <Descriptions.Item label="用户组">{viewUser.user_group || '—'}</Descriptions.Item>
              <Descriptions.Item label="第三方登录">
                {viewUser.oauth_provider ? `${viewUser.oauth_provider} (${viewUser.oauth_openid})` : '—'}
              </Descriptions.Item>
              <Descriptions.Item label="注册时间">{fmtDateTime(viewUser.created_at)}</Descriptions.Item>
              <Descriptions.Item label="最近更新">{fmtDateTime(viewUser.updated_at)}</Descriptions.Item>
            </Descriptions>
            <div style={{ marginTop: 20, display: 'flex', justifyContent: 'flex-end' }}>
              <Button icon={<EditOutlined />} onClick={() => { setViewUser(null); openEdit(viewUser) }}>编辑</Button>
            </div>
          </div>
        )}
      </Drawer>

      {/* 编辑 */}
      <Modal
        title={`编辑用户 · ${editUser?.username || ''}`}
        open={!!editUser}
        onOk={handleSave}
        onCancel={() => setEditUser(null)}
        confirmLoading={saving}
        width={460}
        destroyOnClose
      >
        <Form form={editForm} layout="vertical" requiredMark={false} style={{ marginTop: 16 }}>
          <Form.Item name="nickname" label="昵称"><Input placeholder="用户昵称" /></Form.Item>
          <Form.Item name="email" label="邮箱" rules={[{ required: true, message: '请输入邮箱' }, { type: 'email', message: '邮箱格式不正确' }]}>
            <Input placeholder="邮箱地址" />
          </Form.Item>
          <Form.Item name="role_group" label="角色组" rules={[{ required: true, message: '请选择角色组' }]}>
            <Select options={[{ value: 'admin', label: '管理员' }, { value: 'user', label: '普通用户' }]} />
          </Form.Item>
          <Form.Item name="user_group" label="用户组"><Input placeholder="如：运维组、运营组（仅作标记）" maxLength={64} /></Form.Item>
          <div className="panel-body" style={{ padding: 0 }}>
            <Button icon={<KeyOutlined />} onClick={openReset}>重置该用户密码</Button>
          </div>
        </Form>
      </Modal>

      {/* 重置密码 */}
      <Modal title={`重置 ${editUser?.username || ''} 的密码`} open={resetOpen} onOk={handleReset} onCancel={() => setResetOpen(false)} confirmLoading={resetting} width={420} destroyOnClose>
        <Form form={resetForm} layout="vertical" requiredMark={false} style={{ marginTop: 16 }}>
          <Form.Item name="password" label="新密码" rules={[{ required: true, message: '请输入新密码' }, { min: 6, message: '密码至少 6 位' }]}>
            <Input.Password autoComplete="new-password" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
