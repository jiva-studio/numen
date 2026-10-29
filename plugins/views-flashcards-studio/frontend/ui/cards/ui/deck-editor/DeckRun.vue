<script setup lang="ts">
/**
 * One run of a deck: the heading of the section it is placed under, the cards
 * under it as tiles, and the plus that ends it.
 */
import { SectionHeading } from './section-heading'
import { Card } from './card'
import { CutChoice } from './card-model-choice'
import { Icon } from '../icon'
import { Button } from '@numen/ui'
import { DECK_WORDS, createRunEnd } from '../../lib/deck'
import type { Run } from '../../lib/grid'
import { createSealedMap, type InsertionPoint, type StepDirection } from '../../lib/order'
import type { CardModel } from '../../lib/card'
import type { DeckRunProps } from './DeckRun.types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<DeckRunProps>(), { words: () => DECK_WORDS })

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** A card is dragged over a place that would take it. */
  (event: 'drag-over', target: InsertionPoint | undefined, press: DragEvent): void
  (event: 'drop'): void
  (event: 'rename-section', id: string, name: string): void
  (event: 'remove-section', id: string): void
  (event: 'remove', card: string): void
  (event: 'lift', card: string, press: DragEvent): void
  (event: 'release'): void
  (event: 'step', card: string, direction: StepDirection, press: KeyboardEvent): void
  /** The plus was pressed. */
  (event: 'ask'): void
  (event: 'add', cardModel: CardModel): void
}>()

/* ---------------------------------- State --------------------------------- */
/** A card nothing is wrong with any value of. */
const NO_FIELDS: ReadonlyMap<string, readonly string[]> = createSealedMap()

/* -------------------------------- Handlers -------------------------------- */
/** The section this run is placed under, under a new name. */
const onRenameSection = (name: string): void => {
  const section = props.run.section
  if (section) emit('rename-section', section.id, name)
}

/** The section this run is placed under, gone. */
const onRemoveSection = (): void => {
  const section = props.run.section
  if (section) emit('remove-section', section.id)
}

/* --------------------------------- Helpers -------------------------------- */
/** Where a card let go past the last of this run lands. */
const getRunEnd = (run: Run): InsertionPoint => createRunEnd(run.id)
</script>

<template>
  <!-- A card let go on a section's heading lands at the head of that section. -->
  <div
    v-if="run.section"
    class="deck__section-head caret-below"
    :data-section-head="run.section.id"
    :data-before="run.section.id === at || undefined"
    @dragover.stop="emit('drag-over', run.section.id, $event)"
    @drop.stop="emit('drop')"
  >
    <SectionHeading
      :section="run.section"
      :words="words"
      @rename="onRenameSection"
      @remove="onRemoveSection"
    />
  </div>

  <!-- The place before the first section, where a card is let go to stand
       under no section at all. -->
  <div
    v-else-if="run.isDropTarget"
    class="deck__head caret-below"
    data-head
    :data-before="run.id === at || undefined"
    @dragover.stop="emit('drag-over', run.id, $event)"
    @drop.stop="emit('drop')"
  ></div>

  <div class="deck__grid">
    <div
      v-for="tile in run.tiles"
      :key="tile.id"
      class="deck__tile caret-beside"
      :data-before="tile.id === at || undefined"
      @dragover.stop="emit('drag-over', tile.id, $event)"
      @drop.stop="emit('drop')"
    >
      <Card
        :tile="tile"
        :wrong="wrong.at.get(tile.id) ?? []"
        :wrong-under="wrong.under.get(tile.id) ?? NO_FIELDS"
        :words="words"
        @remove="emit('remove', tile.id)"
        @lift="emit('lift', tile.id, $event)"
        @release="emit('release')"
        @step="(direction, press) => emit('step', tile.id, direction, press)"
      />
    </div>

    <!-- Every section carries a plus of its own. -->
    <article
      v-if="run.plusAt !== null"
      class="deck__tile deck__plus caret-beside rounded-node"
      :aria-posinset="run.plusAt"
      :aria-setsize="of"
      :aria-label="words.add"
      data-plus
      :data-plus-of="run.section?.id"
      :data-before="getRunEnd(run) === at || undefined"
      @dragover.stop="emit('drag-over', getRunEnd(run), $event)"
      @drop.stop="emit('drop')"
    >
      <Button
        v-if="!isAsking"
        variant="ghost"
        class="deck__ask"
        :aria-label="words.add"
        :title="words.add"
        @click="emit('ask')"
      >
        <Icon name="plus" />
      </Button>

      <CutChoice
        v-else
        :card-models="cardModels"
        :words="words"
        @choose="(cardModel) => emit('add', cardModel)"
      />
    </article>
  </div>
</template>

<style scoped>
@import '../caret.css';

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

/* A run that renders neither a card nor a plus takes no space of its own, which is
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

/* A tile is the space one card is rendered in, and the card fills it. */
.deck__tile {
  position: relative;
  display: grid;
  min-inline-size: 0;
}

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
</style>
