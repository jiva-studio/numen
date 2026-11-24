<script setup lang="ts">
/**
 * The question the card asks before it deletes the cloud data of the identity: how long the server keeps it,
 * what stays on the devices and that an active subscription is cancelled.
 */
import { ConfirmDialog } from '@numen/ui'
import { WORDS as words } from '../words'

defineProps<{ hasSubscription: boolean }>()

const emit = defineEmits<{
  (event: 'confirm'): void
  (event: 'cancel'): void
}>()
</script>

<template>
  <ConfirmDialog
    tone="danger"
    :label="words.deleteQuestion"
    :confirm-text="words.deleteConfirm"
    :cancel-text="words.deleteCancel"
    @confirm="emit('confirm')"
    @cancel="emit('cancel')"
  >
    <div class="subscription__delete">
      <p>{{ words.deleteWhen }}</p>
      <p>{{ words.deleteLocal }}</p>
      <p v-if="hasSubscription">{{ words.deleteSubscription }}</p>
    </div>
  </ConfirmDialog>
</template>
