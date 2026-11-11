<script setup lang="ts">
/** One card of a deck: the model it is written in, and the text of each of the model's fields. */
import type { DeckCard } from '../lib/deck'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ card: DeckCard }>()
</script>

<template>
  <article class="deck__card">
    <header class="deck__model">{{ card.cardModel }}</header>
    <p v-for="one in card.filled" :key="one.field" class="deck__field">
      <span class="deck__name">{{ one.field }}</span>
      <span class="deck__value">{{ one.text }}</span>
    </p>
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
  padding: 0.25rem;
  background: var(--numen-table-head-bg);
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  text-align: center;
}

.deck__field {
  display: block;
  margin: 0;
  padding: 0.5rem 1.25rem 0.75rem;
}

.deck__name {
  display: block;
  padding-block-end: 0.25rem;
  border-block-end: var(--numen-stroke) dashed var(--numen-rule);
  font-size: var(--numen-text-2);
}

.deck__value {
  display: block;
  margin-block-start: 0.5rem;
}
</style>
