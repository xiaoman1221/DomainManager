import { Tag } from 'antd'
import { domainStatusInfo } from '../utils/format'

// Unified domain status tag (正常 / 已过期 / 暂停 ...) based on status + WHOIS + expiry.
export default function StatusTag({ data }) {
  const info = domainStatusInfo(data)
  return <Tag color={info.color} style={{ borderRadius: 4 }}>{info.text}</Tag>
}
