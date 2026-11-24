<script setup lang="ts">
/** A labelled input in the style of the fields of the identity dialogs. */
import { useId } from 'vue'

/* ----------------------------- Props & Emits ------------------------------ */
defineProps<{
  label: string
  modelValue: string
  isReadonly?: boolean
  isMono?: boolean
  inputmode?: 'numeric' | 'text'
  maxlength?: number
}>()

const emit = defineEmits<{ (e: 'update:modelValue', value: string): void }>()

/* --------------------------------- State ---------------------------------- */
const inputId = useId()

/* -------------------------------- Handlers -------------------------------- */
function onInput(event: Event): void {
  emit('update:modelValue', (event.target as HTMLInputElement | null)?.value ?? '')
}
</script>

<template>
  <div class="link-field">
    <label class="link-field__label" :for="inputId">{{ label }}</label>
    <input
      :id="inputId"
      class="link-field__input"
      :class="{ 'link-field__input--mono': isMono }"
      type="text"
      autocomplete="off"
      autocapitalize="off"
      spellcheck="false"
      :readonly="isReadonly"
      :inputmode="inputmode"
      :maxlength="maxlength"
      :value="modelValue"
      @input="onInput"
    />
  </div>
</template>

<style scoped>
.link-field {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.link-field__label {
  color: var(--numen-hushed);
}

.link-field__input {
  box-sizing: border-box;
  appearance: none;
  inline-size: 100%;
  margin: 0;
  padding: 0.5rem 0.75rem;
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  background-color: var(--numen-field-bg);
  color: var(--numen-ink);
  font: inherit;
}

.link-field__input--mono {
  font-family: var(--numen-font-mono);
}

.link-field__input:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}
</style>
