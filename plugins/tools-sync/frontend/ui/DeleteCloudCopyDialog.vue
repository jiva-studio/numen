<script setup lang="ts">
/** The question before the cloud copy of a vault that is not synced on this device is deleted. */
import { ConfirmDialog } from '@numen/ui'
import { WORDS as words } from '../words'

defineProps<{ name: string }>()

const emit = defineEmits<{
  (event: 'confirm'): void
  (event: 'cancel'): void
}>()
</script>

<template>
  <ConfirmDialog
    tone="danger"
    :label="words.deleteCloudQuestion(name)"
    :confirm-text="words.deleteCloudConfirm"
    :cancel-text="words.cancel"
    @confirm="emit('confirm')"
    @cancel="emit('cancel')"
  />
</template>
