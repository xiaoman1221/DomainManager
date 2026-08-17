import { useCallback, useEffect, useState } from 'react'
import { Button, Form, Input, Tag, Space, Divider, Popconfirm, Avatar } from 'antd'
import { SaveOutlined, KeyOutlined, LinkOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import AppAvatar from '../components/AppAvatar'
import * as api from '../api/settings'
import * as authApi from '../api/auth'
import { providerMark, providerLabel } from '../utils/oauth'
import { notify } from '../utils/toast'
import PageHead from '../components/PageHead'
import { fmtDateTime } from '../utils/format'

export default function Profile() {
  const { user, logout, fetchProfile } = useAuth()
  const navigate = useNavigate()
  const isAdmin = user?.role_group === 'admin'

  const [profileForm] = Form.useForm()
  const [pwdForm] = Form.useForm()
  const [savingProfile, setSavingProfile] = useState(false)
  const [savingPwd, setSavingPwd] = useState(false)

  const [bindings, setBindings] = useState([])
  const [providers, setProviders] = useState([])
  const [bindingType, setBindingType] = useState(null)

  useEffect(() => {
    if (user) {
      profileForm.setFieldsValue({ nickname: user.nickname || '', email: user.email || '' })
    }
  }, [user, profileForm])

  // OauthGo binding: fetch current bindings + available providers, and surface
  // the ?oauth_bind= success/duplicate/error result from the bind callback.
  const fetchBindings = useCallback(async () => {
    try {
      const res = await authApi.getOauthBindings()
      setBindings(res.data || [])
    } catch {
      /* interceptor */
    }
  }, [])

  const fetchProviders = useCallback(async () => {
    try {
      const res = await authApi.getOauthProviders()
      setProviders(res?.enabled ? res.providers || [] : [])
    } catch {
      /* interceptor */
    }
  }, [])

  useEffect(() => {
    fetchBindings()
    fetchProviders()
  }, [fetchBindings, fetchProviders])

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const status = params.get('oauth_bind')
    if (status) {
      window.history.replaceState({}, '', window.location.pathname)
      if (status === 'success') notify('success', '第三方账号绑定成功')
      else if (status === 'duplicate') notify('error', '该第三方账号已被其他用户绑定')
      else notify('error', '第三方账号绑定失败，请重试')
    }
  }, [])

  const handleBind = async (type) => {
    setBindingType(type)
    try {
      const { url } = await authApi.oauthBind(type)
      if (url) {
        window.location.href = url
        return
      }
      notify('error', '绑定失败，请重试')
    } catch {
      /* interceptor */
    } finally {
      setBindingType(null)
    }
  }

  const handleUnbind = async (provider) => {
    try {
      await authApi.oauthUnbind(provider)
      notify('success', '已解绑')
      fetchBindings()
      fetchProfile()
    } catch {
      /* interceptor */
    }
  }

  const bindableProviders = providers.filter((p) => !bindings.some((b) => b.provider === p.name))

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

  return (
    <div className="page" style={{ maxWidth: 1000 }}>
      <PageHead title="个人设置" sub="头像支持 QQ 邮箱自动识别与 Gravatar" />

      <div className="panel mb-16">
        <div style={{ display: 'flex', gap: 24, padding: 24 }}>
          <AppAvatar email={user?.email} name={user?.nickname || user?.username} avatar={user?.oauth_avatar} size={72} />
          <div style={{ flex: 1 }}>
            <h3 style={{ margin: '0 0 6px', fontSize: 20, fontWeight: 500, letterSpacing: '-0.01em' }}>{user?.nickname || user?.username}</h3>
            <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 12 }}>
              <Tag color={isAdmin ? 'red' : 'default'} style={{ borderRadius: 4 }}>{isAdmin ? '管理员' : '普通用户'}</Tag>
              {user?.user_group ? <Tag style={{ borderRadius: 4 }}>{user.user_group}</Tag> : null}
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

      <div className="panel mt-16">
        <div className="panel-head"><h3 className="panel-title">第三方登录</h3></div>
        <div className="panel-body">
          <p className="muted" style={{ fontSize: 12, lineHeight: 1.7, margin: '0 0 16px' }}>
            绑定后可直接使用第三方账号登录（通过 OauthGo）。同一账号可绑定多个第三方渠道。
          </p>

          {bindings.length > 0 ? (
            <div style={{ display: 'grid', gap: 8, maxWidth: 520 }}>
              {bindings.map((b) => (
                <div key={b.id} className="oauth-binding-item">
                  <Avatar size={34} src={b.avatar || undefined} style={{ background: '#e7e5e4', color: '#1c1917', fontWeight: 600 }}>
                    {(b.nickname || b.provider || '?').charAt(0).toUpperCase()}
                  </Avatar>
                  <div style={{ minWidth: 0, flex: 1 }}>
                    <div style={{ fontWeight: 500, fontSize: 13 }}>{providerLabel(b.provider, b.provider)}</div>
                    <div className="faint" style={{ fontSize: 12 }}>{b.nickname || b.openid}</div>
                  </div>
                  <Popconfirm title={`确定解绑 ${providerLabel(b.provider, b.provider)}？`} onConfirm={() => handleUnbind(b.provider)}>
                    <Button size="small" danger>解绑</Button>
                  </Popconfirm>
                </div>
              ))}
            </div>
          ) : (
            <div className="faint" style={{ fontSize: 13 }}>尚未绑定第三方账号</div>
          )}

          {bindableProviders.length > 0 && (
            <>
              <Divider style={{ margin: '20px 0 16px' }} />
              <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                <span className="muted" style={{ fontSize: 13 }}>绑定新账号：</span>
                {bindableProviders.map((p) => (
                  <Button
                    key={p.name}
                    size="small"
                    icon={<LinkOutlined />}
                    loading={bindingType === p.name}
                    disabled={!!bindingType && bindingType !== p.name}
                    onClick={() => handleBind(p.name)}
                  >
                    <span className="oauth-provider-mark" style={{ width: 16, height: 16, fontSize: 10, marginRight: 4 }}>{providerMark(p)}</span>
                    {p.display_name || p.name}
                  </Button>
                ))}
              </div>
            </>
          )}
        </div>
      </div>

      <div className="panel mt-16" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '14px 20px' }}>
        <span className="muted" style={{ fontSize: 13 }}>退出当前账号</span>
        <Button danger onClick={() => { logout(); navigate('/login') }}>退出登录</Button>
      </div>
    </div>
  )
}
