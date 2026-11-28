<script setup lang="ts">
/**
 * The dialog that restores an identity from the 12 recovery words in one field. The restore waits until
 * the 12 words are in.
 */
import { computed } from 'vue'
import { Button } from '@numen/ui'
import type { ProofReading } from '../../model/readProof'
import { WORDS as words } from '../../words'
import IdentityDialog from './IdentityDialog.vue'
import RecoveryProofField from './RecoveryProofField.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  text: string
  reading: ProofReading
  problem: string
  isBusy: boolean
}>()

const emit = defineEmits<{
  (e: 'change', value: string): void
  (e: 'restore'): void
  (e: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const isComplete = computed(() => props.reading.proof !== undefined)
</script>

<template>
  <IdentityDialog
    :title="words.identityRestoreWords"
    :hint="words.restoreWordsHint"
    :caution="words.restoreUnverifiedNote"
    @submit="emit('restore')"
    @cancel="emit('cancel')"
  >
    <RecoveryProofField
      :text="text"
      :reading="reading"
      :problem="problem"
      :is-disabled="isBusy"
      @change="(value) => emit('change', value)"
    />
    <template #actions>
      <Button type="button" variant="ghost" :disabled="isBusy" @click="emit('cancel')">
        {{ words.identityCancel }}
      </Button>
      <Button type="submit" variant="solid" :disabled="isBusy || !isComplete">
        {{ isBusy ? words.restoringNow : words.restoreAction }}
      </Button>
    </template>
  </IdentityDialog>
</template>
