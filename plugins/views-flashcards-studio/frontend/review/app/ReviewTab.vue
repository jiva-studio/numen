<script setup lang="ts">
/**
 * The review of one vault, as the window shows it: the decks the vault owes
 * today, and the cards themselves.
 *
 * What the review is made of is `review.ts`, made as the window opens it. What
 * is here is what a person sees of it, and the binding between the two.
 */
import { computed, provide } from 'vue'
import { Spinner } from '@numen/ui'

import { Decks } from '../pages/decks'
import { AgentPanel, NotesPanel, Session, SessionSummary } from '../pages/session'
import { MOUNT_VIEW } from '../shared/views'
import type { NotesPage, Review } from './review'
import { REVIEW_WORDS } from './words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  /** The review the window opened on a vault. */
  state: Review
}>()

/* --------------------------------- State ---------------------------------- */
const { on, vaultCount, decks, session, notesPage } = props.state
provide(MOUNT_VIEW, props.state.mountView)

/** What the review says while the decks wait on the vault's count. */
const statusText = computed(() => {
  if (vaultCount.value?.unread) return vaultCount.value.unread
  return vaultCount.value?.isIndexing ? REVIEW_WORDS.indexing : REVIEW_WORDS.counting
})

/* -------------------------------- Handlers -------------------------------- */
function onNotesPage(page: unknown) {
  notesPage.value = (page as NotesPage | null) ?? null
}
</script>

<template>
  <div class="review">
    <template v-if="on === 'decks'">
      <Decks
        v-if="vaultCount?.isCounted"
        :vault="vaultCount"
        :days="decks.reviewDays.days.value"
        :due="decks.reviewDays.due.value"
        :presets="decks.schedules.presets.value"
        :by-deck="decks.schedules.byDeck.value"
        :has-presets="decks.schedules.known.value"
        :today="decks.today.value"
        @start="decks.start"
        @start-preset="decks.startPreset"
        @back="decks.goToVaults"
      />

      <!-- The decks are drawn from the vault's count, which may still be on its way. -->
      <p v-else class="review__waiting" role="status">
        <Spinner v-if="!vaultCount?.unread" />
        {{ statusText }}
      </p>
    </template>

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
        <NotesPanel :ref="onNotesPage" :panel="session.notesPanel" />
      </template>
      <template #panel>
        <AgentPanel :panel="session.agentPanel" />
      </template>
    </Session>
  </div>
</template>

<style scoped>
.review {
  display: flex;
  flex: 1 1 0%;
  flex-direction: column;
  min-block-size: 0;
  gap: var(--numen-inset);
}

/* The review's own quiet voice, which is what it says while it waits. */
.review__waiting {
  display: flex;
  margin: 0;
  align-items: center;
  gap: var(--numen-inset);
  color: var(--numen-edge-label);
  font-size: var(--numen-text-1);
}
</style>
