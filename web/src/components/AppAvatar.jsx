import { useState } from 'react'
import { Avatar } from 'antd'
import { resolveAvatar, avatarInitial } from '../utils/avatar'

export default function AppAvatar({ email, name, size = 32, avatar }) {
  const [failed, setFailed] = useState(false)
  // An explicit avatar URL (e.g. third-party provider avatar) takes priority;
  // otherwise fall back to QQ/Gravatar resolution by email.
  const resolved = failed ? null : (avatar || resolveAvatar(email)?.url)
  const initial = avatarInitial(name, email)
  return (
    <Avatar
      size={size}
      src={resolved}
      onError={() => {
        setFailed(true)
        return true
      }}
      style={{ background: '#e7e5e4', color: '#1c1917', fontWeight: 600, flexShrink: 0 }}
    >
      {initial}
    </Avatar>
  )
}
