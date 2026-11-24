<script setup lang="ts">
/** The question before a vault stops syncing or a received folder is left: the files stay on this device either way. */
import { ConfirmDialog } from '@numen/ui'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ kind: 'stop' | 'leave' }>()

const emit = defineEmits<{
  (event: 'confirm'): void
  (event: 'cancel'): void
}>()
</script>

<template>
  <ConfirmDialog
    :label="kind === 'stop' ? words.stopSyncing : words.leaveTitle"
    :text="kind === 'stop' ? words.stopSyncingText : words.leaveText"
    :confirm-text="kind === 'stop' ? words.stopSyncing : words.leaveShare"
    :cancel-text="words.cancel"
    @confirm="emit('confirm')"
    @cancel="emit('cancel')"
  />
</template>
