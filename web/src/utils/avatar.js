import md5 from 'md5'

// QQ avatar: https://q1.qlogo.cn/g?b=qq&nk=<QQ号>&s=<size>
// Gravatar: MD5(lowercased, trimmed email)
export function resolveAvatar(email) {
  const e = String(email || '').trim().toLowerCase()
  if (!e) return null
  const qq = e.match(/^(\d{5,})@qq\.com$/)
  if (qq) return { type: 'qq', url: `https://q1.qlogo.cn/g?b=qq&nk=${qq[1]}&s=640` }
  return { type: 'gravatar', url: `https://www.gravatar.com/avatar/${md5(e)}?d=retro&s=256` }
}

export function avatarInitial(name, email) {
  return String(name || email || '?').trim().charAt(0).toUpperCase()
}
