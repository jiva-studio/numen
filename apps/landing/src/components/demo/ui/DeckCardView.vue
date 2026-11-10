<script setup lang="ts">
/** One card of a deck: the model it is written in, and a field to write for each of the model's fields. */
import { X } from '@lucide/vue'
import type { DeckCard } from '../lib/deck'

/* ----------------------------- Props & Emits ------------------------------ */
defineProps<{ card: DeckCard }>()

const emit = defineEmits<{
  remove: []
  write: [field: string, text: string]
}>()

/* -------------------------------- Handlers -------------------------------- */
function onWrite(field: string, event: Event) {
  if (event.target instanceof HTMLInputElement) emit('write', field, event.target.value)
}
</script>

<template>
  <article class="deck__card">
    <header class="deck__model">
      {{ card.cardModel }}
      <button type="button" class="deck__remove" aria-label="Remove card" @click="emit('remove')">
        <X class="deck__icon" />
      </button>
    </header>
    <label v-for="one in card.filled" :key="one.field" class="deck__field">
      <span class="deck__name">{{ one.field }}</span>
      <input class="deck__value" :value="one.text" spellcheck="false" @input="onWrite(one.field, $event)" />
    </label>
  </article>
</template>

<style scoped>
.deck__card {
  margin-block-end: 1rem;
  overflow: hidden;
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius-tight);
  background: var(--numen-raised);
}

.deck__model {
  position: relative;
  padding: 0.25rem;
  background: var(--numen-table-head-bg);
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  text-align: center;
}

.deck__remove {
  position: absolute;
  inset-block: 0;
  inset-inline-end: 0.35rem;
  display: grid;
  place-items: center;
  padding: 0;
  border: 0;
  background: none;
  color: inherit;
  cursor: pointer;
}

.deck__field {
  display: block;
  padding: 0.5rem 1.25rem 0.75rem;
}

.deck__name {
  display: block;
  padding-block-end: 0.25rem;
  border-block-end: var(--numen-stroke) dashed var(--numen-rule);
  font-size: var(--numen-text-2);
}

.deck__value {
  inline-size: 100%;
  margin-block-start: 0.5rem;
  padding: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: inherit;
  font: inherit;
}

.deck__icon {
  inline-size: 0.875rem;
  block-size: 0.875rem;
  stroke-width: 1.875;
}
</style>
