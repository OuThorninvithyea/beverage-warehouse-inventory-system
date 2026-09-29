<script setup lang="ts">
/**
 * The BWIMS logo on a white tile. The mark is near-black navy, so it needs a
 * light surface to stay visible on the navy login screen and in dark mode.
 * Size it from the caller with a size-* class.
 *
 * It always sits beside the "BWIMS" wordmark, so it is decorative: an empty
 * alt stops screen readers announcing the name twice.
 */
import logo from '@/assets/logo.png'
</script>

<template>
  <span class="grid shrink-0 place-items-center rounded-xl bg-white shadow-sm ring-1 ring-black/5">
    <img :src="logo" alt="" aria-hidden="true" class="size-[82%] object-contain" />
  </span>
</template>
