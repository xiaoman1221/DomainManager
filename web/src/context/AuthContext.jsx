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

  // Stores an AuthResponse {token, user} as the current session.
  const applyAuth = useCallback((res) => {
    setToken(res.token)
    setUser(res.user)
    localStorage.setItem('token', res.token)
    return res
  }, [])

  const login = async (form) => applyAuth(await authApi.login(form))

  const register = async (form) => applyAuth(await authApi.register(form))

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
    <AuthContext.Provider value={{ token, user, login, register, applyAuth, logout, fetchProfile }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
