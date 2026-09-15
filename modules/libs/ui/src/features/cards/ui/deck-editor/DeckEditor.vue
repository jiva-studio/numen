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
import DeckRun from './DeckRun.vue'
import { Icon } from '../icon'
import { Divider } from '../divider'
import { useDrag } from '../../model/drag'
import { Button } from '@/shared/ui/button'
import {
  DECK_WORDS,
  HEAD,
  NOTHING_WRONG,
  type DeckSection,
  type DeckWords,
  type DeckCard,
  type Wrong,
} from '../../lib/deck'
import { getGrid, isMoved, type Run } from '../../lib/grid'
import { getFreeName, type InsertionPoint, type StepDirection } from '../../lib/order'
import type { Stencil } from '../../lib/card'

// --- Props & Emits ---
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
  (event: 'add', stencil: string, section: string | null): void
  (event: 'remove', card: string): void
  (event: 'move', card: string, at: InsertionPoint): void
  (event: 'write', card: string, field: string, nth: number, text: string): void
  (event: 'add-section', name: string): void
  (event: 'rename-section', id: string, name: string): void
  (event: 'remove-section', id: string): void
}>()

// --- State ---
/**
 * The run whose plus is showing which stencils a new card may be cut by.
 */
const asking = shallowRef<string | null>(null)

/**
 * The card under the pointer's hand, and where letting go would put it.
 */
const { dragged, at, lift, hover, release, drop, step } = useDrag<InsertionPoint | undefined>({
  order: () => [HEAD, ...props.cards.map((card) => card.id)],
  nowhere: undefined,
  isMoved: (held: string, at: InsertionPoint): boolean => isMoved(shown.value.runs, held, at),
  move: (held, at) => emit('move', held, at),
})

const shown = computed(() => getGrid(props.cards, props.sections, props.stencils, dragged.value))

// --- Handlers ---
function onDragOver(targetAt: InsertionPoint | undefined, event: DragEvent): void {
  hover(targetAt, event)
}

function onDrop(): void {
  drop()
}

function onRenameSection(id: string, name: string): void {
  emit('rename-section', id, name)
}

function onRemoveSection(id: string): void {
  emit('remove-section', id)
}

function onRemoveCard(id: string): void {
  emit('remove', id)
}

function onLift(id: string, event: DragEvent): void {
  lift(id, event)
}

function onRelease(): void {
  release()
}

function onStep(id: string, direction: StepDirection, press: KeyboardEvent): void {
  step(id, direction, press)
}

function onWriteCard(id: string, field: string, nth: number, text: string): void {
  emit('write', id, field, nth, text)
}

function onAsk(runId: string): void {
  asking.value = runId
}

function onAddCard(stencil: Stencil, run: Run): void {
  asking.value = null
  emit('add', stencil.name, run.section?.id ?? null)
}

function onAddSection(): void {
  emit(
    'add-section',
    getFreeName(
      props.sections.map((each) => each.name),
      props.words.sectionStem,
    ),
  )
}
</script>

<template>
  <div
    class="deck numen bg-surface text-ink font-sans text-base"
    role="group"
    :aria-label="name"
    @dragover="onDragOver(undefined, $event)"
    @drop="onDrop"
  >
    <DeckRun
      v-for="run in shown.runs"
      :key="run.id"
      :run="run"
      :stencils="stencils"
      :of="shown.of"
      :wrong="wrong"
      :at="at"
      :asking="asking === run.id"
      :words="words"
      @drag-over="onDragOver"
      @drop="onDrop"
      @rename-section="onRenameSection"
      @remove-section="onRemoveSection"
      @remove="onRemoveCard"
      @lift="onLift"
      @release="onRelease"
      @step="onStep"
      @write="onWriteCard"
      @ask="onAsk(run.id)"
      @add="(stencil) => onAddCard(stencil, run)"
    />

    <Divider>
      <Button variant="ghost" size="small" data-add-section @click="onAddSection">
        <Icon name="plus" />
        {{ words.addSection }}
      </Button>
    </Divider>
  </div>
</template>

<style scoped>
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
</style>
