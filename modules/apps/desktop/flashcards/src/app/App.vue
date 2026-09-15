<script setup lang="ts">
/**
 * The flashcards window: the vaults and what each owes, and the cards
 * themselves.
 *
 * It opens on what a person owes today and nothing else. They need not know an
 * editor exists.
 *
 * What the window is made of is `window.ts`. What is here is what a person
 * sees of it, and the binding between the two.
 */
import { Notices } from '@numen/ui'
import '@numen/ui/styles.css'

import { Vaults } from '@/pages/vaults'
import { Decks } from '@/pages/decks'
import { AgentPanel, NotesPanel, Session, SessionSummary } from '@/pages/session'
import { VERSION } from '@/shared/version'
import { useWindow } from './window'

const { on, notices, dismissNotice, vaults, decks, session } = useWindow()
</script>

<template>
  <main class="flashcards">
    <Vaults
      v-if="on === 'vaults'"
      :vaults="vaults.list.value"
      :counting="vaults.counting.value"
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
      :scheduled="decks.schedules.known.value"
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
      :shown="session.state.shown.value"
      :left="session.state.left.value"
      :taken-back="session.state.answers.value.length > 0"
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
        <NotesPanel ref="page" :held="session.notesPanel" />
      </template>
      <template #panel>
        <AgentPanel :held="session.agentPanel" />
      </template>
    </Session>
  </main>

  <!-- What the window has to say, in the corner every window says it in. -->
  <Notices :notices="notices" @dismiss="dismissNotice" />
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
