<script setup lang="ts">
/**
 * The check of a new identity: one input for each of three positions among the twelve words, so the
 * person shows they wrote the words down before the identity is kept.
 */
import { WORDS as words } from '../../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  positions: readonly number[]
  entered: Readonly<Record<number, string>>
  isDisabled: boolean
}>()

const emit = defineEmits<{
  (e: 'change', position: number, value: string): void
  (e: 'submit'): void
}>()

/* -------------------------------- Handlers -------------------------------- */
function onInput(position: number, event: Event): void {
  const target = event.target as HTMLInputElement | null
  emit('change', position, target?.value ?? '')
}
</script>

<template>
  <div class="confirm-words">
    <p class="confirm-words__note">{{ words.identityConfirmHint }}</p>
    <div class="confirm-words__fields">
      <input
        v-for="position in props.positions"
        :key="position"
        class="confirm-words__input"
        type="text"
        autocomplete="off"
        autocapitalize="off"
        spellcheck="false"
        :value="props.entered[position] ?? ''"
        :disabled="props.isDisabled"
        :placeholder="`${position}`"
        :aria-label="words.identityConfirmWord(position)"
        @input="onInput(position, $event)"
        @keydown.enter.prevent="emit('submit')"
      />
    </div>
  </div>
</template>

<style scoped>
.confirm-words {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.confirm-words__note {
  margin: 0;
  font-size: var(--numen-text-1);
  color: var(--numen-hushed);
}

.confirm-words__fields {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.confirm-words__input {
  box-sizing: border-box;
  appearance: none;
  flex: 1 1 8rem;
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

.confirm-words__input:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}
</style>
