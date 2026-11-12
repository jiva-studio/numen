<script setup lang="ts">
/**
 * The text field of the page of secrets, filling its cell. Attributes and listeners fall through to
 * the input.
 */
import { useTemplateRef } from 'vue'

const model = defineModel<string>({ default: '' })

defineExpose({
  focus: () => input.value?.focus(),
  select: () => input.value?.select(),
})

const input = useTemplateRef<HTMLInputElement>('input')
</script>

<template>
  <input ref="input" v-model="model" class="secret-input" />
</template>

<style scoped>
.secret-input {
  --pad: 0.25rem 0.625rem;
  box-sizing: border-box;
  inline-size: 100%;
  min-inline-size: 0;
  block-size: var(--numen-action-size);
  padding: var(--pad);
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
}

.secret-input:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.secret-input[aria-invalid='true'] {
  border-color: var(--numen-alarm);
}
</style>
