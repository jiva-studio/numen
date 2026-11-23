<script setup lang="ts">
/**
 * The two fields the words of two sheets of a recovery kit are typed or pasted into.
 */
import { WORDS as words } from '../../words'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ texts: readonly [string, string]; isDisabled: boolean }>()

const emit = defineEmits<{
  (e: 'change', index: number, value: string): void
}>()

/* -------------------------------- Handlers -------------------------------- */
function onInput(index: number, event: Event): void {
  const target = event.target as HTMLTextAreaElement | null
  emit('change', index, target?.value ?? '')
}
</script>

<template>
  <div class="sheets-input">
    <textarea
      v-for="number in [1, 2]"
      :key="number"
      class="sheets-input__field"
      rows="4"
      autocomplete="off"
      autocapitalize="off"
      spellcheck="false"
      :value="texts.at(number - 1) ?? ''"
      :disabled="isDisabled"
      :placeholder="words.kitSheetPlaceholder"
      :aria-label="words.kitSheetField(number)"
      @input="onInput(number - 1, $event)"
    />
  </div>
</template>

<style scoped>
.sheets-input {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.sheets-input__field {
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

.sheets-input__field:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}
</style>
