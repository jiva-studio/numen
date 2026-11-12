<script setup lang="ts">
import { computed, provide, shallowRef } from 'vue'
import DeckRun from './DeckRun.vue'
import { Icon } from '../icon'
import { Divider } from '../divider'
import { DECK_CONTEXT } from '../../../model/cards/context'
import { useDrag } from '../../../model/cards/drag'
import { Button } from '@numen/ui'
import {
  DECK_WORDS,
  HEAD,
  NO_DECK_PROBLEMS,
  type DeckSection,
  type DeckWords,
  type DeckCard,
  type DeckProblems,
} from '../../../../lib/cards'
import { getGrid, isMoved, type Run } from '../../../../lib/cards'
import { getFreeName, type InsertionPoint, type StepDirection } from '../../../../lib/cards'
import type { CardModel } from '../../../../lib/cards'

/* ----------------------------- Props & Emits ------------------------------ */
const props = withDefaults(
  defineProps<{
    cards: readonly DeckCard[]
    cardModels: readonly CardModel[]
    sections?: readonly DeckSection[]
    name?: string
    wrong?: DeckProblems
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

/* --------------------------------- State ---------------------------------- */
const activePickerRunId = shallowRef<string | null>(null)

const {
  dragged,
  at: dropTarget,
  startDrag,
  hover,
  release,
  drop,
  step,
} = useDrag<InsertionPoint | undefined>({
  order: () => [HEAD, ...props.cards.map((card) => card.id)],
  nowhere: undefined,
  isMoved: (dragged: string, target: InsertionPoint): boolean =>
    isMoved(grid.value.runs, dragged, target),
  move: (dragged, target) => emit('move', dragged, target),
})

const grid = computed(() => getGrid(props, dragged.value))

provide(DECK_CONTEXT, {
  write: (card, target, text) => emit('write', card, target.field, target.index, text),
})

/* -------------------------------- Handlers -------------------------------- */
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

function onDragStart(id: string, event: DragEvent): void {
  startDrag(id, event)
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
      @drag-start="onDragStart"
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
