import { useCallback, useEffect, useRef, useState } from 'react'
import { Button, Form } from 'antd'
import { LoginOutlined } from '@ant-design/icons'
import { useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import AuthDialog from '../components/AuthDialog'
import { notify } from '../utils/toast'
import * as authApi from '../api/auth'

export default function Login() {
  const { login, register, applyAuth } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [form] = Form.useForm()
  const [mode, setMode] = useState('login')
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [oauth, setOauth] = useState({ enabled: false, providers: [], loading: null })
  const isRegister = mode === 'register'
  const [siteConfig, setSiteConfig] = useState(null)
  const ticketHandled = useRef(false)

  // /register opens the popup in register mode; /login keeps it closed until
  // the user clicks "进入控制台". When registration is disabled the /register
  // route redirects to /login and the register form stays hidden.
  const registrationEnabled = siteConfig ? siteConfig.registration_enabled !== false : true
  useEffect(() => {
    if (location.pathname !== '/register') {
      setMode('login')
      setOpen(false)
      return
    }
    if (siteConfig === null) return // wait for the public config to load
    if (!registrationEnabled) {
      setMode('login')
      setOpen(false)
      notify('warning', '系统已关闭注册，请直接登录')
      navigate('/login', { replace: true })
      return
    }
    setMode('register')
    setOpen(true)
  }, [location.pathname, siteConfig, registrationEnabled, navigate])

  // Load the enabled OauthGo channels for the login dialog.
  useEffect(() => {
    let cancelled = false
    authApi
      .getOauthProviders()
      .then((res) => {
        if (!cancelled) {
          setOauth((prev) => ({
            ...prev,
            enabled: !!res?.enabled,
            providers: res?.providers || [],
          }))
        }
      })
      .catch(() => {
        // OauthGo is optional; keep third-party login hidden on failure.
      })
    return () => {
      cancelled = true
    }
  }, [])

  // Load public site config (footer / SNS links).
  useEffect(() => {
    let cancelled = false
    authApi
      .getSiteConfig()
      .then((res) => {
        if (!cancelled) setSiteConfig(res || {})
      })
      .catch(() => {
        // Fall back to defaults (registration enabled) when the public config
        // cannot be loaded; registration is still enforced server-side (403).
        if (!cancelled) setSiteConfig({})
      })
    return () => {
      cancelled = true
    }
  }, [])
  // Handle the redirect back from OauthGo: /login?oauth_ticket=<ticket>.
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const ticket = params.get('oauth_ticket')
    const oauthError = params.get('oauth_error')
    const cleanUrl = () => window.history.replaceState({}, '', window.location.pathname)

    if (oauthError) {
      cleanUrl()
      if (oauthError === 'not_bound') {
        notify('error', '该第三方账号未绑定任何本地账号，请先登录后到「个人设置 → 第三方登录」绑定')
      } else {
        notify('error', '第三方登录失败，请重试')
      }
    }

    if (!ticket || ticketHandled.current) return
    ticketHandled.current = true

    setLoading(true)
    ;(async () => {
      try {
        const res = await authApi.redeemOauthTicket(ticket)
        applyAuth(res)
        cleanUrl()
        notify('success', '登录成功')
        navigate('/')
      } catch {
        cleanUrl()
        // error already toasted by the interceptor
      } finally {
        setLoading(false)
      }
    })()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const switchMode = useCallback((next) => {
    setMode(next)
    form.resetFields()
  }, [form])

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

  const handleOauthLogin = async (type) => {
    setOauth((prev) => ({ ...prev, loading: type }))
    try {
      const { url } = await authApi.oauthLogin(type)
      if (url) {
        window.location.href = url
        return
      }
      notify('error', '第三方登录暂不可用')
    } catch {
      // handled by interceptor
    } finally {
      setOauth((prev) => ({ ...prev, loading: null }))
    }
  }

  return (
    <div className="auth-shell">
      <div className="auth-brand">
        <div>
          <h1>集中管理每一个域名。</h1>
          <p className="lead">
            Domain Manager 将域名、WHOIS、ICP 备案、续费价格与证书集中到一处，
            到期提醒直达手机与邮箱。
          </p>
          <ul className="auth-features">
            <li><span className="dot" />WHOIS / RDAP 与 ICP 备案一键查询</li>
            <li><span className="dot" />多注册商域名批量导入与续费比价</li>
            <li><span className="dot" />证书生命周期跟踪，Certimate 同步</li>
            <li><span className="dot" />Bark / Telegram / 邮件 / Webhook 到期提醒</li>
          </ul>
        </div>
                <div className="footnote">
          <div>{siteConfig?.footer?.description || 'Domain Manager · 域名管理与比价平台'}</div>
          {(siteConfig?.sns || []).length > 0 && (
            <div className="footnote-links">
              {(siteConfig.sns || []).map((s, i) => (
                <a key={i} href={s.url} target="_blank" rel="noreferrer">{s.label}</a>
              ))}
            </div>
          )}
          {(siteConfig?.footer?.links || []).length > 0 && (
            <div className="footnote-links">
              {(siteConfig.footer.links || []).map((l, i) => (
                <a key={i} href={l.url} target="_blank" rel="noreferrer">{l.label}</a>
              ))}
            </div>
          )}
          <div>
            {siteConfig?.footer?.copyright ? <span>{siteConfig.footer.copyright}</span> : null}
            {siteConfig?.footer?.icp ? <a className="footnote-icp" href="https://beian.miit.gov.cn/" target="_blank" rel="noreferrer">{siteConfig.footer.icp}</a> : null}
            {siteConfig?.footer?.police ? <span className="footnote-icp">{siteConfig.footer.police}</span> : null}
          </div>
        </div>
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
        oauth={oauth}
        onOauthLogin={handleOauthLogin}
        registrationEnabled={registrationEnabled}
      />
    </div>
  )
}



