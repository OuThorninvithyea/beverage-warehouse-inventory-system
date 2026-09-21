import { createRouter, createWebHistory } from 'vue-router'

import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: () => import('@/layouts/AppLayout.vue'),
      children: [
        {
          path: '',
          name: 'dashboard',
          component: () => import('@/views/DashboardView.vue'),
        },
        {
          path: 'products',
          name: 'products',
          component: () => import('@/views/ProductsView.vue'),
        },
        {
          path: 'categories',
          name: 'categories',
          component: () => import('@/views/CategoriesView.vue'),
        },
        {
          path: 'inventory',
          name: 'inventory',
          component: () => import('@/views/InventoryView.vue'),
        },
        {
          path: 'alerts',
          name: 'expiry-alerts',
          component: () => import('@/views/ExpiryAlertsView.vue'),
        },
        {
          path: 'movements',
          name: 'movements',
          component: () => import('@/views/MovementsView.vue'),
        },
        {
          path: 'warehouses',
          name: 'warehouses',
          component: () => import('@/views/WarehousesListView.vue'),
        },
        {
          path: 'warehouses/:warehouseId',
          name: 'warehouse-detail',
          component: () => import('@/views/WarehouseDetailView.vue'),
        },
        {
          path: 'users',
          name: 'users',
          component: () => import('@/views/UsersView.vue'),
          meta: { roles: ['admin'] },
        },
        {
          path: 'barcode-test',
          name: 'barcode-test',
          component: () => import('@/views/BarcodeTestView.vue'),
          meta: { public: true },
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/views/NotFoundView.vue'),
      meta: { public: true },
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  await auth.restoreSession()

  const isPublic = to.matched.some((record) => record.meta.public)
  if (!isPublic && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.name === 'login' && auth.isAuthenticated) {
    return { name: 'dashboard' }
  }

  // Role guard
  const requiredRoles = to.meta.roles as string[] | undefined
  if (requiredRoles && auth.user) {
    if (!requiredRoles.includes(auth.user.role)) {
      return { name: 'dashboard' }
    }
  }

  return true
})

export default router
