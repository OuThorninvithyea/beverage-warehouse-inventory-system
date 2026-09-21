<script setup lang="ts">
import { Loader2, Lock, Mail } from 'lucide-vue-next'
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { ApiClientError } from '@/api/client'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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
  <main class="relative grid min-h-screen place-items-center overflow-hidden bg-brand-navy p-6">
    <div
      class="pointer-events-none absolute inset-0"
      style="background-image: radial-gradient(60rem 40rem at 15% -10%, rgb(255 189 89 / 28%), transparent 60%), radial-gradient(50rem 30rem at 110% 110%, rgb(49 107 243 / 35%), transparent 55%)"
    />

    <div class="relative w-full max-w-[420px]">
      <div class="mb-6 flex items-center gap-3">
        <span class="grid size-11 place-items-center rounded-xl bg-brand-amber text-lg font-extrabold text-brand-navy shadow-lg">
          BW
        </span>
        <div class="grid leading-tight">
          <strong class="text-base font-semibold tracking-tight text-white">BWIMS</strong>
          <small class="text-xs text-white/60">Beverage Warehouse Control</small>
        </div>
      </div>

      <Card class="border-white/10 bg-white/95 shadow-2xl backdrop-blur supports-[backdrop-filter]:bg-white/90">
        <CardHeader>
          <CardTitle class="text-xl">Welcome back</CardTitle>
          <CardDescription>Sign in with your assigned warehouse account.</CardDescription>
        </CardHeader>

        <CardContent>
          <form class="grid gap-4" @submit.prevent="submit">
            <div class="grid gap-2">
              <Label for="email">Email address</Label>
              <div class="relative">
                <Mail class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  id="email"
                  v-model="email"
                  type="email"
                  autocomplete="email"
                  placeholder="you@bwims.local"
                  class="pl-9"
                />
              </div>
            </div>

            <div class="grid gap-2">
              <Label for="password">Password</Label>
              <div class="relative">
                <Lock class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  id="password"
                  v-model="password"
                  type="password"
                  autocomplete="current-password"
                  placeholder="••••••••"
                  class="pl-9"
                />
              </div>
            </div>

            <p
              v-if="errorMessage"
              role="alert"
              class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
            >
              {{ errorMessage }}
            </p>

            <Button
              type="submit"
              class="w-full"
              :disabled="auth.loading || !email.trim() || !password"
            >
              <Loader2 v-if="auth.loading" class="size-4 animate-spin" />
              Sign in
            </Button>

            <p class="text-center text-xs text-muted-foreground">
              Access is controlled by your admin, manager, picker or viewer role.
            </p>
          </form>
        </CardContent>
      </Card>
    </div>
  </main>
</template>
