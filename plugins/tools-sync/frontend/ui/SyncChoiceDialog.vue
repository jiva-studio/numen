<script setup lang="ts">
/** The question of "Sync this vault" when the account holds a vault of the same name: use that copy or send this one. */
import { Button, Dialog } from '@numen/ui'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ name: string }>()

const emit = defineEmits<{
  (event: 'use'): void
  (event: 'send'): void
  (event: 'cancel'): void
}>()
</script>

<template>
  <Dialog
    :label="words.syncVaultTitle(name)"
    :hint="words.syncVaultText(name)"
    @cancel="emit('cancel')"
  >
    <template #actions>
      <Button variant="ghost" @click="emit('cancel')">{{ words.cancel }}</Button>
      <Button variant="outline" @click="emit('send')">{{ words.sendThisOne }}</Button>
      <Button @click="emit('use')">{{ words.useCloudCopy }}</Button>
    </template>
  </Dialog>
</template>
