<script setup lang="ts">
import Button from 'primevue/button'
</script>

<template>
  <main
    class="grid min-h-screen place-items-center content-center bg-brand-navy p-6 text-center text-white"
    style="background-image: radial-gradient(circle at top left, rgb(255 189 89 / 35%), transparent 35%)"
  >
    <span class="text-[4rem] font-extrabold text-brand-amber">404</span>
    <h1 class="my-2">Page not found</h1>
    <p class="mb-5 text-[#c6d4e2]">The page may not be part of the current BWIMS milestone.</p>
    <Button label="Return to dashboard" as="router-link" to="/" />
  </main>
</template>
