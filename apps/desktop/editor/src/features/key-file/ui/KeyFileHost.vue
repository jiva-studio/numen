<script setup lang="ts">
/** Shows the question the key file asks: how to keep the keys, or the passphrase that opens them. */
import type { KeyFileStore } from '@/entities/key-file'
import KeyFileDialog from './KeyFileDialog.vue'
import KeyFileUnlockDialog from './KeyFileUnlockDialog.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ keyFile: KeyFileStore }>()
</script>

<template>
  <KeyFileDialog
    v-if="keyFile.request.value === 'choose'"
    :problem="keyFile.problem.value"
    :is-busy="keyFile.isBusy.value"
    @choose="keyFile.dialog.choose"
    @cancel="keyFile.dialog.cancel"
  />
  <KeyFileUnlockDialog
    v-else-if="keyFile.request.value === 'unlock'"
    :problem="keyFile.problem.value"
    :is-busy="keyFile.isBusy.value"
    @unlock="keyFile.dialog.unlock"
    @cancel="keyFile.dialog.cancel"
  />
</template>
