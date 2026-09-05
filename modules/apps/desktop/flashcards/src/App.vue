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

import Vaults from './vaults/Vaults.vue'
import Decks from './decks/Decks.vue'
import Session from './session/Session.vue'
import SessionSummary from './session/SessionSummary.vue'
import AgentPanel from './session/AgentPanel.vue'
import NotesPanel from './session/NotesPanel.vue'
import { VERSION } from './version'
import { useWindow } from './window'

const { on, notices, putAway, vaults, decks, session } = useWindow()
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
      @back="decks.vaultsAgain"
    />

    <SessionSummary
      v-else-if="session.sat.over.value"
      :done="session.sat.done.value"
      @leave="session.leave"
    />

    <Session
      v-else-if="session.sat.card.value"
      :card="session.sat.card.value"
      :shown="session.sat.shown.value"
      :left="session.sat.left.value"
      :taken-back="session.sat.answers.value.length > 0"
      :at="session.at.value"
      @update:at="session.moved"
      @show="session.sat.show"
      @answer="session.answered"
      @take-back="session.sat.takeBack"
      @leave="session.leave"
      @ask="session.talks"
      @read="session.reads"
    >
      <template #reading>
        <NotesPanel :held="session.notesPanel" />
      </template>
      <template #panel>
        <AgentPanel :held="session.agentPanel" />
      </template>
    </Session>
  </main>

  <!-- What the window has to say, in the corner every window says it in. -->
  <Notices :notices="notices" @gone="putAway" />
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
