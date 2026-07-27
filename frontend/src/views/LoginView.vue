<script setup lang="ts">
import Button from 'primevue/button'
import Card from 'primevue/card'
import InputText from 'primevue/inputtext'
import { ref } from 'vue'
import { RouterLink } from 'vue-router'

const email = ref('')
const password = ref('')
</script>

<template>
  <main class="auth-page">
    <RouterLink to="/" class="back-link">← Back to dashboard</RouterLink>
    <Card class="login-card">
      <template #title>Welcome to BWIMS</template>
      <template #subtitle>Sign-in screen foundation</template>
      <template #content>
        <form class="login-form" @submit.prevent>
          <label for="email">Email address</label>
          <InputText id="email" v-model="email" type="email" autocomplete="email" />

          <label for="password">Password</label>
          <InputText
            id="password"
            v-model="password"
            type="password"
            autocomplete="current-password"
          />

          <Button type="submit" label="Sign in" disabled />
          <small>Authentication is implemented with the Week 7–8 API.</small>
        </form>
      </template>
    </Card>
  </main>
</template>
