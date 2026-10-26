<script setup lang="ts">
/**
 * One vault's decks, and what each owes today.
 *
 * top as one button. A deck below it is for the person who came for that deck.
 */
import { computed } from 'vue'
import { Button, KeyCap } from '@numen/ui'
import { DueCount } from '../../../../ui/cards'
import Progress from './Progress.vue'
import Presets from './Presets.vue'
import { DeckRow } from './deck-row'
import { canStart } from '../lib/progress'
import type { DecksProps } from './decks.types'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<DecksProps>()

/* --------------------------------- Events --------------------------------- */
defineEmits<{
  (event: 'start', deck: string): void
  (event: 'start-preset', preset: string): void
  (event: 'back'): void
}>()

/* --------------------------------- State ---------------------------------- */
/** The whole vault's count: what is due today and what has never been asked. */
const allDue = computed(() => props.vault.due + props.vault.new)
</script>

<template>
  <section class="decks">
    <h1 class="decks__title">{{ vault.name }}</h1>

    <Progress :days="days" :due="due" />

    <Presets :presets="presets" :today="today" @start="$emit('start-preset', $event)" />

    <p v-if="!vault.decks.length" class="decks__saying">This vault holds no deck.</p>

    <ul v-else class="decks__list">
      <li v-for="(deck, index) in vault.decks" :key="deck.deck">
        <DeckRow
          :deck="deck"
          :index="index"
          :preset="byDeck.get(deck.deck)"
          :has-presets="hasPresets"
          :is-counted="vault.isCounted"
          :can-open="canStart(deck, byDeck)"
          @start="$emit('start', deck.deck)"
        />
      </li>
    </ul>

    <!-- What a person came here to do stands where the hand is, under the list
         they read: sit down to the whole vault, or go and pick another. -->
    <footer class="decks__actions">
      <Button variant="ghost" @click="$emit('back')">
        <KeyCap :keys="{ icons: [], letter: 'esc' }" />
        Another vault
      </Button>
      <Button class="decks__all" :disabled="allDue === 0" @click="$emit('start', '')">
        <KeyCap :keys="{ icons: [], letter: 'enter' }" />
        Review
        <DueCount :due="allDue" is-bare is-on-button />
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

.decks__actions {
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
  color: var(--numen-hushed);
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
</style>
