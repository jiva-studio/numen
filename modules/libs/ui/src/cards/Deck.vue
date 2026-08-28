<script setup lang="ts">
/**
 * A deck, edited: its cards as tiles in a grid, standing under the sections
 * they are in, and a plus standing last.
 *
 * The grid lays the cards out, carries one from place to place, and asks which
 * stencil a new one is cut by. What a card holds is the card's own. What a card
 * stands for is the caller's.
 */
import { computed, shallowRef } from 'vue'
import Band from './Band.vue'
import Card from './Card.vue'
import Glyph from './Glyph.vue'
import Rule from '../rule/Rule.vue'
import { useCarry } from './carry'
import { Button } from '../components/ui/button'
import {
  blanks,
  DECK_WORDS,
  grid,
  NOTHING_WRONG,
  sealed,
  type Banded,
  type DeckWords,
  type Drawn,
  type Filled,
  type Run,
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
    /** The sections, in the order they stand in the deck. */
    sections?: readonly Banded[]
    /** What the grid is announced as. */
    name?: string
    /** What the caller found wrong with the cards it handed in. */
    wrong?: Wrong
    /** The words it is drawn with. */
    words?: DeckWords
  }>(),
  { sections: () => [], name: 'Deck', wrong: () => NOTHING_WRONG, words: () => DECK_WORDS },
)

const emit = defineEmits<{
  /**
   * A card asked for, cut by a stencil, with a value standing empty under every
   * field that stencil declares. It is made at the end of the deck.
   */
  (event: 'add', stencil: string, filled: readonly Filled[]): void
  (event: 'remove', mark: string): void
  /**
   * A card let go somewhere in the deck: before the card of that mark, at the
   * head of the section of that identity, or at the end.
   */
  (event: 'move', mark: string, at: Landing): void
  /**
   * One value of one card as it now reads. A card writing a field twice is
   * writing two values, of which `nth` says which was typed in.
   */
  (event: 'write', mark: string, field: string, nth: number, text: string): void
  /** A section asked for, under a name nothing has taken. It is made at the end. */
  (event: 'add-section', name: string): void
  (event: 'rename-section', id: string, name: string): void
  /** A section asked to go. Its heading goes, and the cards under it stay. */
  (event: 'remove-section', id: string): void
}>()

/** The plus is showing which stencils a new card may be cut by. */
const asking = shallowRef(false)

/**
 * The card under the pointer's hand, and where letting go would put it. A card
 * let go where it stands moves nothing, and nothing else among them is fixed.
 */
const { carried, at, lift, over, release, drop, step } = useCarry<Landing>({
  order: () => props.cards.map((card) => card.mark),
  nowhere: null,
  lands: (held, at) => at !== held,
  moves: (held, at) => emit('move', held, at),
})

const shown = computed(() => grid(props.cards, props.sections, props.cuts, carried.value))

/** The run the plus stands in, which is the last of them. */
const last = computed<Run | undefined>(() => shown.value.runs.at(-1))

const add = (cut: Cut): void => {
  asking.value = false
  emit('add', cut.name, blanks(declared(cut.fields)))
}

const addSection = (): void => {
  emit('add-section', numbered(props.sections.map((each) => each.name), props.words.sectionStem))
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
    <template v-for="run in shown.runs" :key="run.band?.id ?? ''">
      <!-- A card let go on a section's heading lands at the head of that
           section, which is the one place a section holding none takes one. -->
      <div
        v-if="run.band"
        class="deck__band"
        :data-band="run.band.id"
        :data-before="run.band.id === at || undefined"
        @dragover.stop="over(run.band.id, $event)"
        @drop.stop="drop"
      >
        <Band
          :band="run.band"
          :words="words"
          @rename="(name: string) => emit('rename-section', run.band?.id ?? '', name)"
          @remove="emit('remove-section', run.band?.id ?? '')"
        />
      </div>

      <div class="deck__grid">
        <div
          v-for="tile in run.tiles"
          :key="tile.mark"
          class="deck__tile"
          :data-before="tile.mark === at || undefined"
          @dragover.stop="over(tile.mark, $event)"
          @drop.stop="drop"
        >
          <Card
            :tile="tile"
            :wrong="wrong.at.get(tile.mark) ?? []"
            :wrong-under="wrong.under.get(tile.mark) ?? NO_FIELDS"
            :words="words"
            @remove="emit('remove', tile.mark)"
            @lift="lift(tile.mark, $event)"
            @release="release"
            @step="(way, press) => step(tile.mark, way, press)"
            @write="(field, nth, text) => emit('write', tile.mark, field, nth, text)"
          />
        </div>

        <!-- A card is made at the end of the deck, so the plus stands in the
             last run of it and nowhere else. -->
        <article
          v-if="run === last"
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
    </template>

    <Rule>
      <Button variant="ghost" size="small" data-add-section @click="addSection">
        <Glyph shows="plus" />
        {{ words.addSection }}
      </Button>
    </Rule>
  </div>
</template>

<style scoped>
/* The deck is what scrolls. The grid inside it is as tall as its rows, which is
   what lets every row take the height of the tallest tile of the whole deck. */
.deck {
  --tile: 20rem;
  --gap: 0.75rem;

  display: flex;
  flex-direction: column;
  gap: var(--gap);
  padding: var(--numen-gutter);
  overflow: auto;
}

/* A section's heading runs the width of the grid under it, and takes the caret
   that says a card would land at its head. */
.deck__band {
  position: relative;
}

.deck__band[data-before]::before {
  content: '';
  position: absolute;
  inset-inline: 0;
  inset-block-start: calc(-1 * var(--gap) / 2);
  block-size: var(--numen-caret);
  background: var(--numen-ring);
}

/* A section holding no card takes no room of its own between the headings. */
.deck__grid:empty {
  display: none;
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
