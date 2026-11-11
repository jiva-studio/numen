<script setup lang="ts">
import { SectionHeading } from './section-heading'
import { Card } from './card'
import { CutChoice } from './card-model-choice'
import { Icon } from '../icon'
import { Button } from '@numen/ui'
import { DECK_WORDS, getRunEndPoint } from '../../../../lib/cards'
import { getSealedMap, type InsertionPoint, type StepDirection } from '../../../../lib/cards'
import type { CardModel } from '../../../../lib/cards'
import type { DeckRunProps } from './DeckRun.types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<DeckRunProps>(), { words: () => DECK_WORDS })

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'drag-over', target: InsertionPoint | undefined, press: DragEvent): void
  (event: 'drop'): void
  (event: 'rename-section', id: string, name: string): void
  (event: 'remove-section', id: string): void
  (event: 'remove', card: string): void
  (event: 'drag-start', card: string, press: DragEvent): void
  (event: 'release'): void
  (event: 'step', card: string, direction: StepDirection, press: KeyboardEvent): void
  (event: 'ask'): void
  (event: 'add', cardModel: CardModel): void
}>()

/* ---------------------------------- State --------------------------------- */
const NO_FIELDS: ReadonlyMap<string, readonly string[]> = getSealedMap()

/* -------------------------------- Handlers -------------------------------- */
const onRenameSection = (name: string): void => {
  const section = props.run.section
  if (section) emit('rename-section', section.id, name)
}

const onRemoveSection = (): void => {
  const section = props.run.section
  if (section) emit('remove-section', section.id)
}
</script>

<template>
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
        @drag-start="emit('drag-start', tile.id, $event)"
        @release="emit('release')"
        @step="(direction, press) => emit('step', tile.id, direction, press)"
      />
    </div>

    <article
      v-if="run.plusAt !== null"
      class="deck__tile deck__plus caret-beside rounded-node"
      :aria-posinset="run.plusAt"
      :aria-setsize="of"
      :aria-label="words.add"
      data-plus
      :data-plus-of="run.section?.id"
      :data-before="getRunEndPoint(run.id) === at || undefined"
      @dragover.stop="emit('drag-over', getRunEndPoint(run.id), $event)"
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

.deck__section-head {
  position: relative;
}

.deck__head {
  position: relative;
  block-size: var(--gap);
}

.deck__grid:empty {
  display: none;
}

.deck__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(var(--tile), 100%), 1fr));
  grid-auto-rows: 1fr;
  gap: var(--gap);
}

.deck__tile {
  position: relative;
  display: grid;
  min-inline-size: 0;
}

.deck__plus {
  --plus-min-block-size: 12rem;
  --plus: 2.5rem;

  place-items: center;
  min-block-size: var(--plus-min-block-size);
  padding: var(--numen-box-air);
  border: var(--numen-stroke) dashed var(--numen-rule);
}

.deck__ask {
  block-size: auto;
  inline-size: auto;
  padding: var(--numen-inset);
  font-size: var(--plus);
}
</style>
