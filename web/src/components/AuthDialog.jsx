import { Form, Input, Button } from 'antd'
import { UserOutlined, LockOutlined, MailOutlined, ArrowRightOutlined } from '@ant-design/icons'

import { providerMark } from '../utils/oauth'

export default function AuthDialog({ open, onClose, mode, onSwitchMode, loading, onFinish, form, oauth, onOauthLogin, registrationEnabled = true }) {
  // When registration is disabled the register form and the switch link are
  // hidden; the dialog degrades to login-only.
  const canRegister = registrationEnabled !== false
  const isRegister = mode === 'register' && canRegister
  const showOauth = !isRegister && oauth?.enabled && Array.isArray(oauth.providers) && oauth.providers.length > 0

  // The overlay stays mounted and is hidden with CSS instead of being removed
  // from the DOM. Removing the form nodes while password-manager extensions
  // (e.g. Bitwarden) are observing them makes the extension throw
  // "NotFoundError: insertBefore ... not a child of this node".
  return (
    <div className={`auth-modal${open ? '' : ' is-hidden'}`} onClick={onClose} aria-hidden={!open}>
      <div className="auth-dialog" onClick={(e) => e.stopPropagation()}>
        <button className="auth-dialog-close" onClick={onClose} aria-label="关闭">×</button>
        <div className="auth-dialog-head">
          <div className="app-logo-mark">DM</div>
          <div>
            <div className="auth-dialog-title">{isRegister ? '创建账号' : '欢迎回来'}</div>
            <div className="auth-dialog-sub">
              {isRegister ? '用户名至少 3 位，密码至少 6 位' : '登录你的账号，继续管理域名资产'}
            </div>
          </div>
        </div>
        <Form form={form} layout="vertical" onFinish={onFinish} requiredMark={false} size="large">
          {isRegister && (
            <Form.Item name="email" rules={[{ required: true, message: '请输入邮箱' }, { type: 'email', message: '邮箱格式不正确' }]}>
              <Input prefix={<MailOutlined style={{ color: '#a8a29e' }} />} placeholder="邮箱地址" autoComplete="email" />
            </Form.Item>
          )}
          <Form.Item
            name="username"
            rules={[{ required: true, message: '请输入用户名' }, ...(isRegister ? [{ min: 3, message: '用户名至少 3 个字符' }] : [])]}
          >
            <Input prefix={<UserOutlined style={{ color: '#a8a29e' }} />} placeholder="用户名" autoComplete="username" />
          </Form.Item>
          <Form.Item
            name="password"
            rules={[{ required: true, message: '请输入密码' }, ...(isRegister ? [{ min: 6, message: '密码至少 6 个字符' }] : [])]}
          >
            <Input.Password prefix={<LockOutlined style={{ color: '#a8a29e' }} />} placeholder="密码" autoComplete={isRegister ? 'new-password' : 'current-password'} />
          </Form.Item>
          {isRegister && (
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
          )}
          <Form.Item style={{ marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" loading={loading} block icon={<ArrowRightOutlined />}>
              {isRegister ? '注册' : '登录'}
            </Button>
          </Form.Item>
        </Form>

        {showOauth && (
          <div className="oauth-login">
            <div className="auth-divider"><span>或使用第三方账号登录</span></div>
            <div className="oauth-providers">
              {oauth.providers.map((p) => (
                <Button
                  key={p.name}
                  className="oauth-provider-btn"
                  loading={oauth.loading === p.name}
                  disabled={!!oauth.loading && oauth.loading !== p.name}
                  onClick={() => onOauthLogin(p.name)}
                >
                  <span className="oauth-provider-mark">{providerMark(p)}</span>
                  {p.display_name || p.name}
                </Button>
              ))}
            </div>
          </div>
        )}

        <div className="auth-dialog-footer">
          {isRegister ? (
            <>已有账号？<a onClick={() => onSwitchMode('login')}>去登录</a></>
          ) : canRegister ? (
            <>还没有账号？<a onClick={() => onSwitchMode('register')}>立即注册</a></>
          ) : (
            <span className="muted" style={{ fontSize: 12 }}>当前未开放注册</span>
          )}
        </div>
      </div>
    </div>
  )
}
