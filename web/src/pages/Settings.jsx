import { useEffect, useState } from 'react'
import { Button, Form, Input, Select, Switch, Tabs, Descriptions, Space, Checkbox } from 'antd'
import { SaveOutlined, PlusOutlined, MinusCircleOutlined } from '@ant-design/icons'
import * as api from '../api/settings'
import * as authApi from '../api/auth'
import { notify } from '../utils/toast'
import PageHead from '../components/PageHead'
import { useIsMobile } from '../utils/useIsMobile'

const PAYMENT_PROVIDERS = [
  { value: 'alipay', label: '支付宝' },
  { value: 'wechat', label: '微信支付' },
  { value: 'paypal', label: 'PayPal' },
  { value: 'stripe', label: 'Stripe' },
]

const SMTP_ENCRYPTIONS = [
  { value: 'none', label: '无（明文）' },
  { value: 'ssl', label: 'SSL/TLS（隐式，通常 465）' },
  { value: 'starttls', label: 'STARTTLS（通常 587）' },
]

function parseJSONArray(raw) {
  if (!raw) return []
  try {
    const arr = JSON.parse(raw)
    return Array.isArray(arr) ? arr : []
  } catch {
    return []
  }
}

export default function Settings() {
  const isMobile = useIsMobile()
  const [basicForm] = Form.useForm()
  const [accountForm] = Form.useForm()
  const [oauthForm] = Form.useForm()
  const [smtpForm] = Form.useForm()
  const [paymentForm] = Form.useForm()
  const [snsForm] = Form.useForm()
  const [footerForm] = Form.useForm()

  const [sysInfo, setSysInfo] = useState(null)
  const [oauthProviders, setOauthProviders] = useState([])
  const [saving, setSaving] = useState('')
  const oauthProvider = Form.useWatch('OAUTH_PROVIDER', oauthForm) || 'oauthgo'

  useEffect(() => {
    api.getSystemSettings().then((res) => {
      const d = res || {}
      basicForm.setFieldsValue({
        WHOIS_API_URL: d.WHOIS_API_URL || '',
        UA_WHOIS_SERVER: d.UA_WHOIS_SERVER || 'whois.ua:43',
        ICP_API_URL: d.ICP_API_URL || '',
        DIGITALPLAT_RDAP_URL: d.DIGITALPLAT_RDAP_URL || '',
        PROXY_URL: d.PROXY_URL || '',
      })
      const isRainbow = (d.OAUTH_PROVIDER || 'oauthgo') === 'rainbow'
      let oauthTypes = []
      try {
        oauthTypes = JSON.parse(d[isRainbow ? 'RAINBOW_ENABLED_TYPES' : 'OAUTHGO_ENABLED_TYPES'] || '[]') || []
      } catch { /* ignore */ }
      accountForm.setFieldsValue({
        REGISTRATION_ENABLED: d.REGISTRATION_ENABLED !== 'false',
        OAUTHGO_AUTO_REGISTER: d.OAUTHGO_AUTO_REGISTER !== 'false',
      })
      oauthForm.setFieldsValue({
        OAUTH_PROVIDER: d.OAUTH_PROVIDER || 'oauthgo',
        OAUTHGO_BASE_URL: d.OAUTHGO_BASE_URL || '',
        OAUTHGO_APP_ID: d.OAUTHGO_APP_ID || '',
        OAUTHGO_APP_KEY: d.OAUTHGO_APP_KEY || '',
        OAUTHGO_REDIRECT_URI: d.OAUTHGO_REDIRECT_URI || '',
        RAINBOW_BASE_URL: d.RAINBOW_BASE_URL || '',
        RAINBOW_APP_ID: d.RAINBOW_APP_ID || '',
        RAINBOW_APP_KEY: d.RAINBOW_APP_KEY || '',
        RAINBOW_REDIRECT_URI: d.RAINBOW_REDIRECT_URI || '',
        OAUTH_TYPES: oauthTypes,
      })
      smtpForm.setFieldsValue({
        SMTP_ENABLED: d.SMTP_ENABLED === 'true',
        SMTP_HOST: d.SMTP_HOST || '',
        SMTP_PORT: d.SMTP_PORT || '',
        SMTP_USERNAME: d.SMTP_USERNAME || '',
        SMTP_PASSWORD: d.SMTP_PASSWORD || '',
        SMTP_FROM: d.SMTP_FROM || '',
        SMTP_FROM_NAME: d.SMTP_FROM_NAME || '',
        SMTP_ENCRYPTION: d.SMTP_ENCRYPTION || 'starttls',
      })
      paymentForm.setFieldsValue({
        PAYMENT_ENABLED: d.PAYMENT_ENABLED === 'true',
        PAYMENT_PROVIDER: d.PAYMENT_PROVIDER || 'alipay',
        PAYMENT_MERCHANT_ID: d.PAYMENT_MERCHANT_ID || '',
        PAYMENT_APP_ID: d.PAYMENT_APP_ID || '',
        PAYMENT_APP_KEY: d.PAYMENT_APP_KEY || '',
        PAYMENT_NOTIFY_URL: d.PAYMENT_NOTIFY_URL || '',
      })
      snsForm.setFieldsValue({ sns_items: parseJSONArray(d.SNS_CONFIG) })
      footerForm.setFieldsValue({
        FOOTER_DESCRIPTION: d.FOOTER_DESCRIPTION || '',
        FOOTER_COPYRIGHT: d.FOOTER_COPYRIGHT || '',
        FOOTER_ICP: d.FOOTER_ICP || '',
        FOOTER_POLICE: d.FOOTER_POLICE || '',
        footer_links: parseJSONArray(d.FOOTER_LINKS),
      })
    }).catch(() => {})
    api.getSystemInfo().then(setSysInfo).catch(() => {})
    authApi.getOauthProviders().then((res) => {
      if (res?.enabled) setOauthProviders(res.providers || [])
    }).catch(() => {})
  }, [basicForm, accountForm, oauthForm, smtpForm, paymentForm, snsForm, footerForm])

  // Refresh the channel list whenever the active login service changes so the
  // "启用登录方式" checkboxes always match the selected provider.
  useEffect(() => {
    authApi.getOauthProviders().then((res) => {
      if (res?.enabled) setOauthProviders(res.providers || [])
    }).catch(() => {})
  }, [oauthProvider])

  const saveForm = async (key, form, transform) => {
    const values = await form.validateFields()
    setSaving(key)
    try {
      await api.updateSystemSettings(transform(values))
      notify('success', '设置已保存并生效')
    } catch {
      /* interceptor */
    } finally {
      setSaving('')
    }
  }

  const saveBasic = () => saveForm('basic', basicForm, (v) => v)
  const saveAccount = () => saveForm('account', accountForm, (v) => ({
    REGISTRATION_ENABLED: v.REGISTRATION_ENABLED ? 'true' : 'false',
    OAUTHGO_AUTO_REGISTER: v.OAUTHGO_AUTO_REGISTER ? 'true' : 'false',
  }))
  const saveOauth = () => saveForm('oauth', oauthForm, (v) => ({
    OAUTH_PROVIDER: v.OAUTH_PROVIDER || 'oauthgo',
    OAUTHGO_BASE_URL: v.OAUTHGO_BASE_URL || '',
    OAUTHGO_APP_ID: v.OAUTHGO_APP_ID || '',
    OAUTHGO_APP_KEY: v.OAUTHGO_APP_KEY || '',
    OAUTHGO_REDIRECT_URI: v.OAUTHGO_REDIRECT_URI || '',
    RAINBOW_BASE_URL: v.RAINBOW_BASE_URL || '',
    RAINBOW_APP_ID: v.RAINBOW_APP_ID || '',
    RAINBOW_APP_KEY: v.RAINBOW_APP_KEY || '',
    RAINBOW_REDIRECT_URI: v.RAINBOW_REDIRECT_URI || '',
    [(v.OAUTH_PROVIDER || 'oauthgo') === 'rainbow' ? 'RAINBOW_ENABLED_TYPES' : 'OAUTHGO_ENABLED_TYPES']: JSON.stringify(v.OAUTH_TYPES || []),
  }))
  const saveSMTP = () => saveForm('smtp', smtpForm, (v) => ({
    SMTP_ENABLED: v.SMTP_ENABLED ? 'true' : 'false',
    SMTP_HOST: v.SMTP_HOST || '',
    SMTP_PORT: v.SMTP_PORT || '',
    SMTP_USERNAME: v.SMTP_USERNAME || '',
    SMTP_PASSWORD: v.SMTP_PASSWORD || '',
    SMTP_FROM: v.SMTP_FROM || '',
    SMTP_FROM_NAME: v.SMTP_FROM_NAME || '',
    SMTP_ENCRYPTION: v.SMTP_ENCRYPTION || 'starttls',
  }))
  const savePayment = () => saveForm('payment', paymentForm, (v) => ({
    PAYMENT_ENABLED: v.PAYMENT_ENABLED ? 'true' : 'false',
    PAYMENT_PROVIDER: v.PAYMENT_PROVIDER || '',
    PAYMENT_MERCHANT_ID: v.PAYMENT_MERCHANT_ID || '',
    PAYMENT_APP_ID: v.PAYMENT_APP_ID || '',
    PAYMENT_APP_KEY: v.PAYMENT_APP_KEY || '',
    PAYMENT_NOTIFY_URL: v.PAYMENT_NOTIFY_URL || '',
  }))
  const saveSNS = () => saveForm('sns', snsForm, (v) => ({
    SNS_CONFIG: JSON.stringify(v.sns_items || []),
  }))
  const saveFooter = () => saveForm('footer', footerForm, (v) => ({
    FOOTER_DESCRIPTION: v.FOOTER_DESCRIPTION || '',
    FOOTER_COPYRIGHT: v.FOOTER_COPYRIGHT || '',
    FOOTER_ICP: v.FOOTER_ICP || '',
    FOOTER_POLICE: v.FOOTER_POLICE || '',
    FOOTER_LINKS: JSON.stringify(v.footer_links || []),
  }))

  const renderSaveBtn = (key) => (
    <Button type="primary" icon={<SaveOutlined />} loading={saving === key} onClick={() => {
      const map = { basic: saveBasic, account: saveAccount, oauth: saveOauth, smtp: saveSMTP, payment: savePayment, sns: saveSNS, footer: saveFooter }
      map[key]()
    }}>保存设置</Button>
  )

  const items = [
    {
      key: 'basic',
      label: '基础',
      children: (
        <Form form={basicForm} layout="vertical" requiredMark={false} style={{ maxWidth: 560 }}>
          <Form.Item name="WHOIS_API_URL" label="WHOIS API 地址"><Input placeholder="https://who.zmh.me" /></Form.Item>
          <Form.Item name="UA_WHOIS_SERVER" label="UA（乌克兰）WHOIS 服务器" extra="官方 dig.ua 的底层数据源，用于 *.pp.ua 等 .ua 域名查询"><Input placeholder="whois.ua:43" /></Form.Item>
          <Form.Item name="ICP_API_URL" label="ICP 备案 API 地址"><Input placeholder="http://127.0.0.1:16181" /></Form.Item>
          <Form.Item name="DIGITALPLAT_RDAP_URL" label="DigitalPlat RDAP 地址"><Input placeholder="https://rdap.digitalplat.org" /></Form.Item>
          <Form.Item name="PROXY_URL" label="HTTP 代理地址" extra="留空则直连；格式 http://user:pass@host:port 或 https://host:port。注册商勾选「使用代理」后，其 API 请求经此代理发送"><Input placeholder="http://127.0.0.1:7890" autoComplete="off" /></Form.Item>
          {renderSaveBtn('basic')}
        </Form>
      ),
    },
    {
      key: 'account',
      label: '账号',
      children: (
        <Form form={accountForm} layout="vertical" requiredMark={false} style={{ maxWidth: 560 }}>
          <Form.Item name="REGISTRATION_ENABLED" label="是否开启注册" valuePropName="checked" extra="关闭后隐藏注册入口，注册接口返回 403">
            <Switch />
          </Form.Item>
          <Form.Item name="OAUTHGO_AUTO_REGISTER" label="第三方登录自动注册" valuePropName="checked" extra="关闭后，未绑定的第三方账号需先登录本地账号，再到「个人设置 → 第三方登录」绑定">
            <Switch />
          </Form.Item>
          {renderSaveBtn('account')}
        </Form>
      ),
    },
    {
      key: 'oauthgo',
      label: '第三方登录',
      children: (
        <Form form={oauthForm} layout="vertical" requiredMark={false} style={{ maxWidth: 560 }}>
          <Form.Item name="OAUTH_PROVIDER" label="第三方登录服务" extra="选择接入 OauthGo 或 彩虹聚合登录（connect.php 协议）">
            <Select
              options={[
                { value: 'oauthgo', label: 'OauthGo' },
                { value: 'rainbow', label: '彩虹登录（彩虹聚合登录）' },
              ]}
            />
          </Form.Item>
          {oauthProvider === 'rainbow' ? (
            <>
              <Form.Item name="RAINBOW_BASE_URL" label="彩虹登录 服务地址" extra="留空则禁用第三方登录"><Input placeholder="https://u.cccyun.cc" /></Form.Item>
              <Form.Item name="RAINBOW_APP_ID" label="彩虹登录 应用 ID"><Input placeholder="应用 ID" /></Form.Item>
              <Form.Item name="RAINBOW_APP_KEY" label="彩虹登录 应用密钥"><Input.Password placeholder="应用密钥" autoComplete="new-password" /></Form.Item>
              <Form.Item name="RAINBOW_REDIRECT_URI" label="彩虹登录 回调地址" extra="留空则按请求自动推导"><Input placeholder="https://your-domain.com/api/auth/oauth/callback" /></Form.Item>
            </>
          ) : (
            <>
              <Form.Item name="OAUTHGO_BASE_URL" label="OauthGo 服务地址" extra="留空则禁用第三方登录"><Input placeholder="https://o.1v.fit" /></Form.Item>
              <Form.Item name="OAUTHGO_APP_ID" label="OauthGo 应用 ID"><Input placeholder="应用 ID" /></Form.Item>
              <Form.Item name="OAUTHGO_APP_KEY" label="OauthGo 应用密钥"><Input.Password placeholder="应用密钥" autoComplete="new-password" /></Form.Item>
              <Form.Item name="OAUTHGO_REDIRECT_URI" label="OauthGo 回调地址" extra="留空则按请求自动推导"><Input placeholder="https://your-domain.com/api/auth/oauth/callback" /></Form.Item>
            </>
          )}
          <Form.Item name="OAUTH_TYPES" label="启用登录方式" extra="不勾选则展示所选服务已开启的全部渠道">
            <Checkbox.Group
              options={oauthProviders.map((p) => ({ label: p.display_name || p.name, value: p.name }))}
            />
          </Form.Item>
          {renderSaveBtn('oauth')}
        </Form>
      ),
    },
    {
      key: 'smtp',
      label: 'SMTP',
      children: (
        <Form form={smtpForm} layout="vertical" requiredMark={false} style={{ maxWidth: 560 }}>
          <Form.Item name="SMTP_ENABLED" label="启用 SMTP" valuePropName="checked"><Switch /></Form.Item>
          <Form.Item name="SMTP_HOST" label="SMTP 服务器"><Input placeholder="smtp.example.com" /></Form.Item>
          <Form.Item name="SMTP_PORT" label="端口"><Input placeholder="465 / 587" /></Form.Item>
          <Form.Item name="SMTP_ENCRYPTION" label="加密方式"><Select options={SMTP_ENCRYPTIONS} /></Form.Item>
          <Form.Item name="SMTP_USERNAME" label="用户名"><Input placeholder="发信邮箱账号" autoComplete="off" /></Form.Item>
          <Form.Item name="SMTP_PASSWORD" label="密码 / 授权码"><Input.Password placeholder="密码或授权码" autoComplete="new-password" /></Form.Item>
          <Form.Item name="SMTP_FROM" label="发件人地址"><Input placeholder="noreply@example.com" /></Form.Item>
          <Form.Item name="SMTP_FROM_NAME" label="发件人名称"><Input placeholder="Domain Manager" /></Form.Item>
          <p className="muted" style={{ fontSize: 12, lineHeight: 1.7, maxWidth: 560 }}>
            作为邮件通知渠道的默认 SMTP：当通知渠道未单独填写 SMTP 服务器时自动使用此处配置。
          </p>
          {renderSaveBtn('smtp')}
        </Form>
      ),
    },
    {
      key: 'payment',
      label: '支付系统',
      children: (
        <Form form={paymentForm} layout="vertical" requiredMark={false} style={{ maxWidth: 560 }}>
          <Form.Item name="PAYMENT_ENABLED" label="启用支付" valuePropName="checked"><Switch /></Form.Item>
          <Form.Item name="PAYMENT_PROVIDER" label="支付渠道"><Select options={PAYMENT_PROVIDERS} /></Form.Item>
          <Form.Item name="PAYMENT_MERCHANT_ID" label="商户号"><Input placeholder="商户号 / PID" /></Form.Item>
          <Form.Item name="PAYMENT_APP_ID" label="应用 ID"><Input placeholder="应用 / App ID" /></Form.Item>
          <Form.Item name="PAYMENT_APP_KEY" label="应用密钥"><Input.Password placeholder="应用密钥 / Key" autoComplete="new-password" /></Form.Item>
          <Form.Item name="PAYMENT_NOTIFY_URL" label="异步通知地址"><Input placeholder="https://your-domain.com/api/payment/notify" /></Form.Item>
          {renderSaveBtn('payment')}
        </Form>
      ),
    },
    {
      key: 'sns',
      label: 'SNS',
      children: (
        <Form form={snsForm} layout="vertical" requiredMark={false} style={{ maxWidth: 560 }}>
          <Form.Item label="社交平台链接" style={{ marginBottom: 8 }}>
            <span className="muted" style={{ fontSize: 12 }}>展示在登录页页脚（如 GitHub、Telegram、微博等）</span>
          </Form.Item>
          <Form.List name="sns_items">
            {(fields, { add, remove }) => (
              <>
                {fields.map(({ key, name, ...rest }) => (
                  <Space key={key} align="baseline" style={{ display: 'flex', marginBottom: 8 }}>
                    <Form.Item {...rest} name={[name, 'label']} rules={[{ required: true, message: '名称必填' }]} style={{ marginBottom: 0 }}>
                      <Input placeholder="名称（如 GitHub）" style={{ width: 160 }} />
                    </Form.Item>
                    <Form.Item {...rest} name={[name, 'url']} rules={[{ required: true, message: '链接必填' }]} style={{ marginBottom: 0 }}>
                      <Input placeholder="https://..." style={{ width: 280 }} />
                    </Form.Item>
                    <MinusCircleOutlined onClick={() => remove(name)} style={{ color: '#a8a29e' }} />
                  </Space>
                ))}
                <Button type="dashed" block icon={<PlusOutlined />} onClick={() => add({ label: '', url: '' })} style={{ marginBottom: 16 }}>添加平台</Button>
              </>
            )}
          </Form.List>
          {renderSaveBtn('sns')}
        </Form>
      ),
    },
    {
      key: 'footer',
      label: '页脚',
      children: (
        <Form form={footerForm} layout="vertical" requiredMark={false} style={{ maxWidth: 560 }}>
          <Form.Item name="FOOTER_DESCRIPTION" label="站点描述"><Input placeholder="一句话描述" /></Form.Item>
          <Form.Item name="FOOTER_COPYRIGHT" label="版权信息"><Input placeholder="© 2026 Domain Manager" /></Form.Item>
          <Form.Item name="FOOTER_ICP" label="ICP 备案号"><Input placeholder="如：京ICP备00000000号" /></Form.Item>
          <Form.Item name="FOOTER_POLICE" label="公安备案号"><Input placeholder="如：京公网安备11000000000000号" /></Form.Item>
          <Form.Item label="页脚链接" style={{ marginBottom: 8 }}>
            <span className="muted" style={{ fontSize: 12 }}>展示在登录页页脚</span>
          </Form.Item>
          <Form.List name="footer_links">
            {(fields, { add, remove }) => (
              <>
                {fields.map(({ key, name, ...rest }) => (
                  <Space key={key} align="baseline" style={{ display: 'flex', marginBottom: 8 }}>
                    <Form.Item {...rest} name={[name, 'label']} rules={[{ required: true, message: '名称必填' }]} style={{ marginBottom: 0 }}>
                      <Input placeholder="名称（如 关于我们）" style={{ width: 160 }} />
                    </Form.Item>
                    <Form.Item {...rest} name={[name, 'url']} rules={[{ required: true, message: '链接必填' }]} style={{ marginBottom: 0 }}>
                      <Input placeholder="https://..." style={{ width: 280 }} />
                    </Form.Item>
                    <MinusCircleOutlined onClick={() => remove(name)} style={{ color: '#a8a29e' }} />
                  </Space>
                ))}
                <Button type="dashed" block icon={<PlusOutlined />} onClick={() => add({ label: '', url: '' })} style={{ marginBottom: 16 }}>添加链接</Button>
              </>
            )}
          </Form.List>
          {renderSaveBtn('footer')}
        </Form>
      ),
    },
  ]

  return (
    <div className="page" style={{ maxWidth: 980 }}>
      <PageHead title="系统设置" sub="运行时配置保存在数据库中，保存后立即生效（仅管理员可用）" />

      <div className="panel">
        <div className="panel-head"><h3 className="panel-title">配置</h3></div>
        <div style={{ padding: '0 20px 20px' }}>
          <Tabs items={items} tabPosition={isMobile ? 'top' : 'left'} />
        </div>
      </div>

      <div className="panel mt-16">
        <div className="panel-head"><h3 className="panel-title">系统信息</h3></div>
        <div style={{ padding: 20 }}>
          {sysInfo ? (
            <Descriptions column={isMobile ? 1 : 4} size="small" bordered>
              <Descriptions.Item label="域名总数">{sysInfo.domains}</Descriptions.Item>
              <Descriptions.Item label="证书总数">{sysInfo.certificates}</Descriptions.Item>
              <Descriptions.Item label="注册商数量">{sysInfo.registrars}</Descriptions.Item>
              <Descriptions.Item label="用户数量">{sysInfo.users}</Descriptions.Item>
              <Descriptions.Item label="系统版本">{sysInfo.version}</Descriptions.Item>
            </Descriptions>
          ) : null}
        </div>
      </div>
    </div>
  )
}
