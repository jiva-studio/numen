<script setup lang="ts">
/**
 * The fields that prove an account: a choice between the twelve recovery words, typed into the
 * numbered boxes of the setup, and two sheets of a recovery kit, with a hint for each. Both sets of
 * fields stay laid out over each other and only the chosen one is shown, so a change of choice moves
 * nothing in the dialog.
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
  <div class="proof-fields">
    <SegmentedControl
      class="proof-fields__choice"
      :model-value="kind"
      :choices="choices"
      :is-disabled="isDisabled"
      :aria-label="words.proofChoice"
      @update:model-value="onChoose"
    />
    <div class="proof-fields__stack">
      <div
        class="proof-fields__pane"
        :class="{ 'proof-fields__pane--away': kind !== 'words' }"
        :inert="kind !== 'words' || undefined"
      >
        <p class="proof-fields__hint">{{ words.identityRestoreHint }}</p>
        <RecoveryWords
          :recovery-words="entered"
          is-editable
          :is-disabled="isDisabled"
          @change="(index, value) => emit('change-word', index, value)"
        />
      </div>
      <div
        class="proof-fields__pane"
        :class="{ 'proof-fields__pane--away': kind !== 'sheets' }"
        :inert="kind !== 'sheets' || undefined"
      >
        <p class="proof-fields__hint">{{ words.restoreSheetsHint }}</p>
        <SheetsInput
          :texts="texts"
          :is-disabled="isDisabled"
          @change="(index, value) => emit('change-sheet', index, value)"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.proof-fields {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.proof-fields__choice {
  align-self: flex-start;
}

.proof-fields__stack {
  display: grid;
}

.proof-fields__pane {
  grid-area: 1 / 1;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  min-inline-size: 0;
}

.proof-fields__pane--away {
  visibility: hidden;
}

.proof-fields__hint {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
