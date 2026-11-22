<script setup lang="ts">
/**
 * The field the twelve recovery words of an identity are typed into to restore it.
 */
import { WORDS as words } from '../../words'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ text: string; isDisabled: boolean }>()

const emit = defineEmits<{
  (e: 'change', value: string): void
}>()

/* -------------------------------- Handlers -------------------------------- */
function onInput(event: Event): void {
  const target = event.target as HTMLTextAreaElement | null
  emit('change', target?.value ?? '')
}
</script>

<template>
  <textarea
    class="restore-words"
    rows="3"
    autocomplete="off"
    autocapitalize="off"
    spellcheck="false"
    :value="text"
    :disabled="isDisabled"
    :placeholder="words.identityRestorePlaceholder"
    :aria-label="words.identityRestoreLabel"
    @input="onInput"
  />
</template>

<style scoped>
.restore-words {
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

.restore-words:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}
</style>
