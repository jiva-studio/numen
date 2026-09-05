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

import Vaults from './Vaults.vue'
import Decks from './Decks.vue'
import Session from './Session.vue'
import SessionSummary from './SessionSummary.vue'
import AgentPanel from './AgentPanel.vue'
import NotesPanel from './NotesPanel.vue'
import { VERSION } from './version'
import { useWindow } from './window'

const {
  answered,
  at,
  busy,
  choose,
  chosen,
  done,
  leave,
  moved,
  notices,
  on,
  panel,
  putAway,
  read,
  reads,
  sat,
  schedules,
  start,
  startPreset,
  talks,
  today,
  vaults,
  vaultsAgain,
} = useWindow()
</script>

<template>
  <main class="flashcards">
    <Vaults
      v-if="on === 'vaults'"
      :vaults="vaults"
      :counting="busy"
      :version="VERSION"
      @choose="choose"
    />

    <Decks
      v-else-if="on === 'decks' && chosen"
      :vault="chosen"
      :days="done.days.value"
      :due="done.due.value"
      :presets="schedules.presets.value"
      :by-deck="schedules.byDeck.value"
      :scheduled="schedules.known.value"
      :today="today"
      @start="start"
      @start-preset="startPreset"
      @back="vaultsAgain"
    />

    <SessionSummary v-else-if="sat.over.value" :done="sat.done.value" @leave="leave" />

    <Session
      v-else-if="sat.card.value"
      :card="sat.card.value"
      :shown="sat.shown.value"
      :left="sat.left.value"
      :taken-back="sat.answers.value.length > 0"
      :at="at"
      @update:at="moved"
      @show="sat.show"
      @answer="answered"
      @take-back="sat.takeBack"
      @leave="leave"
      @ask="talks"
      @read="reads"
    >
      <template #reading>
        <NotesPanel ref="page" :held="read" />
      </template>
      <template #panel>
        <AgentPanel :held="panel" />
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
