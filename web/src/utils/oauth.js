// Provider display marks shared by the login dialog and the profile page.
export const PROVIDER_MARKS = {
  qq: 'QQ',
  wx: '微',
  wechat: '微',
  alipay: '支',
  sina: '博',
  weibo: '博',
  baidu: '百',
  douyin: '抖',
  dingtalk: '钉',
  gitee: 'G',
  github: 'GH',
  google: 'G',
  microsoft: 'M',
  discord: 'D',
  facebook: 'F',
  wework: '企',
  wangzhan: '站',
}

export function providerMark(provider) {
  return PROVIDER_MARKS[provider.name] || String(provider.display_name || provider.name || '?').trim().charAt(0)
}

export function providerLabel(name, fallback) {
  return PROVIDER_LABELS[name] || fallback || name
}

export const PROVIDER_LABELS = {
  qq: 'QQ',
  wx: '微信',
  wechat: '微信',
  alipay: '支付宝',
  sina: '微博',
  weibo: '微博',
  baidu: '百度',
  douyin: '抖音',
  dingtalk: '钉钉',
  gitee: 'Gitee',
  github: 'GitHub',
  google: 'Google',
  microsoft: 'Microsoft',
  discord: 'Discord',
  facebook: 'Facebook',
  wework: '企业微信',
}
