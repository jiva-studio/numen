<script setup lang="ts">
/**
 * A masked field a value is entered into. The value is sent once, on Enter, on leaving the field
 * or on a change, and the field is cleared as it is sent: nothing here keeps a value or shows one.
 */
import { ref } from 'vue'

/* ----------------------------- Props & Emits ------------------------------ */
defineProps<{
  label: string
  placeholder: string
}>()

const emit = defineEmits<{
  (e: 'commit', value: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const entry = ref('')

/* -------------------------------- Handlers -------------------------------- */
function onCommit(): void {
  const value = entry.value.trim()
  if (value === '') return
  entry.value = ''
  emit('commit', value)
}
</script>

<template>
  <input
    v-model="entry"
    type="password"
    autocomplete="off"
    spellcheck="false"
    :aria-label="label"
    :placeholder="placeholder"
    class="secret-value"
    @blur="onCommit"
    @change="onCommit"
    @keydown.enter="onCommit"
  />
</template>

<style scoped>
.secret-value {
  box-sizing: border-box;
  inline-size: 100%;
  min-inline-size: 0;
  block-size: var(--numen-action-size);
  padding: 0.25rem 0.625rem;
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
}

.secret-value:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}
</style>
