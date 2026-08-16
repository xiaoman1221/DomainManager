import { createContext, useContext, useEffect, useState, useCallback } from 'react'
import * as authApi from '../api/auth'

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [token, setToken] = useState(() => localStorage.getItem('token') || '')
  const [user, setUser] = useState(null)

  const logout = useCallback(() => {
    setToken('')
    setUser(null)
    localStorage.removeItem('token')
  }, [])

  const login = async (form) => {
    const res = await authApi.login(form)
    setToken(res.token)
    setUser(res.user)
    localStorage.setItem('token', res.token)
    return res
  }

  const register = async (form) => {
    const res = await authApi.register(form)
    setToken(res.token)
    setUser(res.user)
    localStorage.setItem('token', res.token)
    return res
  }

  const fetchProfile = useCallback(async () => {
    if (!token) return
    try {
      const res = await authApi.getProfile()
      setUser(res)
    } catch {
      logout()
    }
  }, [token, logout])

  useEffect(() => {
    if (token) fetchProfile()
  }, [token, fetchProfile])

  return (
    <AuthContext.Provider value={{ token, user, login, register, logout, fetchProfile }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
