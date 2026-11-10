<script setup lang="ts">
/**
 * The review of one vault, as the window shows it: the decks the vault has due
 * today, and the cards themselves.
 *
 * What the review is made of is `review.ts`, created when the window opens it.
 * This component renders it and binds the two.
 */
import { computed, provide, type ComponentPublicInstance } from 'vue'
import { Spinner } from '@numen/ui'

import Decks from './decks/Decks.vue'
import AgentPanel from './session/AgentPanel.vue'
import NotesPanel from './session/NotesPanel.vue'
import Session from './session/Session.vue'
import SessionSummary from './session/SessionSummary.vue'
import { MOUNT_VIEW } from '../model/views'
import { isNotesPage, type Review } from '../model/review'
import { REVIEW_WORDS } from '../lib/words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  state: Review
}>()

/* --------------------------------- State ---------------------------------- */
const { on, vaultCount, decks, session, actions, notesPage } = props.state
provide(MOUNT_VIEW, props.state.mountView)

/** The status shown while the decks wait on the vault's count. */
const statusText = computed(() => {
  if (vaultCount.value?.unread) return vaultCount.value.unread
  return vaultCount.value?.isIndexing ? REVIEW_WORDS.indexing : REVIEW_WORDS.counting
})

/* -------------------------------- Handlers -------------------------------- */
function onNotesPage(page: Element | ComponentPublicInstance | null) {
  notesPage.value = isNotesPage(page) ? page : null
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
        @start="actions.start"
        @start-preset="actions.startPreset"
        @back="actions.goToVaults"
      />

      <!-- The decks are drawn from the vault's count, which may still be on its way. -->
      <p v-else class="review__waiting" role="status">
        <Spinner v-if="!vaultCount?.unread" />
        {{ statusText }}
      </p>
    </template>

    <SessionSummary
      v-else-if="session.state.cards.progress.isOver.value"
      :done="session.state.cards.progress.completedCount.value"
      @leave="actions.leave"
    />

    <Session
      v-else-if="session.state.cards.queue.card.value"
      :card="session.state.cards.queue.card.value"
      :is-shown="session.state.cards.queue.isAnswerShown.value"
      :left="session.state.cards.progress.left.value"
      :can-take-back="session.state.cards.queue.answers.value.length > 0"
      :at="session.panels.at.value"
      @update:at="session.panels.moveTo"
      @show="session.state.show"
      @answer="actions.recordReview"
      @take-back="session.state.answering.takeBack"
      @leave="actions.leave"
      @ask="session.panels.toggleAgent"
      @read="session.panels.toggleNotes"
    >
      <template #reading>
        <NotesPanel :ref="onNotesPage" :panel="session.panels.notesPanel" />
      </template>
      <template #panel>
        <AgentPanel :panel="session.panels.agentPanel" />
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

/* The review's muted waiting text, shown while it waits. */
.review__waiting {
  display: flex;
  margin: 0;
  align-items: center;
  gap: var(--numen-inset);
  color: var(--numen-edge-label);
  font-size: var(--numen-text-1);
}
</style>
