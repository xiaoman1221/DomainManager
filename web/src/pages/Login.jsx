import { useState } from 'react'
import { Button, Form, Input } from 'antd'
import { UserOutlined, LockOutlined, ArrowRightOutlined } from '@ant-design/icons'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { notify } from '../utils/toast'

export default function Login() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)

  const onFinish = async (values) => {
    setLoading(true)
    try {
      await login(values)
      notify('success', '登录成功')
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
          <h1>集中管理你的每一个域名。</h1>
          <p className="lead">
            Domain Manager 将域名、WHOIS、ICP 备案、续费价格与证书集中到一处，
            到期提醒直达你的手机与邮箱。
          </p>
          <ul className="auth-features">
            <li><span className="dot" />WHOIS / RDAP 与 ICP 备案一键查询</li>
            <li><span className="dot" />多注册商域名批量导入与续费比价</li>
            <li><span className="dot" />证书生命周期跟踪，Certimate 同步</li>
            <li><span className="dot" />Bark / Telegram / 邮件 / Webhook 到期提醒</li>
          </ul>
        </div>
        <div className="footnote">Domain Manager · 域名管理与比价平台</div>
      </div>
      <div className="auth-form">
        <div className="auth-card">
          <h2>登录</h2>
          <p className="sub">使用你的账号继续</p>
          <Form layout="vertical" onFinish={onFinish} requiredMark={false} size="large">
            <Form.Item name="username" rules={[{ required: true, message: '请输入用户名' }]}>
              <Input prefix={<UserOutlined style={{ color: '#a8a29e' }} />} placeholder="用户名" autoComplete="username" />
            </Form.Item>
            <Form.Item name="password" rules={[{ required: true, message: '请输入密码' }]}>
              <Input.Password prefix={<LockOutlined style={{ color: '#a8a29e' }} />} placeholder="密码" autoComplete="current-password" />
            </Form.Item>
            <Form.Item style={{ marginBottom: 12 }}>
              <Button type="primary" htmlType="submit" loading={loading} block icon={<ArrowRightOutlined />}>
                登录
              </Button>
            </Form.Item>
          </Form>
          <div className="auth-alt">
            还没有账号？<Link to="/register">立即注册</Link>
          </div>
        </div>
      </div>
    </div>
  )
}
