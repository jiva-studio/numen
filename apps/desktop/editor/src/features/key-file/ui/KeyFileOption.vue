<script setup lang="ts">
/** One way to keep the keys, drawn as a card with a radio: its title and, when it has one, a detail. */

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  value: 'passphrase' | 'plaintext'
  isOn: boolean
  title: string
  detail?: string
}>()

const emit = defineEmits<{ (e: 'pick', value: 'passphrase' | 'plaintext'): void }>()

/* -------------------------------- Handlers -------------------------------- */
function onChange(): void {
  emit('pick', props.value)
}
</script>

<template>
  <label class="key-file-option" :class="{ 'key-file-option--on': isOn }">
    <input type="radio" name="key-file-mode" :value="value" :checked="isOn" @change="onChange" />
    <span class="key-file-option__title">{{ title }}</span>
    <span v-if="detail" class="key-file-option__detail">{{ detail }}</span>
  </label>
</template>

<style scoped>
.key-file-option {
  display: grid;
  grid-template-columns: auto 1fr;
  column-gap: 0.625rem;
  row-gap: 0.125rem;
  align-items: start;
  padding: 0.625rem 0.75rem;
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  cursor: pointer;
}

.key-file-option--on {
  background: var(--numen-selection);
  border-color: var(--numen-accent);
}

.key-file-option input {
  grid-row: 1 / span 2;
}

.key-file-option__title {
  grid-column: 2;
  font-weight: 500;
}

.key-file-option__detail {
  grid-column: 2;
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
}
</style>
