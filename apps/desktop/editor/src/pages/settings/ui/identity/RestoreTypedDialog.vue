<script setup lang="ts">
/**
 * The dialog that restores an identity from two sheets of a recovery kit typed: the words of one sheet in
 * each field. It is the way on a machine with no camera.
 */
import { computed, useId } from 'vue'
import { Button } from '@numen/ui'
import type { ProofReading } from '../../model/readProof'
import { WORDS as words } from '../../words'
import IdentityDialog from './IdentityDialog.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  sheets: readonly [string, string]
  reading: ProofReading
  problem: string
  isBusy: boolean
}>()

const emit = defineEmits<{
  (e: 'change', position: 0 | 1, value: string): void
  (e: 'restore'): void
  (e: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const noteId = useId()
const isComplete = computed(() => props.reading.proof !== undefined)
const isProblem = computed(() => props.problem !== '' || props.reading.isProblem)
const note = computed(() => (props.problem !== '' ? props.problem : props.reading.note))

/* -------------------------------- Handlers -------------------------------- */
function onInput(position: 0 | 1, event: Event): void {
  emit('change', position, (event.target as HTMLTextAreaElement | null)?.value ?? '')
}
</script>

<template>
  <IdentityDialog
    :title="words.restoreTypedTitle"
    :hint="words.restoreTypedHint"
    @submit="emit('restore')"
    @cancel="emit('cancel')"
  >
    <textarea
      v-for="position in [0, 1] as const"
      :key="position"
      class="typed-sheet"
      rows="4"
      autocomplete="off"
      autocapitalize="off"
      spellcheck="false"
      :value="sheets[position]"
      :disabled="isBusy"
      :aria-label="words.restoreSheetField(position + 1)"
      :placeholder="words.restoreSheetField(position + 1)"
      :aria-describedby="note !== '' ? noteId : undefined"
      :aria-invalid="isProblem || undefined"
      @input="onInput(position, $event)"
    />
    <p
      v-if="note !== ''"
      :id="noteId"
      class="typed-note"
      :class="{ 'typed-note--problem': isProblem }"
      :role="isProblem ? 'alert' : 'status'"
    >
      {{ note }}
    </p>
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

<style scoped>
.typed-sheet {
  box-sizing: border-box;
  appearance: none;
  inline-size: 100%;
  margin: 0;
  padding: 0.625rem 0.75rem;
  resize: vertical;
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-2);
  line-height: var(--numen-line-height);
}

.typed-sheet::placeholder {
  color: var(--numen-hushed);
}

.typed-sheet:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.typed-sheet[aria-invalid='true'] {
  border-color: var(--numen-alarm);
}

.typed-note {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
}

.typed-note--problem {
  color: var(--numen-alarm);
}
</style>
