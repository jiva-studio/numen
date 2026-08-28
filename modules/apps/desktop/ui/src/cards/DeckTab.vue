<script setup lang="ts">
/**
 * A deck tab: the cards as a grid under the sections they stand in, and the
 * questions the file puts.
 *
 * The grid takes cards and hands identities back, and is handed what is wrong
 * with each of them under the same identities, so a card under no stencil and
 * two cards of one mark are each marked where they were read from.
 */
import { computed } from 'vue'
import { Deck as DeckView } from '@numen/ui'
import type { CardLanding, Filled } from '@numen/ui'
import type { Held } from './deck'
import { WORDS as words } from './words'

const props = defineProps<{ held: Held }>()

const drawn = computed(() => props.held.drawn())
const marks = computed(() => props.held.marks())

/** What the grid draws against the cards it was handed. */
const wrong = computed(() => ({ at: marks.value.at, under: marks.value.under }))
</script>

<template>
  <div class="deck-tab">
    <p v-if="props.held.saying()" role="alert" class="warning">{{ props.held.saying() }}</p>

    <p v-if="props.held.shown().state === 'gone'" role="status" class="warning answering">
      {{ words.gone }}
      <button type="button" class="answering__answer" @click="props.held.keep()">
        {{ words.makeAgain }}
      </button>
    </p>
    <p v-if="props.held.shown().state === 'overtaken'" role="status" class="warning answering">
      {{ words.overtaken }}
      <button type="button" class="answering__answer" @click="props.held.keep()">
        {{ words.keep }}
      </button>
      <button type="button" class="answering__answer" @click="props.held.take()">
        {{ words.take }}
      </button>
    </p>

    <ul v-if="marks.whole.length" class="warning wrong" :aria-label="words.problems">
      <li v-for="(text, at) in marks.whole" :key="at">{{ text }}</li>
    </ul>

    <DeckView
      class="deck-tab__grid"
      :cards="drawn"
      :sections="props.held.bands()"
      :cuts="props.held.cuts()"
      :name="words.deck"
      :wrong="wrong"
      @add="
        (stencil: string, filled: readonly Filled[]) => props.held.adds(stencil, filled)
      "
      @remove="(id: string) => props.held.removes(id)"
      @move="(id: string, at: CardLanding) => props.held.moves(id, at)"
      @write="
        (id: string, field: string, nth: number, text: string) =>
          props.held.writes(id, field, nth, text)
      "
      @add-section="(name: string) => props.held.addsSection(name)"
      @rename-section="(id: string, name: string) => props.held.namesSection(id, name)"
      @remove-section="(id: string) => props.held.removesSection(id)"
    />

  </div>
</template>

<style scoped>
/* The grid takes what the bands above it leave, and scrolls inside itself. */
.deck-tab {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
}

.deck-tab__grid {
  flex: 1;
  min-block-size: 0;
}

/* A warning carries a filesystem path, and a long one breaks where it stands. */
.warning {
  margin: 0;
  padding: 0.4rem 1rem;
  font-family: var(--numen-font-sans);
  font-size: calc(var(--numen-font-size) * 12.8 / 13);
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  overflow-wrap: break-word;
}

/* The question the file holds: the two answers on the line the sentence is on. */
.answering {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0 0.9rem;
}

.answering__answer {
  padding: 0;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  text-decoration: underline;
  text-underline-offset: 0.15em;
  cursor: pointer;
}

.answering__answer:focus-visible {
  outline: 1px solid currentColor;
  outline-offset: 2px;
}

.wrong {
  margin: 0;
  padding-inline-start: 1.1rem;
  list-style: disc;
}

/* The mark stands inside a tile, in the alarm the tile draws its own in. */
.wrong--tile {
  padding-block: 0;
  color: var(--numen-alarm);
  font-size: calc(var(--numen-font-size) * 12 / 13);
  overflow-wrap: anywhere;
}
</style>
