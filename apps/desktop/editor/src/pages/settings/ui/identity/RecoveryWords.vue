<script setup lang="ts">
/**
 * The twelve recovery words of an identity in numbered boxes, three to a row, in the order they must
 * be kept. Shown, the boxes hold the words of a new identity; editable, each box is the field one
 * word of a restore is typed into.
 */
import { WORDS as words } from '../../words'

/* --------------------------------- Props ---------------------------------- */
withDefaults(
  defineProps<{ recoveryWords: readonly string[]; isEditable?: boolean; isDisabled?: boolean }>(),
  { isEditable: false, isDisabled: false },
)

const emit = defineEmits<{
  (e: 'change', index: number, value: string): void
}>()

/* -------------------------------- Handlers -------------------------------- */
function onInput(index: number, event: Event): void {
  const target = event.target as HTMLInputElement | null
  emit('change', index, target?.value ?? '')
}
</script>

<template>
  <ol class="words" :aria-label="words.identityRecoveryWords">
    <li v-for="(word, index) in recoveryWords" :key="index" class="words__box">
      <input
        v-if="isEditable"
        class="words__input"
        type="text"
        autocomplete="off"
        autocapitalize="off"
        spellcheck="false"
        :value="word"
        :disabled="isDisabled"
        :aria-label="words.identityRestoreWord(index + 1)"
        @input="onInput(index, $event)"
      />
      <template v-else>{{ word }}</template>
    </li>
  </ol>
</template>

<style scoped>
.words {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.375rem;
  margin: 0;
  padding: 0;
  list-style: none;
  counter-reset: word;
}

.words__box {
  counter-increment: word;
  display: flex;
  align-items: baseline;
  gap: 0.375rem;
  min-inline-size: 0;
  padding: 0.375rem 0.5rem;
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  background: var(--numen-field-bg);
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-2);
}

.words__box::before {
  content: counter(word);
  min-inline-size: 1.1em;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  font-variant-numeric: tabular-nums;
  text-align: end;
}

.words__box:focus-within {
  border-color: var(--numen-accent);
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.words__input {
  box-sizing: border-box;
  appearance: none;
  flex: 1;
  min-inline-size: 0;
  margin: 0;
  padding: 0;
  border: none;
  background: none;
  color: var(--numen-ink);
  font: inherit;
}

.words__input:focus {
  outline: none;
}

.words__input:disabled {
  opacity: 0.5;
}
</style>
