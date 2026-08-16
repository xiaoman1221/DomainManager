import { useState } from 'react'
import { Button, Form, Input } from 'antd'
import { UserOutlined, LockOutlined, MailOutlined, ArrowRightOutlined } from '@ant-design/icons'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { notify } from '../utils/toast'

export default function Register() {
  const { register } = useAuth()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)

  const onFinish = async (values) => {
    setLoading(true)
    try {
      await register({
        username: values.username,
        email: values.email,
        password: values.password,
      })
      notify('success', '注册成功，欢迎使用')
      navigate('/')
    } catch {
      // handled by interceptor
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="auth-shell">
      <div className="auth-brand">
        <div>
          <h1>从一个账号，掌控全部资产。</h1>
          <p className="lead">
            注册即可开始管理域名与证书。头像支持 QQ 邮箱头像自动识别与
            Gravatar 全局头像。
          </p>
          <ul className="auth-features">
            <li><span className="dot" />每行一个域名的批量导入</li>
            <li><span className="dot" />注册商 API 自动同步（阿里云 / 腾讯云 / Cloudflare…）</li>
            <li><span className="dot" />到期前 30 天分级提醒</li>
          </ul>
        </div>
        <div className="footnote">Domain Manager · 域名管理与比价平台</div>
      </div>
      <div className="auth-form">
        <div className="auth-card">
          <h2>注册账号</h2>
          <p className="sub">用户名至少 3 位，密码至少 6 位</p>
          <Form layout="vertical" onFinish={onFinish} requiredMark={false} size="large">
            <Form.Item name="username" rules={[{ required: true, message: '请输入用户名' }, { min: 3, message: '用户名至少 3 个字符' }]}>
              <Input prefix={<UserOutlined style={{ color: '#a8a29e' }} />} placeholder="用户名" autoComplete="username" />
            </Form.Item>
            <Form.Item name="email" rules={[{ required: true, message: '请输入邮箱' }, { type: 'email', message: '邮箱格式不正确' }]}>
              <Input prefix={<MailOutlined style={{ color: '#a8a29e' }} />} placeholder="邮箱地址" autoComplete="email" />
            </Form.Item>
            <Form.Item name="password" rules={[{ required: true, message: '请输入密码' }, { min: 6, message: '密码至少 6 个字符' }]}>
              <Input.Password prefix={<LockOutlined style={{ color: '#a8a29e' }} />} placeholder="密码" autoComplete="new-password" />
            </Form.Item>
            <Form.Item
              name="confirm"
              dependencies={['password']}
              rules={[
                { required: true, message: '请再次输入密码' },
                ({ getFieldValue }) => ({
                  validator(_, value) {
                    if (!value || getFieldValue('password') === value) return Promise.resolve()
                    return Promise.reject(new Error('两次输入的密码不一致'))
                  },
                }),
              ]}
            >
              <Input.Password prefix={<LockOutlined style={{ color: '#a8a29e' }} />} placeholder="确认密码" autoComplete="new-password" />
            </Form.Item>
            <Form.Item style={{ marginBottom: 12 }}>
              <Button type="primary" htmlType="submit" loading={loading} block icon={<ArrowRightOutlined />}>
                注册
              </Button>
            </Form.Item>
          </Form>
          <div className="auth-alt">
            已有账号？<Link to="/login">去登录</Link>
          </div>
        </div>
      </div>
    </div>
  )
}
