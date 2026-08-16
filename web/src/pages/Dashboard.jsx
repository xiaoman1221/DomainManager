import { useEffect, useMemo, useState } from 'react'
import { Table, Tag, Empty } from 'antd'
import { PlusOutlined, DollarOutlined, RightOutlined, WarningOutlined } from '@ant-design/icons'
import { Link, useNavigate } from 'react-router-dom'
import dayjs from 'dayjs'
import { getDomainStats, getDomains } from '../api/domain'
import { useAuth } from '../context/AuthContext'
import { fmtDate, calcDays, daysColor, daysLabel, domainStatusInfo } from '../utils/format'

const STATUS_TAG_COLOR = {
  success: 'success',
  error: 'error',
  warning: 'warning',
  default: 'default',
}

export default function Dashboard() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [stats, setStats] = useState({ total: 0, active: 0, expiring_soon: 0 })
  const [recent, setRecent] = useState([])
  const [expiring, setExpiring] = useState([])

  useEffect(() => {
    Promise.all([
      getDomainStats(),
      getDomains({ page: 1, page_size: 8 }),
      getDomains({ status: 'expiring_30', page: 1, page_size: 5 }),
    ])
      .then(([s, r, e]) => {
        setStats(s)
        setRecent(r.data || [])
        setExpiring(e.data || [])
      })
      .catch(() => {})
  }, [])

  const greeting = useMemo(() => {
    const h = dayjs().hour()
    const part = h < 6 ? '夜深了' : h < 12 ? '早上好' : h < 18 ? '下午好' : '晚上好'
    const name = user?.nickname || user?.username || ''
    return `${part}${name ? '，' + name : ''}`
  }, [user])

  const today = dayjs().format('YYYY年M月D日 dddd')

  const columns = [
    { title: '域名', dataIndex: 'name', key: 'name', render: (v, r) => <span className="domain-link" onClick={() => navigate('/domains')}>{v}</span> },
    { title: '注册商', dataIndex: 'registrar', key: 'registrar', render: (v) => v || <span className="faint">-</span>, width: 130 },
    {
      title: '到期时间',
      dataIndex: 'expiry_date',
      key: 'expiry_date',
      render: (v) => (v ? <span style={{ color: daysColor(calcDays(v)) }}>{fmtDate(v)}</span> : <span className="faint">-</span>),
      width: 120,
    },
    {
      title: '剩余',
      dataIndex: 'expiry_date',
      key: 'days',
      render: (v) => {
        const days = calcDays(v)
        return <span style={{ color: daysColor(days), fontWeight: 500 }}>{daysLabel(days)}</span>
      },
      width: 70,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (_, r) => {
        const s = domainStatusInfo(r)
        return <Tag color={STATUS_TAG_COLOR[s.color]} style={{ borderRadius: 4, fontSize: 12 }}>{s.text}</Tag>
      },
      width: 90,
    },
  ]

  return (
    <div className="page">
      <h1 className="greeting">{greeting}</h1>
      <p className="greeting-date">{today}</p>

      <div className="stats-band">
        <div className="stat-block main">
          <div className="stat-label">域名总数</div>
          <div className="stat-value">{stats.total}<small>个</small></div>
          <div className="stat-hint">全部纳入管理的域名资产</div>
        </div>
        <div className="stat-block sub">
          <div className="stat-label">正常</div>
          <div className="stat-value">{stats.active}<small>个</small></div>
          <div className="stat-hint">状态正常且未过期</div>
        </div>
        <div className="stat-block sub">
          <div className="stat-label">30 天内到期</div>
          <div className="stat-value" style={{ color: stats.expiring_soon > 0 ? '#d97706' : undefined }}>{stats.expiring_soon}<small>个</small></div>
          <div className="stat-hint">需要关注续费</div>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1.8fr 1fr', gap: 20 }}>
        <div className="panel">
          <div className="panel-head">
            <h3 className="panel-title">最近添加的域名</h3>
            <Link to="/domains" style={{ fontSize: 13, color: '#57534e' }}>查看全部 <RightOutlined style={{ fontSize: 10 }} /></Link>
          </div>
          <Table
            rowKey="id"
            columns={columns}
            dataSource={recent}
            pagination={false}
            size="middle"
            locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无域名" /> }}
            scroll={{ x: 560 }}
          />
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
          <div className="panel">
            <div className="panel-head"><h3 className="panel-title">快捷操作</h3></div>
            <div className="panel-body" style={{ paddingTop: 8, paddingBottom: 8 }}>
              <Link to="/domains" className="quick-action">
                <span>
                  <div className="qa-label"><PlusOutlined style={{ marginRight: 8, fontSize: 13 }} />添加域名</div>
                  <div className="qa-desc">手动登记或 CSV 批量导入</div>
                </span>
                <RightOutlined style={{ color: '#a8a29e', fontSize: 11 }} />
              </Link>
              <Link to="/price" className="quick-action">
                <span>
                  <div className="qa-label"><DollarOutlined style={{ marginRight: 8, fontSize: 13 }} />域名比价</div>
                  <div className="qa-desc">对比各注册商参考价</div>
                </span>
                <RightOutlined style={{ color: '#a8a29e', fontSize: 11 }} />
              </Link>
              <Link to="/registrars" className="quick-action">
                <span>
                  <div className="qa-label"><RightOutlined style={{ marginRight: 8, fontSize: 13 }} />注册商同步</div>
                  <div className="qa-desc">从注册商 API 自动导入</div>
                </span>
                <RightOutlined style={{ color: '#a8a29e', fontSize: 11 }} />
              </Link>
            </div>
          </div>

          <div className="panel">
            <div className="panel-head">
              <h3 className="panel-title" style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <WarningOutlined style={{ color: stats.expiring_soon > 0 ? '#d97706' : '#a8a29e' }} />即将到期
              </h3>
              <span className="faint" style={{ fontSize: 12 }}>30 天内</span>
            </div>
            <div className="panel-body">
              {expiring.length === 0 ? (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="近期没有到期域名" />
              ) : (
                <div className="mini-list">
                  {expiring.map((d) => {
                    const days = calcDays(d.expiry_date)
                    return (
                      <div key={d.id} className="mini-list-item">
                        <span style={{ fontWeight: 500 }}>{d.name}</span>
                        <span style={{ color: daysColor(days), fontWeight: 600, fontSize: 12 }}>
                          {fmtDate(d.expiry_date)} · {daysLabel(days)}
                        </span>
                      </div>
                    )
                  })}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
