import { beforeEach, describe, expect, it, vi } from 'vitest'

import {
  createUser,
  deactivateUser,
  getUser,
  listUsers,
  resetPassword,
  updateUser,
} from '@/api/users'

vi.mock('@/api/client', () => ({
  apiRequest: vi.fn(),
}))

import { apiRequest } from '@/api/client'

const mockApiRequest = apiRequest as unknown as ReturnType<typeof vi.fn>

describe('users API client', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  it('listUsers passes search query and filters', async () => {
    mockApiRequest.mockResolvedValueOnce({ items: [], page: { next_cursor: null, has_more: false } })
    await listUsers({ search: 'john', role: 'admin' })
    expect(mockApiRequest).toHaveBeenCalledWith('/users?search=john&role=admin')
  })

  it('getUser calls user by id', async () => {
    mockApiRequest.mockResolvedValueOnce({ id: 'u-1' })
    await getUser('u-1')
    expect(mockApiRequest).toHaveBeenCalledWith('/users/u-1')
  })

  it('createUser posts create payload', async () => {
    mockApiRequest.mockResolvedValueOnce({ id: 'u-1' })
    const payload = {
      email: 'user@bwims.test',
      full_name: 'Test User',
      role: 'picker' as const,
      password: 'password12345',
    }
    await createUser(payload)
    expect(mockApiRequest).toHaveBeenCalledWith('/users', {
      method: 'POST',
      body: JSON.stringify(payload),
    })
  })

  it('updateUser puts update payload', async () => {
    mockApiRequest.mockResolvedValueOnce({ id: 'u-1' })
    const payload = {
      full_name: 'Updated Name',
      role: 'warehouse_manager' as const,
    }
    await updateUser('u-1', payload)
    expect(mockApiRequest).toHaveBeenCalledWith('/users/u-1', {
      method: 'PUT',
      body: JSON.stringify(payload),
    })
  })

  it('deactivateUser calls DELETE', async () => {
    mockApiRequest.mockResolvedValueOnce(undefined)
    await deactivateUser('u-1')
    expect(mockApiRequest).toHaveBeenCalledWith('/users/u-1', { method: 'DELETE' })
  })

  it('resetPassword posts password reset payload', async () => {
    mockApiRequest.mockResolvedValueOnce(undefined)
    await resetPassword('u-1', 'newpassword123')
    expect(mockApiRequest).toHaveBeenCalledWith('/users/u-1/password-reset', {
      method: 'POST',
      body: JSON.stringify({ password: 'newpassword123' }),
    })
  })
})
