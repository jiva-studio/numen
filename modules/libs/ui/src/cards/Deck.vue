<script setup lang="ts">
/**
 * A deck, edited: its cards as tiles in a grid, and a plus standing last.
 *
 * The grid lays the cards out, carries one from place to place, and asks which
 * stencil a new one is cut by. What a card holds is the card's own. What a card
 * stands for is the caller's.
 */
import { computed, shallowRef } from 'vue'
import Card from './Card.vue'
import Glyph from './Glyph.vue'
import { useCarry } from './carry'
import { Button } from '../components/ui/button'
import {
  blanks,
  DECK_WORDS,
  grid,
  NOTHING_WRONG,
  sealed,
  type DeckWords,
  type Drawn,
  type Filled,
  type Wrong,
} from './deck'
import { declared, numbered, type Landing } from './order'
import type { Cut } from './stencil'

/** A card nothing is wrong with any value of. */
const NO_FIELDS: ReadonlyMap<string, readonly string[]> = sealed()

const props = withDefaults(
  defineProps<{
    /** The cards, in the order they are drawn. */
    cards: readonly Drawn[]
    /** The stencils a card may be cut by. */
    cuts: readonly Cut[]
    /** What the grid is announced as. */
    name?: string
    /** What the caller found wrong with the cards it handed in. */
    wrong?: Wrong
    /** The words it is drawn with. */
    words?: DeckWords
  }>(),
  { name: 'Deck', wrong: () => NOTHING_WRONG, words: () => DECK_WORDS },
)

const emit = defineEmits<{
  /**
   * A card asked for, cut by a stencil, named by something nothing has taken.
   * The name is the first field's value and stands nowhere among the rest,
   * which stand empty.
   */
  (event: 'add', name: string, stencil: string, filled: readonly Filled[]): void
  (event: 'remove', id: string): void
  /** A card let go somewhere in the order: before another, or at the end. */
  (event: 'move', id: string, at: Landing): void
  /**
   * One value of one card as it now reads. `names` says the card's name was
   * typed in, and a card writing a field twice is writing two values, of which
   * `nth` says which was typed in.
   */
  (event: 'write', id: string, field: string, nth: number, names: boolean, text: string): void
}>()

/** The plus is showing which stencils a new card may be cut by. */
const asking = shallowRef(false)

/**
 * The card under the pointer's hand, and where letting go would put it. A card
 * let go where it stands moves nothing, and nothing else among them is fixed.
 */
const { carried, at, lift, over, release, drop, step } = useCarry<Landing>({
  order: () => props.cards.map((card) => card.id),
  nowhere: null,
  lands: (held, at) => at !== held,
  moves: (held, at) => emit('move', held, at),
})

const shown = computed(() => grid(props.cards, props.cuts, carried.value))

const add = (cut: Cut): void => {
  asking.value = false
  emit(
    'add',
    numbered(props.cards.map((card) => card.name), props.words.cardStem),
    cut.name,
    blanks(declared(cut.fields).slice(1)),
  )
}
</script>

<template>
  <div
    class="deck numen bg-surface font-sans text-base text-ink"
    role="group"
    :aria-label="name"
    @dragover="over(null, $event)"
    @drop="drop"
  >
    <div class="deck__grid">
      <div
        v-for="tile in shown.tiles"
        :key="tile.id"
        class="deck__tile"
        :data-before="tile.id === at || undefined"
        @dragover.stop="over(tile.id, $event)"
        @drop.stop="drop"
      >
        <Card
          :tile="tile"
          :wrong="wrong.at.get(tile.id) ?? []"
          :wrong-under="wrong.under.get(tile.id) ?? NO_FIELDS"
          :words="words"
          @remove="emit('remove', tile.id)"
          @lift="lift(tile.id, $event)"
          @release="release"
          @step="(way, press) => step(tile.id, way, press)"
          @write="(field, nth, names, text) => emit('write', tile.id, field, nth, names, text)"
        />
      </div>

      <article
        class="deck__tile deck__plus rounded-node"
        :aria-posinset="shown.plusAt"
        :aria-setsize="shown.of"
        :aria-label="words.add"
        data-plus
      >
        <!-- The plus says what it is for by standing alone in the middle. What
             it is called is read aloud and shown on hovering, and not beside it.
             What it opens takes its place, so it says nothing of being open. -->
        <Button
          v-if="!asking"
          variant="ghost"
          class="deck__ask"
          :aria-label="words.add"
          :title="words.add"
          @click="asking = true"
        >
          <Glyph shows="plus" />
        </Button>

        <div v-else class="deck__asking flex flex-col items-center">
          <p class="deck__silence caps-numen text-small text-hushed">{{ words.cut }}</p>
          <div class="deck__cuts flex flex-wrap justify-center">
            <Button
              v-for="cut in cuts"
              :key="cut.name"
              variant="outline"
              size="small"
              :data-cut="cut.name"
              @click="add(cut)"
              >{{ cut.name }}</Button
            >
          </div>
        </div>
      </article>
    </div>
  </div>
</template>

<style scoped>
/* The deck is what scrolls. The grid inside it is as tall as its rows, which is
   what lets every row take the height of the tallest tile of the whole deck. */
.deck {
  --tile: 20rem;
  --gap: 0.75rem;

  padding: var(--numen-gutter);
  overflow: auto;
}

/*
 * Tiles stand in as many columns as fit at the narrowest one is drawn, and one
 * column where there is not the width for two. Every row is one flexible track,
 * so all of them come to the height of the fullest tile and no tile is drawn to
 * a size of its own.
 */
.deck__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(var(--tile), 100%), 1fr));
  grid-auto-rows: 1fr;
  gap: var(--gap);
}

/* A tile is the room one card is drawn in, and the card fills it. */
.deck__tile {
  position: relative;
  display: grid;
  min-inline-size: 0;
}

.deck__tile[data-before]::before {
  content: '';
  position: absolute;
  inset-block: 0;
  inset-inline-start: calc(-1 * var(--gap) / 2);
  inline-size: var(--numen-caret);
  background: var(--numen-ring);
}

/* The plus is an outline until it is pressed: it holds nothing yet. What it
   holds stands in the middle, which is where it was pressed. */
.deck__plus {
  --plus: 2.5rem;

  display: grid;
  place-items: center;
  min-block-size: 6rem;
  padding: var(--numen-box-air);
  border: var(--numen-stroke) dashed var(--numen-node-border);
}

/* A mark is drawn at the height of the text around it, so the plus is as large
   as the type the button is set in. */
.deck__ask {
  block-size: auto;
  inline-size: auto;
  padding: var(--numen-inset);
  font-size: var(--plus);
}

.deck__asking {
  gap: var(--numen-inset);
}

.deck__cuts {
  gap: var(--numen-inset);
}

.deck__silence {
  margin: 0;
}
</style>
