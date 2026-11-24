<script setup lang="ts">
/** The question before a received folder is left: the files stay on this device either way. */
import { ConfirmDialog } from '@numen/ui'
import { WORDS as words } from '../words'

const emit = defineEmits<{
  (event: 'confirm'): void
  (event: 'cancel'): void
}>()
</script>

<template>
  <ConfirmDialog
    :label="words.leaveTitle"
    :text="words.leaveText"
    :confirm-text="words.leaveShare"
    :cancel-text="words.cancel"
    @confirm="emit('confirm')"
    @cancel="emit('cancel')"
  />
</template>
