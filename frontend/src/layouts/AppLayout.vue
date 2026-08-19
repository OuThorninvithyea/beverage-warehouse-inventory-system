<script setup lang="ts">
import Button from 'primevue/button'
import { RouterLink, RouterView } from 'vue-router'
import { useRouter } from 'vue-router'

import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

async function signOut() {
  await auth.signOut()
  await router.push({ name: 'login' })
}
</script>

<template>
  <div class="app-shell">
    <aside class="sidebar">
      <div class="brand">
        <span class="brand-mark">BW</span>
        <div>
          <strong>BWIMS</strong>
          <small>Warehouse control</small>
        </div>
      </div>

      <nav aria-label="Primary navigation">
        <RouterLink to="/" class="nav-link">Dashboard</RouterLink>
        <RouterLink to="/warehouses" class="nav-link">Warehouses</RouterLink>
        <RouterLink to="/barcode-test" class="nav-link">Barcode test</RouterLink>
        <span class="nav-link nav-link--disabled">Inventory</span>
        <span class="nav-link nav-link--disabled">Movements</span>
      </nav>

      <div class="sidebar-note">
        <small>Week 11</small>
        <span>Backend complete through inventory & movements. Warehouses is the first live frontend screen.</span>
      </div>
    </aside>

    <div class="workspace">
      <header class="topbar">
        <div>
          <small class="eyebrow">Beverage warehouse</small>
          <strong>Inventory Management System</strong>
        </div>
        <div class="session-summary">
          <span v-if="auth.user">
            {{ auth.user.full_name }}
            <small>{{ auth.user.role.replace('_', ' ') }}</small>
          </span>
          <Button label="Sign out" size="small" severity="secondary" @click="signOut" />
        </div>
      </header>

      <main class="page-content">
        <RouterView />
      </main>
    </div>
  </div>
</template>
