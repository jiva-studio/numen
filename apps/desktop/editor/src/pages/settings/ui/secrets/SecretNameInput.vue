<script setup lang="ts">
/**
 * The field a secret's name is typed into. It emits `commit` on Enter, on a change and on leaving
 * the field, and `cancel` on Escape.
 */
import { WORDS as words } from '../../words'

defineProps<{ modelValue: string }>()

const emit = defineEmits<{
  (e: 'update:modelValue', name: string): void
  (e: 'commit'): void
  (e: 'cancel'): void
}>()
</script>

<template>
  <input
    :value="modelValue"
    type="text"
    :aria-label="words.secretRenameName"
    class="secret-name"
    @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    @keydown.enter="emit('commit')"
    @keydown.esc="emit('cancel')"
    @change="emit('commit')"
    @blur="emit('commit')"
  />
</template>

<style scoped>
.secret-name {
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

.secret-name:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}
</style>
