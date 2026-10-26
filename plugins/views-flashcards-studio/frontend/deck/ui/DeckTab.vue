<script setup lang="ts">
/**
 * Displays flashcard deck editor in grid view and handles deck management gestures.
 */
import { computed } from 'vue'
import {
  getBlanks as cardBlanks,
  getDeclaredFields as cardFields,
  DeckEditor,
  type InsertionPoint,
} from '../../ui/cards'
import { parseFileConflict } from '../../buffer'
import FileConflictPrompt from '../../ui/FileConflictPrompt.vue'
import type { DeckTabState } from '../types'
import DeckScheduleBar from './DeckScheduleBar.vue'
import { WORDS as words } from '../../cards'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ state: DeckTabState }>()

/* --------------------------------- State ---------------------------------- */
const { choices, deckCards, marks, errorMessage, note, scheduled, sections, stencils } = props.state

/** Validation errors against cards as reported by deck marks. */
const validationErrors = computed(() => ({ at: marks.value.at, ['under']: marks.value.bySection }))

/* -------------------------------- Handlers -------------------------------- */
function onChooseSchedule(path: string) {
  props.state.setSchedule(path)
}

function onAddCard(stencil: string, section: string | null) {
  props.state.addCard(stencil, empty(stencil), section)
}

function onRemoveCard(id: string) {
  props.state.removeCard(id)
}

function onMoveCard(id: string, target: InsertionPoint) {
  props.state.moveCard(id, target)
}

function onWriteCard(...[id, field, index, text]: [string, string, number, string]) {
  props.state.writeCardField(id, field, index, text)
}

function onAddSection(name: string) {
  props.state.addSection(name)
}

function onRenameSection(id: string, name: string) {
  props.state.renameSection(id, name)
}

function onRemoveSection(id: string) {
  props.state.removeSection(id)
}

function onKeep() {
  props.state.keepMine()
}

function onTake() {
  props.state.takeFile()
}

/* -------------------------------- Helpers --------------------------------- */
/** The untyped values a card cut by that stencil is made with. */
function empty(stencil: string) {
  return cardBlanks(cardFields(stencils.value.find((one) => one.name === stencil)?.fields ?? []))
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

    <DeckEditor
      class="deck-tab__grid"
      :cards="deckCards"
      :sections="sections"
      :stencils="stencils"
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
  </div>
</template>

<style scoped>
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

.wrong {
  margin: 0;
  padding-inline-start: 1.1rem;
  list-style: disc;
}
</style>
