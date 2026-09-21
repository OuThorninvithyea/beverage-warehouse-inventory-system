import { defineStore } from 'pinia'
import { ref } from 'vue'

import {
  type CreateUserInput,
  type UpdateUserInput,
  type User,
  type UserListFilter,
  createUser,
  deactivateUser,
  getUser,
  listUsers,
  resetPassword,
  updateUser,
} from '@/api/users'

export const useUsersStore = defineStore('users', () => {
  const users = ref<User[]>([])
  const currentUser = ref<User | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const hasMore = ref(false)

  async function fetchUsers(filter: UserListFilter = {}) {
    loading.value = true
    error.value = null
    try {
      const page = await listUsers(filter)
      users.value = page.items
      hasMore.value = page.page.has_more
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch users'
    } finally {
      loading.value = false
    }
  }

  async function fetchUser(id: string) {
    loading.value = true
    error.value = null
    try {
      const user = await getUser(id)
      currentUser.value = user
      return user
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch user'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function addUser(input: CreateUserInput) {
    loading.value = true
    error.value = null
    try {
      const user = await createUser(input)
      users.value.unshift(user)
      return user
    } catch (e: any) {
      error.value = e.message || 'Failed to create user'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function editUser(id: string, input: UpdateUserInput) {
    loading.value = true
    error.value = null
    try {
      const updated = await updateUser(id, input)
      const idx = users.value.findIndex((u) => u.id === id)
      if (idx !== -1) {
        users.value[idx] = updated
      }
      return updated
    } catch (e: any) {
      error.value = e.message || 'Failed to update user'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function removeUser(id: string) {
    loading.value = true
    error.value = null
    try {
      await deactivateUser(id)
      const idx = users.value.findIndex((u) => u.id === id)
      if (idx !== -1) {
        users.value[idx].is_active = false
      }
    } catch (e: any) {
      error.value = e.message || 'Failed to deactivate user'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function doResetPassword(id: string, newPassword: string) {
    loading.value = true
    error.value = null
    try {
      await resetPassword(id, newPassword)
    } catch (e: any) {
      error.value = e.message || 'Password reset failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    users,
    currentUser,
    loading,
    error,
    hasMore,
    fetchUsers,
    fetchUser,
    addUser,
    editUser,
    removeUser,
    doResetPassword,
  }
})
