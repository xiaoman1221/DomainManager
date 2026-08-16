import { useEffect, useRef, useState } from 'react'
import { Table, Input, Button, Tag, Alert } from 'antd'
import { SearchOutlined, ExportOutlined } from '@ant-design/icons'
import { useSearchParams } from 'react-router-dom'
import { comparePrices, getSupportedTLDs } from '../api/price'
import { notify } from '../utils/toast'

function fmt(v, cur) {
  return `${cur === 'CNY' ? '¥' : '$'}${Number(v).toFixed(2)}`
}

export default function Price() {
  const [params] = useSearchParams()
  const [domain, setDomain] = useState('')
  const [prices, setPrices] = useState([])
  const [loading, setLoading] = useState(false)
  const [tlds, setTlds] = useState([])

  const handleCompare = async (value) => {
    const d = String(value || domain).trim()
    if (!d) {
      notify('warning', '请输入域名')
      return
    }
    setLoading(true)
    try {
      const res = await comparePrices({ domain: d })
      setPrices(res.prices || [])
      if (!res.prices || res.prices.length === 0) notify('info', '暂无该域名的比价数据')
    } catch {
      /* interceptor */
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    getSupportedTLDs().then((res) => setTlds(res.tlds || [])).catch(() => {})
  }, [])

  const comparedRef = useRef(null)
  useEffect(() => {
    const d = params.get('domain')
    if (d && comparedRef.current !== d) {
      comparedRef.current = d
      setDomain(d)
      handleCompare(d)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [params])

  const isCheapest = (row) => {
    const same = prices.filter((p) => p.currency === row.currency)
    if (same.length <= 1) return false
    const min = Math.min(...same.map((p) => Number(p.register_price)))
    return Number(row.register_price) === min
  }

  const columns = [
    {
      title: '注册商',
      dataIndex: 'registrar',
      key: 'registrar',
      width: 200,
      render: (v, r) => (
        <span>
          <span style={{ fontWeight: 500 }}>{v}</span>
          {r.reference && <Tag style={{ marginLeft: 8, borderRadius: 4 }}>参考</Tag>}
        </span>
      ),
    },
    { title: '后缀', dataIndex: 'tld', key: 'tld', width: 90, render: (v) => <Tag style={{ borderRadius: 4 }}>.{v}</Tag> },
    { title: '注册价', dataIndex: 'register_price', key: 'register_price', align: 'right', width: 140, render: (v, r) => <span className="price-num" style={{ color: '#dc2626' }}>{fmt(v, r.currency)}</span> },
    { title: '续费价', dataIndex: 'renew_price', key: 'renew_price', align: 'right', width: 140, render: (v, r) => <span className="price-num">{fmt(v, r.currency)}</span> },
    { title: '转入价', dataIndex: 'transfer_price', key: 'transfer_price', align: 'right', width: 140, render: (v, r) => <span className="price-num">{fmt(v, r.currency)}</span> },
    {
      title: '',
      key: 'cheapest',
      width: 90,
      align: 'center',
      render: (_, r) => (isCheapest(r) ? <Tag color="success" style={{ borderRadius: 4 }}>最低价</Tag> : null),
    },
    {
      title: '',
      key: 'action',
      width: 110,
      align: 'right',
      render: (_, r) => r.url && <Button type="link" size="small" href={r.url} target="_blank" rel="noreferrer">前往购买 <ExportOutlined /></Button>,
    },
  ]

  return (
    <div className="page">
      <div className="page-head">
        <div>
          <h1 className="page-title">域名比价</h1>
          <p className="page-sub">对比各注册商的注册 / 续费 / 转入价格</p>
        </div>
      </div>

      <div className="panel mb-16">
        <div className="panel-body" style={{ display: 'flex', gap: 12 }}>
          <Input
            size="large"
            placeholder="请输入域名，如 example.com"
            value={domain}
            onChange={(e) => setDomain(e.target.value)}
            onPressEnter={() => handleCompare()}
            style={{ maxWidth: 420 }}
          />
          <Button size="large" type="primary" icon={<SearchOutlined />} loading={loading} onClick={() => handleCompare()}>
            查询比价
          </Button>
        </div>
      </div>

      {prices.length > 0 && (
        <Alert
          type="warning"
          showIcon
          closable={false}
          style={{ marginBottom: 16, border: '1px solid #fde68a', background: '#fffbeb' }}
          message="当前价格为估算参考价（非实时注册商报价），仅用于比价参考。"
        />
      )}

      <div className="panel">
        <Table rowKey={(r) => `${r.registrar}-${r.tld}`} columns={columns} dataSource={prices} loading={loading} pagination={false} size="middle" locale={{ emptyText: '请输入域名进行比价' }} />
      </div>

      <div className="panel mt-16">
        <div className="panel-head"><h3 className="panel-title">支持的域名后缀</h3></div>
        <div className="panel-body">
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
            {tlds.map((t) => (
              <Tag
                key={t}
                style={{ borderRadius: 4, cursor: 'pointer', padding: '4px 10px' }}
                onClick={() => {
                  const head = String(domain || '').split('.')[0]
                  const next = head ? `${head}.${t}` : `.${t}`
                  setDomain(next)
                }}
              >
                .{t}
              </Tag>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
