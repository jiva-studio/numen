<script setup lang="ts">
/** A field for a passphrase: the text is hidden and the browser is told not to keep it. */
import { useId } from 'vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ label: string; value: string; isDisabled: boolean; isInvalid?: boolean }>()

const emit = defineEmits<{ (e: 'change', value: string): void }>()

/* --------------------------------- State ---------------------------------- */
const id = useId()

/* -------------------------------- Handlers -------------------------------- */
function onInput(event: Event): void {
  const target = event.target as HTMLInputElement | null
  emit('change', target?.value ?? '')
}
</script>

<template>
  <div class="passphrase-field">
    <label :for="id" class="passphrase-field__label">{{ label }}</label>
    <input
      :id="id"
      class="passphrase-field__input"
      type="password"
      autocomplete="new-password"
      autocapitalize="off"
      spellcheck="false"
      :value="value"
      :disabled="isDisabled"
      :aria-invalid="isInvalid || undefined"
      @input="onInput"
    />
  </div>
</template>

<style scoped>
.passphrase-field {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.passphrase-field__label {
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
}

.passphrase-field__input {
  box-sizing: border-box;
  appearance: none;
  inline-size: 100%;
  margin: 0;
  padding: 0.5rem 0.75rem;
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font: inherit;
  font-size: var(--numen-text-2);
}

.passphrase-field__input:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.passphrase-field__input[aria-invalid='true'] {
  border-color: var(--numen-alarm);
}
</style>
