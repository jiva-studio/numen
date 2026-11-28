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
  <template v-if="stage === 'words'">
    <Button type="button" variant="ghost" @click="emit('cancel')">
      {{ words.identityCancel }}
    </Button>
    <Button type="submit" variant="solid">{{ words.identityWordsWritten }}</Button>
  </template>
  <template v-else>
    <Button type="button" variant="ghost" :disabled="isBusy" @click="emit('back')">
      {{ words.identityBack }}
    </Button>
    <Button type="submit" variant="solid" :disabled="isBusy || !isComplete">
      {{ isBusy ? words.identityKeeping : words.identityContinue }}
    </Button>
  </template>
</template>
