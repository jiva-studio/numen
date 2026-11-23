<script setup lang="ts">
/**
 * The buttons at the foot of the identity setup dialog: cancel and the confirmation that the words
 * are written down in the first step, back and continue in the check. The last button submits the
 * form the buttons are in.
 */
import { Button } from '@numen/ui'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
defineProps<{
  stage: 'words' | 'check'
  isBusy: boolean
  isComplete: boolean
}>()

const emit = defineEmits<{
  (e: 'cancel'): void
  (e: 'back'): void
}>()
</script>

<template>
  <div v-if="stage === 'words'" class="identity-setup-actions">
    <Button type="button" variant="outline" @click="emit('cancel')">
      {{ words.identityCancel }}
    </Button>
    <Button type="submit" variant="solid">{{ words.identityWordsWritten }}</Button>
  </div>
  <div v-else class="identity-setup-actions">
    <Button type="button" variant="outline" :disabled="isBusy" @click="emit('back')">
      {{ words.identityBack }}
    </Button>
    <Button type="submit" variant="solid" :disabled="isBusy || !isComplete">
      {{ isBusy ? words.identityKeeping : words.identityContinue }}
    </Button>
  </div>
</template>

<style scoped>
.identity-setup-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.375rem;
}
</style>
