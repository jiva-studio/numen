<script setup lang="ts">
/**
 * One vault's decks, and what each owes today.
 *
 * The whole vault is the ordinary way to sit down to this, so it stands at the
 * top as one button. A deck below it is for the person who came for that deck.
 */
import { Owed } from '@numen/ui'
import { deckName } from './core'
import type { Owing } from './core'

const props = defineProps<{ vault: Owing }>()

defineEmits<{
  (event: 'start', deck: string): void
  (event: 'back'): void
}>()

const owed = () => props.vault.due + props.vault.new
</script>

<template>
  <section class="decks">
    <header class="decks__head">
      <button class="decks__back" type="button" @click="$emit('back')">Vaults</button>
      <h1 class="decks__title">{{ vault.name }}</h1>
    </header>

    <button
      class="decks__all"
      type="button"
      :disabled="owed() === 0"
      @click="$emit('start', '')"
    >
      <span class="decks__all-said">Review everything</span>
      <Owed :waiting="owed()" over />
    </button>

    <p v-if="!vault.decks.length" class="decks__saying">This vault holds no deck.</p>

    <ul v-else class="decks__list">
      <li v-for="deck in vault.decks" :key="deck.deck">
        <button
          class="decks__deck"
          type="button"
          :disabled="deck.due + deck.new === 0"
          @click="$emit('start', deck.deck)"
        >
          <span class="decks__name">{{ deckName(deck.deck) }}</span>
          <Owed :waiting="deck.due + deck.new" />
        </button>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.decks {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-block-size: 0;
  overflow-y: auto;
  gap: var(--numen-inset-wide);
}

.decks__head {
  display: flex;
  align-items: baseline;
  gap: var(--numen-inset);
}

.decks__back {
  padding: 0;
  border: none;
  background: none;
  color: var(--numen-edge-label);
  font: inherit;
  font-size: 0.8125rem;
  cursor: pointer;
}

.decks__back:hover {
  color: var(--numen-node-fg);
}

.decks__back::after {
  content: ' /';
}

.decks__title {
  margin: 0;
  font-size: 1.125rem;
  font-weight: 600;
}

/* The whole vault is the daily act, so it is the one button drawn as one. */
.decks__all {
  display: flex;
  align-items: center;
  padding: var(--numen-inset-wide);
  border: 1px solid var(--numen-focus-border);
  border-radius: var(--numen-radius);
  background: var(--numen-focus-bg);
  color: var(--numen-focus-fg);
  font: inherit;
  gap: var(--numen-inset);
  cursor: pointer;
}

.decks__all:disabled {
  border-color: var(--numen-node-border);
  background: var(--numen-node-bg);
  color: var(--numen-edge-label);
  cursor: default;
}

.decks__all-said {
  flex: 1;
  font-weight: 600;
  text-align: start;
}

.decks__saying {
  margin: 0;
  color: var(--numen-edge-label);
}

.decks__list {
  display: flex;
  margin: 0;
  padding: 0;
  flex-direction: column;
  gap: var(--numen-inset);
  list-style: none;
}

.decks__deck {
  display: flex;
  inline-size: 100%;
  align-items: baseline;
  padding: var(--numen-inset);
  border: 1px solid var(--numen-node-border);
  border-radius: var(--numen-radius);
  background: var(--numen-node-bg);
  color: var(--numen-node-fg);
  font: inherit;
  gap: var(--numen-inset);
  text-align: start;
  cursor: pointer;
}

.decks__deck:hover:not(:disabled) {
  background: var(--numen-highlight);
}

.decks__deck:disabled {
  opacity: 0.55;
  cursor: default;
}

.decks__name {
  flex: 1;
}

</style>
