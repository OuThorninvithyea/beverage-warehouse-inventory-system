import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import * as authApi from '@/api/auth'
import { setApiAccessToken } from '@/api/client'

const REFRESH_TOKEN_KEY = 'bwims.refresh_token'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<authApi.SessionUser | null>(null)
  const initialized = ref(false)
  const loading = ref(false)
  const isAuthenticated = computed(() => user.value !== null)

  function acceptSession(session: authApi.TokenPair) {
    setApiAccessToken(session.access_token)
    sessionStorage.setItem(REFRESH_TOKEN_KEY, session.refresh_token)
    user.value = session.user
  }

  function clearSession() {
    setApiAccessToken('')
    sessionStorage.removeItem(REFRESH_TOKEN_KEY)
    user.value = null
  }

  async function signIn(email: string, password: string) {
    loading.value = true
    try {
      acceptSession(await authApi.login(email, password))
    } finally {
      loading.value = false
      initialized.value = true
    }
  }

  async function restoreSession() {
    if (initialized.value) {
      return
    }

    const refreshToken = sessionStorage.getItem(REFRESH_TOKEN_KEY)
    if (!refreshToken) {
      initialized.value = true
      return
    }

    try {
      acceptSession(await authApi.refresh(refreshToken))
    } catch {
      clearSession()
    } finally {
      initialized.value = true
    }
  }

  async function signOut() {
    const refreshToken = sessionStorage.getItem(REFRESH_TOKEN_KEY)
    try {
      if (refreshToken) {
        await authApi.logout(refreshToken)
      }
    } finally {
      clearSession()
    }
  }

  return {
    user,
    initialized,
    loading,
    isAuthenticated,
    signIn,
    signOut,
    restoreSession,
  }
})
