import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as loginApi, register as registerApi, getProfile } from '../api/auth'

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref(null)

  const isLoggedIn = computed(() => !!token.value)

  async function login(form) {
    const res = await loginApi(form)
    token.value = res.token
    user.value = res.user
    localStorage.setItem('token', res.token)
    return res
  }

  async function register(form) {
    const res = await registerApi(form)
    token.value = res.token
    user.value = res.user
    localStorage.setItem('token', res.token)
    return res
  }

  async function fetchProfile() {
    if (!token.value) return
    try {
      const res = await getProfile()
      user.value = res
    } catch {
      logout()
    }
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
  }

  return { token, user, isLoggedIn, login, register, fetchProfile, logout }
})
