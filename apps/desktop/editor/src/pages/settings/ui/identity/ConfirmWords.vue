<script setup lang="ts">
/**
 * The check of a new identity: one labelled input for each of three positions among the twelve
 * words, so the person shows they wrote the words down before the identity is kept. A wrong word
 * marks its input.
 */
import { useId } from 'vue'
import { WORDS as words } from '../../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  positions: readonly number[]
  entered: Readonly<Record<number, string>>
  wrong: readonly number[]
  isDisabled: boolean
}>()

const emit = defineEmits<{
  (e: 'change', position: number, value: string): void
  (e: 'submit'): void
}>()

/* --------------------------------- State ---------------------------------- */
const prefix = useId()

/* -------------------------------- Handlers -------------------------------- */
function onInput(position: number, event: Event): void {
  const target = event.target as HTMLInputElement | null
  emit('change', position, target?.value ?? '')
}
</script>

<template>
  <div v-for="position in props.positions" :key="position" class="confirm-words__field">
    <label class="word-check" :for="`${prefix}-${position}`">
      <span class="word-check__n">{{ words.identityConfirmWord(position) }}</span>
      <input
        :id="`${prefix}-${position}`"
        class="confirm-words__input"
        type="text"
        autocomplete="off"
        autocapitalize="off"
        spellcheck="false"
        :aria-invalid="props.wrong.includes(position) || undefined"
        :value="props.entered[position] ?? ''"
        :disabled="props.isDisabled"
        :placeholder="words.identityConfirmPlaceholder"
        @input="onInput(position, $event)"
        @keydown.enter.prevent="emit('submit')"
      />
    </label>
  </div>
</template>

<style scoped>
.confirm-words__field {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.word-check {
  display: grid;
  grid-template-columns: 4.5rem minmax(0, 1fr);
  align-items: center;
  gap: 0.5rem;
}

.word-check__n {
  color: var(--numen-hushed);
  font-variant-numeric: tabular-nums;
}

.confirm-words__input {
  box-sizing: border-box;
  appearance: none;
  min-inline-size: 0;
  block-size: var(--numen-action-size);
  margin: 0;
  padding: 0 0.625rem;
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-2);
}

.confirm-words__input::placeholder {
  color: var(--numen-hushed);
}

.confirm-words__input:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.confirm-words__input[aria-invalid='true'] {
  border-color: var(--numen-alarm);
}
</style>
