<script setup lang="ts">
/**
 * A deck of flashcards in sections: the review preset that schedules it, and each card with the fields
 * its model fills in.
 */
import { computed } from 'vue'
import { Plus } from '@lucide/vue'
import { Select } from '@numen/ui'
import type { Deck } from '../model/deck'
import DeckCardView from './DeckCardView.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ deck: Deck }>()

/* --------------------------------- State ---------------------------------- */
const choices = computed(() => props.deck.review.presets.map((one) => ({ id: one.path, text: one.name })))
const preset = computed({
  get: () => props.deck.review.preset.value.path,
  set: (path: string) => props.deck.review.setPreset(path),
})

/* -------------------------------- Helpers --------------------------------- */
const getCardsIn = (section: string) => props.deck.cards.value.filter((one) => one.section === section)
</script>

<template>
  <div class="deck">
    <label class="deck__preset">
      <span class="deck__label">Scheduled by</span>
      <span class="deck__select"><Select v-model="preset" :choices="choices" name="Review preset" /></span>
    </label>

    <section v-for="section in deck.sections" :key="section.id" class="deck__section">
      <h2 class="deck__title">{{ section.name }}</h2>

      <DeckCardView
        v-for="card in getCardsIn(section.id)"
        :key="card.id"
        :card="card"
        @remove="deck.removeCard(card.id)"
        @write="(field, text) => deck.writeField(card.id, field, text)"
      />

      <button type="button" class="deck__add" @click="deck.addCard('Concept', section.id)">
        <Plus class="deck__icon" />
        Add card
      </button>
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
}

.deck__select {
  display: block;
  inline-size: 8rem;
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

.deck__add {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.25rem 0;
  border: 0;
  background: none;
  color: var(--numen-hushed);
  font: inherit;
  cursor: pointer;
}

.deck__icon {
  inline-size: 0.875rem;
  block-size: 0.875rem;
  stroke-width: 1.875;
}
</style>
