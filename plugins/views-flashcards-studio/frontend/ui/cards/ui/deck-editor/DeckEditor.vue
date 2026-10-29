<script setup lang="ts">
/**
 * A deck editor: its cards as tiles in a grid, grouped under their sections,
 * with a plus button last.
 *
 * The grid lays out the cards, drags a card between positions, and requests
 * which card model a new card is created from. A card's content belongs to the
 * card. What a card means is the caller's.
 */
import { computed, provide, shallowRef } from 'vue'
import DeckRun from './DeckRun.vue'
import { Icon } from '../icon'
import { Divider } from '../divider'
import { DECK_CONTEXT } from '../../model/context'
import { useDrag } from '../../model/drag'
import { Button } from '@numen/ui'
import {
  DECK_WORDS,
  HEAD,
  NO_DECK_PROBLEMS,
  type DeckSection,
  type DeckWords,
  type DeckCard,
  type DeckProblems,
} from '../../lib/deck'
import { getGrid, isMoved, type Run } from '../../lib/grid'
import { getFreeName, type InsertionPoint, type StepDirection } from '../../lib/order'
import type { CardModel } from '../../lib/card'

// --- Props & Emits ---
const props = withDefaults(
  defineProps<{
    /** The cards, in the order they are drawn. */
    cards: readonly DeckCard[]
    /** The card models a card may be cut by. */
    cardModels: readonly CardModel[]
    /** The sections, in the order they stand in the deck. */
    sections?: readonly DeckSection[]
    /** What the grid is announced as. */
    name?: string
    /** What the caller found wrong with the cards it handed in. */
    wrong?: DeckProblems
    /** The words it is drawn with. */
    words?: DeckWords
  }>(),
  { sections: () => [], name: 'Deck', wrong: () => NO_DECK_PROBLEMS, words: () => DECK_WORDS },
)

const emit = defineEmits<{
  (event: 'add', cardModel: string, section: string | null): void
  (event: 'remove', card: string): void
  (event: 'move', card: string, target: InsertionPoint): void
  (event: 'write', card: string, field: string, index: number, text: string): void
  (event: 'add-section', name: string): void
  (event: 'rename-section', id: string, name: string): void
  (event: 'remove-section', id: string): void
}>()

// --- State ---
/**
 * The run whose plus is showing which card models a new card may be cut by.
 */
const activePickerRunId = shallowRef<string | null>(null)

/**
 * The card under the pointer, and where dropping it would place it.
 */
const {
  dragged,
  at: dropTarget,
  lift,
  hover,
  release,
  drop,
  step,
} = useDrag<InsertionPoint | undefined>({
  order: () => [HEAD, ...props.cards.map((card) => card.id)],
  nowhere: undefined,
  isMoved: (held: string, target: InsertionPoint): boolean =>
    isMoved(grid.value.runs, held, target),
  move: (held, target) => emit('move', held, target),
})

const grid = computed(() => getGrid(props, dragged.value))

provide(DECK_CONTEXT, {
  write: (card, target, text) => emit('write', card, target.field, target.index, text),
})

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

function onAsk(runId: string): void {
  activePickerRunId.value = runId
}

function onAddCard(cardModel: CardModel, run: Run): void {
  activePickerRunId.value = null
  emit('add', cardModel.name, run.section?.id ?? null)
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
      v-for="run in grid.runs"
      :key="run.id"
      :run="run"
      :card-models="cardModels"
      :of="grid.of"
      :wrong="wrong"
      :at="dropTarget"
      :is-asking="activePickerRunId === run.id"
      :words="words"
      @drag-over="onDragOver"
      @drop="onDrop"
      @rename-section="onRenameSection"
      @remove-section="onRemoveSection"
      @remove="onRemoveCard"
      @lift="onLift"
      @release="onRelease"
      @step="onStep"
      @ask="onAsk(run.id)"
      @add="(cardModel) => onAddCard(cardModel, run)"
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
  --caret-out: calc(-1 * var(--gap) / 2);

  display: flex;
  flex-direction: column;
  gap: var(--gap);
  padding: var(--numen-gutter);
  overflow: auto;
  scrollbar-gutter: stable;
}
</style>
