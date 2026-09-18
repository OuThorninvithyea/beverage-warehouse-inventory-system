<script setup lang="ts">
import { Eye, EyeOff, KeyRound } from 'lucide-vue-next'
import { ref, watch } from 'vue'

import type { User } from '@/api/users'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useUsersStore } from '@/stores/users'

const props = defineProps<{
  visible: boolean
  user?: User | null
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}>()

const usersStore = useUsersStore()

const newPassword = ref('')
const errorMessage = ref('')
const showPassword = ref(false)

watch(
  () => props.visible,
  (isVis) => {
    if (isVis) {
      newPassword.value = ''
      errorMessage.value = ''
      showPassword.value = false
    }
  },
)

async function resetUserPassword() {
  errorMessage.value = ''
  if (!props.user?.id) return

  if (!newPassword.value || newPassword.value.length < 12) {
    errorMessage.value = 'Password must be at least 12 characters long'
    return
  }

  try {
    await usersStore.doResetPassword(props.user.id, newPassword.value)
    emit('success')
    closeDialog()
  } catch (err: unknown) {
    errorMessage.value = err instanceof Error ? err.message : 'Password reset failed'
  }
}

function closeDialog() {
  emit('update:visible', false)
}
</script>

<template>
  <Dialog :open="visible" @update:open="(value: boolean) => emit('update:visible', value)">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>Reset User Password</DialogTitle>
        <DialogDescription>
          Resetting password for <strong>{{ user?.full_name }}</strong> ({{ user?.email }}).
          This immediately revokes all active refresh tokens for this user.
        </DialogDescription>
      </DialogHeader>

      <div class="grid gap-4">
        <p
          v-if="errorMessage"
          class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ errorMessage }}
        </p>

        <div class="grid gap-2">
          <Label for="new-pw">New Password (min 12 characters) *</Label>
          <div class="relative">
            <Input
              id="new-pw"
              v-model="newPassword"
              :type="showPassword ? 'text' : 'password'"
              placeholder="••••••••••••"
              class="pr-10"
              required
            />
            <button
              type="button"
              class="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              :aria-label="showPassword ? 'Hide password' : 'Show password'"
              @click="showPassword = !showPassword"
            >
              <EyeOff v-if="showPassword" class="size-4" />
              <Eye v-else class="size-4" />
            </button>
          </div>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="closeDialog">Cancel</Button>
        <Button :disabled="usersStore.loading" @click="resetUserPassword">
          <KeyRound class="size-4" />
          Reset Password
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
