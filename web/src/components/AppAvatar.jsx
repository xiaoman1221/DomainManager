import { useState } from 'react'
import { Avatar } from 'antd'
import { resolveAvatar, avatarInitial } from '../utils/avatar'

export default function AppAvatar({ email, name, size = 32 }) {
  const [failed, setFailed] = useState(false)
  const resolved = failed ? null : resolveAvatar(email)
  const initial = avatarInitial(name, email)
  return (
    <Avatar
      size={size}
      src={resolved?.url}
      onError={() => {
        setFailed(true)
        return true
      }}
      style={{ background: '#18181b', color: '#ffffff', fontWeight: 600, flexShrink: 0 }}
    >
      {initial}
    </Avatar>
  )
}
