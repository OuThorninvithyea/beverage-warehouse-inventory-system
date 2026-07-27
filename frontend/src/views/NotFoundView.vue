<script setup lang="ts">
import Button from 'primevue/button'
</script>

<template>
  <main class="not-found">
    <span>404</span>
    <h1>Page not found</h1>
    <p>The page may not be part of the current BWIMS milestone.</p>
    <Button label="Return to dashboard" as="router-link" to="/" />
  </main>
</template>
