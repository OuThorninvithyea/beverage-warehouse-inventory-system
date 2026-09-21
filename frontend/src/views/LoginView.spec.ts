import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const { push, signIn } = vi.hoisted(() => ({
  push: vi.fn(),
  signIn: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
  useRouter: () => ({ push }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ loading: false, signIn }),
}))

async function mountDemoLogin() {
  vi.stubEnv('VITE_DEMO_MODE', 'true')
  vi.resetModules()
  const { default: LoginView } = await import('@/views/LoginView.vue')
  return mount(LoginView)
}

beforeEach(() => {
  vi.clearAllMocks()
  signIn.mockResolvedValue(undefined)
})

afterEach(() => {
  vi.unstubAllEnvs()
})

describe('LoginView demo login', () => {
  it('uses the normal auth store for a selected seeded demo account', async () => {
    const wrapper = await mountDemoLogin()
    const managerButton = wrapper.findAll('button').find((button) => button.text() === 'Warehouse Manager')

    expect(wrapper.text()).toContain('Quick Demo Login')
    expect(managerButton).toBeDefined()

    await managerButton!.trigger('click')

    expect(signIn).toHaveBeenCalledWith('manager@bwims.local', 'DemoPass123!')
    expect(push).toHaveBeenCalledWith('/')
  })

  it('keeps the email and password form on the same authentication flow', async () => {
    const wrapper = await mountDemoLogin()

    await wrapper.get('#email').setValue('person@example.test')
    await wrapper.get('#password').setValue('a-password')
    await wrapper.get('form').trigger('submit')

    expect(signIn).toHaveBeenCalledWith('person@example.test', 'a-password')
    expect(push).toHaveBeenCalledWith('/')
  })
})
