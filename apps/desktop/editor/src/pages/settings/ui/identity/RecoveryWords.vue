<script setup lang="ts">
/**
 * The twelve recovery words of a new identity, numbered in the order they must be kept.
 */
import { WORDS as words } from '../../words'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ recoveryWords: readonly string[] }>()
</script>

<template>
  <ol class="words" :aria-label="words.identityRecoveryWords">
    <li v-for="(word, index) in recoveryWords" :key="index">{{ word }}</li>
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

.words li {
  counter-increment: word;
  display: flex;
  align-items: baseline;
  gap: 0.375rem;
  padding: 0.375rem 0.5rem;
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  background: var(--numen-field-bg);
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-2);
}

.words li::before {
  content: counter(word);
  min-inline-size: 1.1em;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  font-variant-numeric: tabular-nums;
  text-align: end;
}
</style>
