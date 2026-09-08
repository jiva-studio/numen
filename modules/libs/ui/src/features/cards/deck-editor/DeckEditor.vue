<script setup lang="ts">
/**
 * A deck, edited: its cards as tiles in a grid, standing under the sections
 * they are in, and a plus standing last.
 *
 * The grid lays the cards out, drags one from place to place, and asks which
 * stencil a new one is cut by. What a card holds is the card's own. What a card
 * stands for is the caller's.
 */
import { computed, shallowRef } from 'vue'
import { SectionHeading } from './section-heading'
import { Card } from './card'
import { Icon } from '../icon'
import { Divider } from '../divider'
import { useDrag } from '../drag'
import { Button } from '@/shared/ui/button'
import {
  DECK_WORDS,
  endOf,
  grid,
  HEAD,
  lands,
  NOTHING_WRONG,
  type DeckSection,
  type DeckWords,
  type DeckCard,
  type Run,
  type Wrong,
} from '../deck'
import { numbered, sealed, type InsertionPoint } from '../order'
import type { Stencil } from '../card'

/** A card nothing is wrong with any value of. */
const NO_FIELDS: ReadonlyMap<string, readonly string[]> = sealed()

const props = withDefaults(
  defineProps<{
    /** The cards, in the order they are drawn. */
    cards: readonly DeckCard[]
    /** The stencils a card may be cut by. */
    stencils: readonly Stencil[]
    /** The sections, in the order they stand in the deck. */
    sections?: readonly DeckSection[]
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
   * A card asked for, cut by the stencil of that name. It is made at the end of
   * the section its plus stands in, and nothing names the cards before the
   * first section. What a new card holds is the caller's.
   */
  (event: 'add', stencil: string, section: string | null): void
  (event: 'remove', card: string): void
  /**
   * A card let go somewhere in the deck: before the card of that identity, at
   * the head of the deck, at the head of the section of that identity, past
   * the last card standing under a heading, or at the end.
   */
  (event: 'move', card: string, at: InsertionPoint): void
  /**
   * One value of one card as it now reads. A card writing a field twice is
   * writing two values, of which `nth` says which was typed in.
   */
  (event: 'write', card: string, field: string, nth: number, text: string): void
  /** A section asked for, under a name nothing has taken. It is made at the end. */
  (event: 'add-section', name: string): void
  (event: 'rename-section', id: string, name: string): void
  /** A section asked to go. Its heading goes, and the cards under it stay. */
  (event: 'remove-section', id: string): void
}>()

/**
 * The run whose plus is showing which stencils a new card may be cut by, and
 * nothing while none of them is. One plus asks at a time.
 */
const asking = shallowRef<string | null>(null)

/**
 * The card under the pointer's hand, and where letting go would put it. A card
 * let go where it stands moves nothing, and nothing else among them is fixed.
 *
 * The head of the deck stands first in the order, so the card at the top of the
 * first section is dragged out of it by the keyboard as it is by the pointer.
 */
const { dragged, at, lift, over, release, drop, step } = useDrag<InsertionPoint | undefined>({
  order: () => [HEAD, ...props.cards.map((card) => card.id)],
  nowhere: undefined,
  lands: (held: string, at: InsertionPoint): boolean => lands(shown.value.runs, held, at),
  moves: (held, at) => emit('move', held, at),
})

const shown = computed(() => grid(props.cards, props.sections, props.stencils, dragged.value))

/** Where the plus of a run stands in the order: past everything under it. */
const after = (run: Run): InsertionPoint => endOf(run.id)

const add = (stencil: Stencil, run: Run): void => {
  asking.value = null
  emit('add', stencil.name, run.section?.id ?? null)
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
    @dragover="over(undefined, $event)"
    @drop="drop"
  >
    <template v-for="run in shown.runs" :key="run.id">
      <!-- A card let go on a section's heading lands at the head of that
           section, which is the one place a section holding none takes one. -->
      <div
        v-if="run.section"
        class="deck__section-head caret-below"
        :data-section-head="run.section.id"
        :data-before="run.section.id === at || undefined"
        @dragover.stop="over(run.section.id, $event)"
        @drop.stop="drop"
      >
        <SectionHeading
          :section="run.section"
          :words="words"
          @rename="(name: string) => emit('rename-section', run.section?.id ?? '', name)"
          @remove="emit('remove-section', run.section?.id ?? '')"
        />
      </div>

      <!-- The place before the first section, where a card is let go to stand
           under no section at all. -->
      <div
        v-else-if="run.landing"
        class="deck__head caret-below"
        data-head
        :data-before="run.id === at || undefined"
        @dragover.stop="over(run.id, $event)"
        @drop.stop="drop"
      ></div>

      <div class="deck__grid">
        <div
          v-for="tile in run.tiles"
          :key="tile.id"
          class="deck__tile caret-beside"
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
            @step="(direction, press) => step(tile.id, direction, press)"
            @write="(field, nth, text) => emit('write', tile.id, field, nth, text)"
          />
        </div>

        <!-- A card is made at the end of a section, so every section carries a
             plus of its own. What stands before the first section carries one
             wherever a card stands there, and a deck nobody has divided is one
             such run. -->
        <article
          v-if="run.plusAt !== null"
          class="deck__tile deck__plus caret-beside rounded-node"
          :aria-posinset="run.plusAt"
          :aria-setsize="shown.of"
          :aria-label="words.add"
          data-plus
          :data-plus-of="run.section?.id"
          :data-before="after(run) === at || undefined"
          @dragover.stop="over(after(run), $event)"
          @drop.stop="drop"
        >
          <!-- The plus says what it is for by standing alone in the middle. What
               it is called is read aloud and shown on hovering, and not beside it.
               What it opens takes its place, so it says nothing of being open. -->
          <Button
            v-if="asking !== run.id"
            variant="ghost"
            class="deck__ask"
            :aria-label="words.add"
            :title="words.add"
            @click="asking = run.id"
          >
            <Icon shows="plus" />
          </Button>

          <div v-else class="deck__asking flex flex-col items-center">
            <p class="deck__silence caps-numen text-small text-hushed">{{ words.cut }}</p>
            <div class="deck__cuts flex flex-wrap justify-center">
              <Button
                v-for="stencil in stencils"
                :key="stencil.name"
                variant="outline"
                size="small"
                :data-cut="stencil.name"
                @click="add(stencil, run)"
                >{{ stencil.name }}</Button
              >
            </div>
          </div>
        </article>
      </div>
    </template>

    <Divider>
      <Button variant="ghost" size="small" data-add-section @click="addSection">
        <Icon shows="plus" />
        {{ words.addSection }}
      </Button>
    </Divider>
  </div>
</template>

<style scoped>
@import '../caret.css';

/* The deck is what scrolls. The grid inside it is as tall as its rows, which is
   what lets every row take the height of the tallest tile of the whole deck.
   The room the bar takes is kept whether it is drawn or not: how many columns
   fit is read off this width. */
.deck {
  --tile: 20rem;
  --gap: 0.75rem;
  /* The caret stands in the middle of the room between two tiles. */
  --caret-out: calc(-1 * var(--gap) / 2);

  display: flex;
  flex-direction: column;
  gap: var(--gap);
  padding: var(--numen-gutter);
  overflow: auto;
  scrollbar-gutter: stable;
}

/* A section's heading runs the width of the grid under it, and takes the caret
   that says a card would land at its head. */
.deck__section-head {
  position: relative;
}

/* The place before the first section is a strip the width of the grid, standing
   where a card let go there would go and taking the caret a heading takes. */
.deck__head {
  position: relative;
  block-size: var(--gap);
}

/* A run drawing neither a card nor a plus takes no room of its own, which is
   the place before the first section of a deck holding nothing there. */
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

/* The plus is an outline until it is pressed: it holds nothing yet. What it
   holds stands in the middle, which is where it was pressed. */
.deck__plus {
  --plus: 2.5rem;

  place-items: center;
  min-block-size: 12rem;
  padding: var(--numen-box-air);
  border: var(--numen-stroke) dashed var(--numen-rule);
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
