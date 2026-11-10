<script setup lang="ts">
/**
 * A deck of flashcards in sections: the review preset that schedules it, and each card with the fields
 * its model fills in.
 */
import { computed } from 'vue'
import { Plus, X } from '@lucide/vue'
import { Select } from '@numen/ui'
import type { Deck } from '../model/deck'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ deck: Deck }>()

/* --------------------------------- State ---------------------------------- */
const choices = computed(() => props.deck.presets.map((one) => ({ id: one.path, text: one.name })))
const preset = computed({
  get: () => props.deck.preset.value.path,
  set: (path: string) => props.deck.setPreset(path),
})

/* -------------------------------- Helpers --------------------------------- */
const getCardsIn = (section: string) => props.deck.cards.value.filter((one) => one.section === section)

function onWrite(card: string, field: string, event: Event) {
  props.deck.writeField(card, field, (event.target as HTMLInputElement).value)
}
</script>

<template>
  <div class="deck">
    <label class="deck__preset">
      <span class="deck__label">Scheduled by</span>
      <span class="deck__select"><Select v-model="preset" :choices="choices" name="Review preset" /></span>
    </label>

    <section v-for="section in deck.sections" :key="section.id" class="deck__section">
      <h2 class="deck__title">{{ section.name }}</h2>

      <article v-for="card in getCardsIn(section.id)" :key="card.id" class="deck__card">
        <header class="deck__model">
          {{ card.cardModel }}
          <button type="button" class="deck__remove" aria-label="Remove card" @click="deck.removeCard(card.id)">
            <X class="deck__icon" />
          </button>
        </header>
        <label v-for="one in card.filled" :key="one.field" class="deck__field">
          <span class="deck__name">{{ one.field }}</span>
          <input class="deck__value" :value="one.text" spellcheck="false" @input="onWrite(card.id, one.field, $event)" />
        </label>
      </article>

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

.deck__label,
.deck__name {
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
