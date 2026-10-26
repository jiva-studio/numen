<script setup lang="ts">
/**
 * The flashcards window: the vaults and what each owes, and the cards
 * themselves.
 *
 * What the window is made of is `window.ts`. What is here is what a person
 * sees of it, and the binding between the two.
 */
import type { NoticeHost } from '@numen/plugin-sdk'

import { Vaults } from '../pages/vaults'
import { Decks } from '../pages/decks'
import { AgentPanel, NotesPanel, Session, SessionSummary } from '../pages/session'
import { VERSION } from '../shared/version'
import { useWindow } from './window'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  /** The corner of the window the notices are drawn in. */
  notices: NoticeHost
}>()

/* --------------------------------- State ---------------------------------- */
const { on, vaults, decks, session } = useWindow(props.notices)
</script>

<template>
  <main class="flashcards">
    <Vaults
      v-if="on === 'vaults'"
      :vaults="vaults.list.value"
      :is-counting="vaults.isCounting.value"
      :version="VERSION"
      @choose="vaults.choose"
    />

    <Decks
      v-else-if="on === 'decks' && decks.chosen.value"
      :vault="decks.chosen.value"
      :days="decks.done.days.value"
      :due="decks.done.due.value"
      :presets="decks.schedules.presets.value"
      :by-deck="decks.schedules.byDeck.value"
      :has-presets="decks.schedules.known.value"
      :today="decks.today.value"
      @start="decks.start"
      @start-preset="decks.startPreset"
      @back="decks.goToVaults"
    />

    <SessionSummary
      v-else-if="session.state.over.value"
      :done="session.state.done.value"
      @leave="session.leave"
    />

    <Session
      v-else-if="session.state.card.value"
      :card="session.state.card.value"
      :is-shown="session.state.shown.value"
      :left="session.state.left.value"
      :can-take-back="session.state.answers.value.length > 0"
      :at="session.at.value"
      @update:at="session.moveTo"
      @show="session.state.show"
      @answer="session.answerCard"
      @take-back="session.state.takeBack"
      @leave="session.leave"
      @ask="session.toggleAgent"
      @read="session.toggleNotes"
    >
      <template #reading>
        <NotesPanel :panel="session.notesPanel" />
      </template>
      <template #panel>
        <AgentPanel :panel="session.agentPanel" />
      </template>
    </Session>
  </main>
</template>

<style scoped>
.flashcards {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  padding: var(--numen-inset-wide);
  gap: var(--numen-inset);
}
</style>
