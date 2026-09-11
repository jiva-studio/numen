<script setup lang="ts">
/**
 * Displays flashcard deck editor in grid view and handles deck management gestures.
 */
import { computed, ref } from 'vue'
import { cardBlanks, cardFields, DeckEditor, Menu } from '@numen/ui'
import type { InsertionPoint, Position } from '@numen/ui'
import { ChevronDown } from '@lucide/vue'
import FileConflictPrompt from '../shared/saving/FileConflictPrompt.vue'
import { conflictIn } from '../shared/saving/flushing'
import type { DeckTabState } from './types'
import { WORDS as words } from '../shared/flashcards/words'

// --- Props & Emits ---
const props = defineProps<{ state: DeckTabState }>()

// --- State ---
// The tab's state outlives this component, so what it holds is bound once here
// and the template unwraps it.
const { choices, drawn, marks, errorMessage, scheduled, sections, shown, stencils } = props.state

/** What the grid draws against the cards it was handed. */
const wrong = computed(() => ({ at: marks.value.at, under: marks.value.under }))

/**
 * The presets on offer. The defaults stand in a group of their own, so the line
 * between them and the notes says which is which.
 */
const offered = computed(() =>
  choices.value.map((one) => ({
    id: one.path,
    text: one.name,
    group: one.path === '' ? 'defaults' : 'presets',
  })),
)

/** Where the presets were asked for, and nothing while they are not. */
const asking = ref<{ at: Position; from: HTMLElement } | null>(null)

// --- Handlers ---
/** The line saying which preset schedules this deck opens the presets under it. */
function onOpenScheduleMenu(event: Event) {
  const line = event.currentTarget
  if (!(line instanceof HTMLElement)) return
  const box = line.getBoundingClientRect()
  asking.value = { at: { x: box.left, y: box.bottom }, from: line }
}

function onChooseSchedule(path: string) {
  asking.value = null
  props.state.setSchedule(path)
}

function onDismissScheduleMenu() {
  asking.value = null
}

function onAddCard(stencil: string, section: string | null) {
  props.state.addCard(stencil, empty(stencil), section)
}

function onRemoveCard(id: string) {
  props.state.removeCard(id)
}

function onMoveCard(id: string, at: InsertionPoint) {
  props.state.moveCard(id, at)
}

function onWriteCard(id: string, field: string, nth: number, text: string) {
  props.state.writeCardField(id, field, nth, text)
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

// --- Helpers ---
/** The untyped values a card cut by that stencil is made with. */
function empty(stencil: string) {
  return cardBlanks(
    cardFields(stencils.value.find((one) => one.name === stencil)?.fields ?? []),
  )
}
</script>

<template>
  <div class="deck-tab">
    <FileConflictPrompt
      :errorMessage="errorMessage"
      :conflict="conflictIn(shown.state)"
      :words="words"
      @keep="onKeep"
      @take="onTake"
    />

    <ul v-if="marks.whole.length" class="caution wrong" :aria-label="words.problems">
      <li v-for="(text, at) in marks.whole" :key="at">{{ text }}</li>
    </ul>

    <!-- Which preset schedules this deck, and the presets under it when it is
         asked. What the deck names and the vault does not hold is said here. -->
    <p class="deck-tab__scheduled">
      <span id="deck-scheduled" class="deck-tab__by">{{ words.scheduledBy }}</span>
      <button
        type="button"
        class="deck-tab__choice"
        aria-haspopup="menu"
        aria-labelledby="deck-scheduled"
        @click="onOpenScheduleMenu"
      >
        {{ scheduled.name }}
        <ChevronDown class="deck-tab__icon" aria-hidden="true" />
      </button>
      <span v-if="scheduled.errorMessage" role="status" class="deck-tab__wrong">
        {{ scheduled.errorMessage }}
      </span>
    </p>

    <DeckEditor
      class="deck-tab__grid"
      :cards="drawn"
      :sections="sections"
      :stencils="stencils"
      :name="words.deck"
      :wrong="wrong"
      @add="onAddCard"
      @remove="onRemoveCard"
      @move="onMoveCard"
      @write="onWriteCard"
      @add-section="onAddSection"
      @rename-section="onRenameSection"
      @remove-section="onRemoveSection"
    />

    <Menu
      v-if="asking"
      :items="offered"
      :at="asking.at"
      :from="asking.from"
      :current="scheduled.path"
      open
      opening="keyboard"
      :name="words.scheduledBy"
      @choose="onChooseSchedule"
      @dismiss="onDismissScheduleMenu"
    />
  </div>
</template>

<style scoped>
/* The grid takes what the rows above it leave, and scrolls inside itself. */
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

/* The line above the grid, which takes the height it needs and no more. */
.deck-tab__scheduled {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
  margin: 0;
  padding: 0.75rem 1rem 0.6rem;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
}

.deck-tab__by {
  color: var(--numen-hushed);
}

/* A line of text carrying the mark that says it opens. */
.deck-tab__choice {
  display: inline-flex;
  align-items: center;
  gap: var(--numen-node-gap);
  padding: 0.15rem 0.4rem;
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius-tight);
  background: var(--numen-field-bg);
  color: inherit;
  font: inherit;
  line-height: 1.2;
  cursor: pointer;
}

.deck-tab__choice:focus-visible {
  outline: none;
  outline-offset: var(--numen-stroke);
}

.deck-tab__icon {
  inline-size: 1em;
  block-size: 1em;
}

.deck-tab__wrong {
  color: var(--numen-alarm);
  overflow-wrap: anywhere;
}

.wrong {
  margin: 0;
  padding-inline-start: 1.1rem;
  list-style: disc;
}
</style>
