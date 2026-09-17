<script setup lang="ts">
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import Message from 'primevue/message'
import Password from 'primevue/password'
import { ref, watch } from 'vue'

import type { User } from '@/api/users'
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

watch(
  () => props.visible,
  (isVis) => {
    if (isVis) {
      newPassword.value = ''
      errorMessage.value = ''
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
  } catch (err: any) {
    errorMessage.value = err.message || 'Password reset failed'
  }
}

function closeDialog() {
  emit('update:visible', false)
}
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    header="Reset User Password"
    :style="{ width: '90vw', maxWidth: '440px' }"
    @update:visible="closeDialog"
  >
    <div class="grid gap-4 py-2">
      <p class="text-xs text-brand-muted m-0">
        Resetting password for <strong>{{ user?.full_name }}</strong> ({{ user?.email }}).
        This will immediately revoke all active refresh tokens for this user.
      </p>

      <Message v-if="errorMessage" severity="error">
        {{ errorMessage }}
      </Message>

      <div class="grid gap-1">
        <label for="new-pw" class="text-xs font-semibold text-brand-muted">New Password (min 12 characters) *</label>
        <Password
          id="new-pw"
          v-model="newPassword"
          toggle-mask
          placeholder="••••••••••••"
          input-class="w-full"
          class="w-full"
          required
        />
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button label="Cancel" severity="secondary" outlined @click="closeDialog" />
        <Button
          label="Reset Password"
          icon="pi pi-key"
          severity="warn"
          :loading="usersStore.loading"
          @click="resetUserPassword"
        />
      </div>
    </template>
  </Dialog>
</template>
