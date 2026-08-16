import axios from 'axios'
import { notify } from '../utils/toast'

const request = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

request.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

request.interceptors.response.use(
  (response) => response.data,
  (error) => {
    const status = error.response?.status
    const msg = error.response?.data?.error || '请求失败，请稍后重试'
    if (status === 401 && !window.location.pathname.startsWith('/login')) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    } else {
      notify('error', msg)
    }
    return Promise.reject(error)
  }
)

export default request
