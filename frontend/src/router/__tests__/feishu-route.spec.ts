import { describe, expect, it, vi } from 'vitest'

const authStore = vi.hoisted(() => ({
  checkAuth: vi.fn(),
  isAuthenticated: false,
  isAdmin: false,
  isSimpleMode: false,
}))

const appStore = vi.hoisted(() => ({
  siteName: 'Sub2API',
  backendModeEnabled: false,
  cachedPublicSettings: null as null | Record<string, unknown>,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore,
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({
    customMenuItems: [],
  }),
}))

vi.mock('@/composables/useNavigationLoading', () => ({
  useNavigationLoadingState: () => ({
    startNavigation: vi.fn(),
    endNavigation: vi.fn(),
    isLoading: { value: false },
  }),
}))

vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({
    triggerPrefetch: vi.fn(),
    cancelPendingPrefetch: vi.fn(),
    resetPrefetchState: vi.fn(),
  }),
}))

describe('router Feishu OAuth route', () => {
  it('registers the Feishu callback route as a public route', async () => {
    const { default: router } = await import('@/router')
    const route = router.getRoutes().find((record) => record.name === 'auth-feishu-callback')

    expect(route?.path).toBe('/auth/feishu/callback')
    expect(route?.meta.requiresAuth).toBe(false)
  })

  it('loads the dedicated Feishu callback view instead of LinuxDo callback view', async () => {
    const { default: router } = await import('@/router')
    const route = router.getRoutes().find((record) => record.name === 'auth-feishu-callback')
    const loaded = route?.components?.default ? await route.components.default() : null

    expect(loaded).toBeTruthy()
    expect(String((loaded as any)?.default?.__file || '')).toContain('FeishuCallbackView.vue')
  })
})
