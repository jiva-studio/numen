<script setup lang="ts">
/**
 * The fields that prove an account: a choice between the twelve recovery words, one field each, and
 * two sheets of a recovery kit.
 */
import { SegmentedControl } from '@numen/ui'
import type { ProofKind } from '../../model/useRecoveryProof'
import { WORDS as words } from '../../words'
import RecoveryWords from './RecoveryWords.vue'
import SheetsInput from './SheetsInput.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  kind: ProofKind
  entered: readonly string[]
  texts: readonly [string, string]
  isDisabled: boolean
}>()

const emit = defineEmits<{
  (e: 'change-kind', kind: ProofKind): void
  (e: 'change-word', index: number, value: string): void
  (e: 'change-sheet', index: number, value: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const choices = [
  { id: 'words', text: words.proofWords },
  { id: 'sheets', text: words.proofSheets },
] as const

/* -------------------------------- Handlers -------------------------------- */
function onChoose(value: string): void {
  emit('change-kind', value === 'sheets' ? 'sheets' : 'words')
}
</script>

<template>
  <SegmentedControl
    class="proof-fields__choice"
    :model-value="kind"
    :choices="choices"
    :is-disabled="isDisabled"
    :aria-label="words.proofChoice"
    @update:model-value="onChoose"
  />
  <RecoveryWords
    v-if="kind === 'words'"
    :recovery-words="entered"
    is-editable
    :is-disabled="isDisabled"
    @change="(index, value) => emit('change-word', index, value)"
  />
  <SheetsInput
    v-else
    :texts="texts"
    :is-disabled="isDisabled"
    @change="(index, value) => emit('change-sheet', index, value)"
  />
</template>

<style scoped>
.proof-fields__choice {
  align-self: flex-start;
}
</style>
