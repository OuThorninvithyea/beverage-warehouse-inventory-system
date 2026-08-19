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
  <div class="grid min-h-screen grid-cols-[260px_minmax(0,1fr)] max-[800px]:grid-cols-1">
    <aside class="flex flex-col gap-8 bg-brand-navy p-6 text-[#f7f9fc] max-[800px]:gap-4">
      <div class="flex items-center gap-3">
        <span
          class="grid h-[42px] w-[42px] place-items-center rounded-[12px] bg-brand-amber font-extrabold text-brand-navy"
        >BW</span>
        <div class="grid">
          <strong>BWIMS</strong>
          <small class="text-[#a8bdd5]">Warehouse control</small>
        </div>
      </div>

      <nav
        aria-label="Primary navigation"
        class="grid gap-[0.4rem] max-[800px]:grid-cols-4 max-[800px]:overflow-x-auto max-[520px]:grid-cols-2"
      >
        <RouterLink
          to="/"
          class="rounded-[10px] px-[0.9rem] py-[0.8rem] [&.router-link-active]:bg-brand-amber [&.router-link-active]:font-bold [&.router-link-active]:text-brand-ink"
        >Dashboard</RouterLink>
        <RouterLink
          to="/warehouses"
          class="rounded-[10px] px-[0.9rem] py-[0.8rem] [&.router-link-active]:bg-brand-amber [&.router-link-active]:font-bold [&.router-link-active]:text-brand-ink"
        >Warehouses</RouterLink>
        <RouterLink
          to="/barcode-test"
          class="rounded-[10px] px-[0.9rem] py-[0.8rem] [&.router-link-active]:bg-brand-amber [&.router-link-active]:font-bold [&.router-link-active]:text-brand-ink"
        >Barcode test</RouterLink>
        <span class="cursor-not-allowed rounded-[10px] px-[0.9rem] py-[0.8rem] text-[#9fb1c6]">Inventory</span>
        <span class="cursor-not-allowed rounded-[10px] px-[0.9rem] py-[0.8rem] text-[#9fb1c6]">Movements</span>
      </nav>

      <div
        class="mt-auto grid gap-[0.35rem] rounded-[12px] border border-white/12 bg-white/5 p-4 text-[#dce7f2] max-[800px]:hidden"
      >
        <small class="text-[#a8bdd5]">Week 11</small>
        <span>Backend complete through inventory & movements. Warehouses is the first live frontend screen.</span>
      </div>
    </aside>

    <div class="min-w-0">
      <header
        class="flex min-h-[76px] items-center justify-between border-b border-[#dde4ee] bg-white px-8 py-4 max-[520px]:px-4"
      >
        <div class="grid gap-[0.2rem]">
          <small class="uppercase tracking-[0.08em] text-brand-muted">Beverage warehouse</small>
          <strong class="max-[520px]:text-[0.9rem]">Inventory Management System</strong>
        </div>
        <div class="flex items-center gap-[0.8rem]">
          <span v-if="auth.user" class="grid text-right text-[0.9rem] font-[650] text-brand-ink">
            {{ auth.user.full_name }}
            <small class="font-medium capitalize text-brand-muted">{{ auth.user.role.replace('_', ' ') }}</small>
          </span>
          <Button label="Sign out" size="small" severity="secondary" @click="signOut" />
        </div>
      </header>

      <main class="p-[clamp(1.25rem,3vw,2.5rem)]">
        <RouterView />
      </main>
    </div>
  </div>
</template>
