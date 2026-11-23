<script setup lang="ts">
/**
 * The proof asked for before a later kit: the 12 words, or two sheets of an earlier kit.
 */
import { Button } from '@numen/ui'
import { WORDS as words } from '../../words'
import SheetsInput from './SheetsInput.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ wordsText: string; texts: readonly [string, string]; isBusy: boolean }>()

const emit = defineEmits<{
  (e: 'change-words', value: string): void
  (e: 'change-sheet', index: number, value: string): void
  (e: 'make'): void
  (e: 'cancel'): void
}>()

/* -------------------------------- Handlers -------------------------------- */
function onInput(event: Event): void {
  const target = event.target as HTMLTextAreaElement | null
  emit('change-words', target?.value ?? '')
}
</script>

<template>
  <p class="kit-proof__hint">{{ words.kitProofHint }}</p>
  <textarea
    class="kit-proof__words"
    rows="3"
    autocomplete="off"
    autocapitalize="off"
    spellcheck="false"
    :value="wordsText"
    :disabled="isBusy"
    :placeholder="words.kitWordsPlaceholder"
    :aria-label="words.kitWordsLabel"
    @input="onInput"
  />
  <SheetsInput
    :texts="texts"
    :is-disabled="isBusy"
    @change="(index, value) => emit('change-sheet', index, value)"
  />
  <p class="kit-proof__hint">{{ words.kitOlderStays }}</p>
  <div class="kit-proof__actions">
    <Button type="button" variant="solid" :disabled="isBusy" @click="emit('make')">
      {{ words.kitMake }}
    </Button>
    <Button type="button" variant="outline" :disabled="isBusy" @click="emit('cancel')">
      {{ words.identityCancel }}
    </Button>
  </div>
</template>

<style scoped>
.kit-proof__hint {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.kit-proof__words {
  box-sizing: border-box;
  appearance: none;
  inline-size: 100%;
  margin: 0;
  padding: 0.5rem 0.625rem;
  resize: vertical;
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-2);
  line-height: var(--numen-line-height);
}

.kit-proof__words:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.kit-proof__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
</style>
