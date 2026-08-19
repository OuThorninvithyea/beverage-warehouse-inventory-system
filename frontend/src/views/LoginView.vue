<script setup lang="ts">
import Button from 'primevue/button'
import Card from 'primevue/card'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { ApiClientError } from '@/api/client'
import { useAuthStore } from '@/stores/auth'

const email = ref('')
const password = ref('')
const errorMessage = ref('')
const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

async function submit() {
  errorMessage.value = ''
  try {
    await auth.signIn(email.value, password.value)
    const redirect =
      typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    await router.push(redirect)
  } catch (error) {
    errorMessage.value =
      error instanceof ApiClientError
        ? error.message
        : 'Sign in could not be completed. Please try again.'
  }
}
</script>

<template>
  <main
    class="grid min-h-screen place-items-center content-center bg-brand-navy p-6"
    style="background-image: radial-gradient(circle at top left, rgb(255 189 89 / 35%), transparent 35%)"
  >
    <Card class="w-full max-w-[430px]">
      <template #title>Welcome to BWIMS</template>
      <template #subtitle>Use your assigned warehouse account</template>
      <template #content>
        <form class="grid gap-3" @submit.prevent="submit">
          <label for="email" class="mt-[0.4rem] font-[650]">Email address</label>
          <InputText id="email" v-model="email" type="email" autocomplete="email" />

          <label for="password" class="mt-[0.4rem] font-[650]">Password</label>
          <InputText
            id="password"
            v-model="password"
            type="password"
            autocomplete="current-password"
          />

          <Message v-if="errorMessage" severity="error">{{ errorMessage }}</Message>
          <Button
            type="submit"
            label="Sign in"
            :loading="auth.loading"
            :disabled="!email.trim() || !password"
          />
          <small class="text-center text-brand-muted">Access is controlled by your admin, manager, picker or viewer role.</small>
        </form>
      </template>
    </Card>
  </main>
</template>
