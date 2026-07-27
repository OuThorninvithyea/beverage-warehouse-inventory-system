import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

export interface SessionUser {
  id: string
  fullName: string
  role: string
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<SessionUser | null>(null)
  const isAuthenticated = computed(() => user.value !== null)

  function setUser(nextUser: SessionUser | null) {
    user.value = nextUser
  }

  return { user, isAuthenticated, setUser }
})
