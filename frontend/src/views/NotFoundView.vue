<script setup lang="ts">
import { RouterLink } from 'vue-router'

import { Button } from '@/components/ui/button'
</script>

<template>
  <main
    class="relative grid min-h-screen place-items-center content-center bg-brand-navy p-6 text-center text-white"
  >
    <div
      class="pointer-events-none absolute inset-0"
      style="background-image: radial-gradient(60rem 40rem at 15% -10%, rgb(255 189 89 / 28%), transparent 60%)"
    />
    <div class="relative grid justify-items-center gap-2">
      <span class="text-[4rem] font-extrabold leading-none text-brand-amber">404</span>
      <h1 class="text-2xl font-semibold tracking-tight">Page not found</h1>
      <p class="mb-4 text-sm text-white/70">
        The page may not be part of the current BWIMS milestone.
      </p>
      <Button as-child>
        <RouterLink to="/">Return to dashboard</RouterLink>
      </Button>
    </div>
  </main>
</template>
