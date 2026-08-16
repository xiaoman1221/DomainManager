import { useEffect, useState } from 'react'
import { Button, Form } from 'antd'
import { LoginOutlined } from '@ant-design/icons'
import { useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import AuthDialog from '../components/AuthDialog'
import { notify } from '../utils/toast'

export default function Login() {
  const { login, register } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [form] = Form.useForm()
  const [mode, setMode] = useState('login')
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const isRegister = mode === 'register'

  // /register opens the popup in register mode; /login keeps it closed until
  // the user clicks "进入控制台".
  useEffect(() => {
    if (location.pathname === '/register') {
      setMode('register')
      setOpen(true)
    } else {
      setMode('login')
      setOpen(false)
    }
  }, [location.pathname])

  const switchMode = (next) => {
    setMode(next)
    form.resetFields()
  }

  const onFinish = async (values) => {
    setLoading(true)
    try {
      if (isRegister) {
        await register({ username: values.username, email: values.email, password: values.password })
        notify('success', '注册成功，欢迎使用')
      } else {
        await login(values)
        notify('success', '登录成功')
      }
      setOpen(false)
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
        <div className="auth-console">
          <p className="auth-console-sub">
            登录后进入控制台，集中管理域名、证书与到期提醒。
          </p>
          <Button
            type="primary"
            size="large"
            className="auth-enter-btn"
            icon={<LoginOutlined />}
            onClick={() => { switchMode('login'); setOpen(true) }}
          >
            进入控制台
          </Button>
        </div>
      </div>

      <AuthDialog
        open={open}
        onClose={() => setOpen(false)}
        mode={mode}
        onSwitchMode={switchMode}
        loading={loading}
        onFinish={onFinish}
        form={form}
      />
    </div>
  )
}
