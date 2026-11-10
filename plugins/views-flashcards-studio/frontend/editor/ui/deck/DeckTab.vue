<script setup lang="ts">
/**
 * Displays flashcard deck editor in grid view and handles deck management gestures.
 */
import { computed } from 'vue'
import {
  getBlanks as cardBlanks,
  getDeclaredFields as cardFields,
  type InsertionPoint,
} from '../../../lib/cards'
import { DeckEditor } from '../cards/deck-editor'
import { parseFileConflict } from '../../lib/buffer/words'
import FileConflictPrompt from '../FileConflictPrompt.vue'
import DeckScheduleBar from './DeckScheduleBar.vue'
import { WORDS as words } from '../../../words'
import type { DeckTabState } from '../../model/deck/tabState'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ state: DeckTabState }>()

/* --------------------------------- State ---------------------------------- */
const { deckCards, marks, errorMessage, note, sections, cardModels } = props.state.data
const { choices, scheduled } = props.state.schedule

/** Validation errors against cards as reported by deck marks. */
const validationErrors = computed(() => ({ at: marks.value.at, ['under']: marks.value.bySection }))

/* -------------------------------- Handlers -------------------------------- */
function onChooseSchedule(path: string) {
  props.state.schedule.setSchedule(path)
}

function onAddCard(cardModel: string, section: string | null) {
  props.state.actions.addCard(cardModel, empty(cardModel), section)
}

function onRemoveCard(id: string) {
  props.state.actions.removeCard(id)
}

function onMoveCard(id: string, target: InsertionPoint) {
  props.state.actions.moveCard(id, target)
}

function onWriteCard(...[id, field, index, text]: [string, string, number, string]) {
  props.state.actions.writeCardField(id, field, index, text)
}

function onAddSection(name: string) {
  props.state.actions.addSection(name)
}

function onRenameSection(id: string, name: string) {
  props.state.actions.renameSection(id, name)
}

function onRemoveSection(id: string) {
  props.state.actions.removeSection(id)
}

function onKeep() {
  props.state.file.keepMine()
}

function onTake() {
  props.state.file.takeFile()
}

/* -------------------------------- Helpers --------------------------------- */
function empty(cardModel: string) {
  return cardBlanks(
    cardFields(cardModels.value.find((one) => one.name === cardModel)?.fields ?? []),
  )
}
</script>

<template>
  <div class="deck-tab">
    <FileConflictPrompt
      :error-message="errorMessage"
      :conflict="parseFileConflict(note.state)"
      :words="words"
      @keep="onKeep"
      @take="onTake"
    />

    <ul v-if="marks.whole.length" class="caution wrong" :aria-label="words.problems">
      <li v-for="(text, at) in marks.whole" :key="at">{{ text }}</li>
    </ul>

    <DeckScheduleBar :scheduled="scheduled" :choices="choices" @choose="onChooseSchedule" />

    <!-- The editor takes no input until the file is read. -->
    <fieldset class="deck-tab__body" :disabled="!note.isRead">
      <DeckEditor
        class="deck-tab__grid"
        :cards="deckCards"
        :sections="sections"
        :card-models="cardModels"
        :name="words.deck"
        :wrong="validationErrors"
        @add="onAddCard"
        @remove="onRemoveCard"
        @move="onMoveCard"
        @write="onWriteCard"
        @add-section="onAddSection"
        @rename-section="onRenameSection"
        @remove-section="onRemoveSection"
      />
    </fieldset>
  </div>
</template>

<style scoped>
.deck-tab {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
}

.deck-tab__body {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-block-size: 0;
  min-inline-size: 0;
  margin: 0;
  padding: 0;
  border: 0;
}

.deck-tab__grid {
  flex: 1;
  min-block-size: 0;
}

.wrong {
  --wrong-padding-inline-start: 1.1rem;
  margin: 0;
  padding-inline-start: var(--wrong-padding-inline-start);
  list-style: disc;
}
</style>
