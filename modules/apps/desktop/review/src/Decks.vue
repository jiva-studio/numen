<script setup lang="ts">
/**
 * One vault's decks, and what each owes today.
 *
 * The whole vault is the ordinary way to sit down to this, so it stands at the
 * top as one button. A deck below it is for the person who came for that deck.
 */
import { computed } from 'vue'
import { Button, Owed } from '@numen/ui'
import { deckName } from './core'
import type { Owing } from './core'

const props = defineProps<{ vault: Owing }>()

defineEmits<{
  (event: 'start', deck: string): void
  (event: 'back'): void
}>()

/** What the whole vault owes: what is due today and what has never been asked. */
const owed = computed(() => props.vault.due + props.vault.new)
</script>

<template>
  <section class="decks">
    <h1 class="decks__title">{{ vault.name }}</h1>

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

    <!-- What a person came here to do stands where the hand is, under the list
         they read: sit down to the whole vault, or go and pick another. -->
    <footer class="decks__deeds">
      <Button variant="ghost" @click="$emit('back')">Another vault</Button>
      <Button class="decks__all" :disabled="owed === 0" @click="$emit('start', '')">
        Review
        <Owed :waiting="owed" bare over />
      </Button>
    </footer>
  </section>
</template>

<style scoped>
.decks {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-block-size: 0;
  gap: var(--numen-inset-wide);
}

.decks__title {
  flex: none;
  margin: 0;
  font-size: var(--numen-title-size);
  font-weight: 600;
}

/* What is done stands at the foot of the screen, whatever is above it. */
.decks__deeds {
  display: flex;
  flex: none;
  margin-block-start: auto;
  align-items: center;
  justify-content: space-between;
  gap: var(--numen-inset);
}

/* The whole vault is the daily act, so it is the one button filled in. It
   stands as tall as the one beside it: two things done from one line. */
.decks__all {
  font-weight: 600;
}

.decks__saying {
  margin: 0;
  color: var(--numen-edge-label);
}

.decks__list {
  display: flex;
  flex: 1;
  margin: 0;
  padding: 0;
  flex-direction: column;
  min-block-size: 0;
  overflow-y: auto;
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
