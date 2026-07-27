<script setup lang="ts">
import Button from 'primevue/button'
import { RouterLink, RouterView } from 'vue-router'
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
        <span class="nav-link nav-link--disabled">Inventory</span>
        <span class="nav-link nav-link--disabled">Movements</span>
        <span class="nav-link nav-link--disabled">Reports</span>
      </nav>

      <div class="sidebar-note">
        <small>Week 6 foundation</small>
        <span>Core modules arrive in Weeks 7–10.</span>
      </div>
    </aside>

    <div class="workspace">
      <header class="topbar">
        <div>
          <small class="eyebrow">Beverage warehouse</small>
          <strong>Inventory Management System</strong>
        </div>
        <Button label="Sign in" as="router-link" to="/login" size="small" />
      </header>

      <main class="page-content">
        <RouterView />
      </main>
    </div>
  </div>
</template>
