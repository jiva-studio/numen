<script setup lang="ts">
/**
 * A deck of flashcards in sections: the review preset that schedules it, and each card with the fields
 * its model fills in.
 */
import { DECK_PRESET, INITIAL_CARDS, INITIAL_SECTIONS } from '../lib/deck'
import DeckCardView from './DeckCardView.vue'

/* -------------------------------- Helpers --------------------------------- */
const getCardsIn = (section: string) => INITIAL_CARDS.filter((one) => one.section === section)
</script>

<template>
  <div class="deck">
    <p class="deck__preset">
      <span class="deck__label">Scheduled by</span>
      <span>{{ DECK_PRESET }}</span>
    </p>

    <section v-for="section in INITIAL_SECTIONS" :key="section.id" class="deck__section">
      <h2 class="deck__title">{{ section.name }}</h2>

      <DeckCardView v-for="card in getCardsIn(section.id)" :key="card.id" :card="card" />
    </section>
  </div>
</template>

<style scoped>
.deck {
  block-size: 100%;
  overflow-y: auto;
  padding: 0.75rem var(--numen-gutter) 1.5rem;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-reading-size);
  color: var(--numen-ink);
  scrollbar-width: none;
}

.deck::-webkit-scrollbar {
  display: none;
}

.deck__preset {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0;
}


.deck__label {
  white-space: nowrap;
  color: var(--numen-hushed);
}

.deck__title {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin: 2rem 0 1rem;
  font-size: inherit;
  font-weight: 500;
}

.deck__title::before,
.deck__title::after {
  flex: 1;
  content: '';
  border-block-start: var(--numen-stroke) dashed var(--numen-rule);
}
</style>
