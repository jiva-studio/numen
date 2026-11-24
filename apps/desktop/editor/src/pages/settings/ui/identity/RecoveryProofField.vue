<script setup lang="ts">
/**
 * The one field that takes what proves an account: the 12 recovery words, or two sheets of a recovery
 * kit with a sheet on each line. A line under the field says what was recognized, or what is wrong.
 */
import { computed, useId } from 'vue'
import type { ProofReading } from '../../model/readProof'
import { WORDS as words } from '../../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  text: string
  reading: ProofReading
  problem: string
  isDisabled: boolean
}>()

const emit = defineEmits<{
  (e: 'change', value: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const noteId = useId()
const isProblem = computed(() => props.problem !== '' || props.reading.isProblem)
const note = computed(() => (props.problem !== '' ? props.problem : props.reading.note))

/* -------------------------------- Handlers -------------------------------- */
function onInput(event: Event): void {
  const target = event.target as HTMLTextAreaElement | null
  emit('change', target?.value ?? '')
}
</script>

<template>
  <div class="proof-field">
    <textarea
      class="proof-field__input"
      rows="6"
      autocomplete="off"
      autocapitalize="off"
      spellcheck="false"
      :value="text"
      :disabled="isDisabled"
      :placeholder="words.proofPlaceholder"
      :aria-label="words.proofField"
      :aria-describedby="noteId"
      :aria-invalid="isProblem || undefined"
      @input="onInput"
    />
    <p
      :id="noteId"
      class="proof-field__note"
      :class="{ 'proof-field__note--problem': isProblem }"
      :role="isProblem ? 'alert' : 'status'"
    >
      {{ note }}
    </p>
  </div>
</template>

<style scoped>
.proof-field {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.proof-field__input {
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

.proof-field__input::placeholder {
  color: var(--numen-hushed);
}

.proof-field__input:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.proof-field__input[aria-invalid='true'] {
  border-color: var(--numen-alarm);
}

.proof-field__note {
  min-block-size: calc(1em * var(--numen-line-height));
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.proof-field__note--problem {
  color: var(--numen-alarm);
}
</style>
