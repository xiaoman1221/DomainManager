import React, { useEffect } from 'react'
import ReactDOM from 'react-dom/client'
import { ConfigProvider, App as AntApp } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import { BrowserRouter } from 'react-router-dom'
import App from './App'
import { AuthProvider } from './context/AuthContext'
import './styles/global.css'

dayjs.locale('zh-cn')

// Global toast bridge: lets non-component code (axios interceptor) show toasts.
function ToastBridge() {
  const { message } = AntApp.useApp()
  useEffect(() => {
    const handler = (e) => {
      const { type, content } = e.detail || {}
      if (type && content && message[type]) message[type](content)
    }
    window.addEventListener('app:toast', handler)
    return () => window.removeEventListener('app:toast', handler)
  }, [message])
  return null
}

const theme = {
  token: {
    colorPrimary: '#18181b',
    colorInfo: '#18181b',
    colorLink: '#2563eb',
    colorTextBase: '#1c1917',
    colorText: '#1c1917',
    colorTextSecondary: '#57534e',
    colorTextTertiary: '#a8a29e',
    colorBgLayout: '#fafaf9',
    colorBorder: '#e7e5e4',
    colorBorderSecondary: '#f0efee',
    borderRadius: 6,
    fontSize: 14,
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'Helvetica Neue', Arial, sans-serif",
    controlHeight: 34,
  },
  components: {
    Layout: { headerBg: 'transparent', siderBg: '#ffffff', bodyBg: '#fafaf9' },
    Menu: {
      itemSelectedBg: '#18181b',
      itemSelectedColor: '#ffffff',
      itemHeight: 38,
      itemBorderRadius: 6,
      activeBarBorderWidth: 0,
    },
    Table: {
      headerBg: '#fafaf9',
      headerColor: '#57534e',
      headerSplitColor: 'transparent',
      borderColor: '#ececeb',
      rowHoverBg: '#fafaf9',
      cellPaddingBlock: 12,
    },
    Button: { fontWeight: 500, primaryShadow: 'none' },
    Card: { headerBg: 'transparent' },
    Modal: { titleFontSize: 17 },
    Tabs: { inkBarColor: '#18181b', itemSelectedColor: '#1c1917' },
    Drawer: { paddingLG: 24 },
  },
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <ConfigProvider locale={zhCN} theme={theme}>
      <AntApp>
        <ToastBridge />
        <BrowserRouter>
          <AuthProvider>
            <App />
          </AuthProvider>
        </BrowserRouter>
      </AntApp>
    </ConfigProvider>
  </React.StrictMode>
)
