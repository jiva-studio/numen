<script setup lang="ts">
/**
 * The twelve recovery words of a new identity, numbered in the order they must be kept.
 */
import { WORDS as words } from '../../words'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ recoveryWords: readonly string[] }>()
</script>

<template>
  <div class="recovery-words">
    <p class="recovery-words__note">{{ words.identityWriteDown }}</p>
    <ol class="recovery-words__list" :aria-label="words.identityRecoveryWords">
      <li v-for="(word, index) in recoveryWords" :key="index" class="recovery-words__word">
        {{ word }}
      </li>
    </ol>
  </div>
</template>

<style scoped>
.recovery-words {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.recovery-words__note {
  margin: 0;
  font-size: var(--numen-text-1);
  color: var(--numen-hushed);
}

.recovery-words__list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(8rem, 1fr));
  gap: 0.375rem 1rem;
  margin: 0;
  padding: 0.75rem 0.75rem 0.75rem 2.5rem;
  background-color: var(--numen-bubble-bg);
  border-radius: var(--numen-radius);
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
}

.recovery-words__word {
  padding-inline-start: 0.25rem;
}
</style>
