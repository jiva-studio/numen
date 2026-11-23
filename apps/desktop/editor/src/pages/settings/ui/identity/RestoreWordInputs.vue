<script setup lang="ts">
/**
 * The twelve fields the recovery words of an identity are typed into, numbered in three columns.
 */
import { WORDS as words } from '../../words'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ entered: readonly string[]; isDisabled: boolean }>()

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
  <ol class="restore-words">
    <li v-for="(word, index) in entered" :key="index" class="restore-words__item">
      <span class="restore-words__n" aria-hidden="true">{{ index + 1 }}</span>
      <input
        class="restore-words__input"
        type="text"
        autocomplete="off"
        autocapitalize="off"
        spellcheck="false"
        :value="word"
        :disabled="isDisabled"
        :aria-label="words.identityRestoreWord(index + 1)"
        @input="onInput(index, $event)"
      />
    </li>
  </ol>
</template>

<style scoped>
.restore-words {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.375rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.restore-words__item {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  min-inline-size: 0;
}

.restore-words__n {
  min-inline-size: 1.1em;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  font-variant-numeric: tabular-nums;
  text-align: end;
}

.restore-words__input {
  box-sizing: border-box;
  appearance: none;
  flex: 1;
  min-inline-size: 0;
  block-size: var(--numen-action-size);
  margin: 0;
  padding: 0 0.5rem;
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-2);
}

.restore-words__input:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}
</style>
