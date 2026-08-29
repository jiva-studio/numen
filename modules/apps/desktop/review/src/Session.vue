<script setup lang="ts">
/**
 * One sitting: where the card is from, the card itself, and how it came back.
 *
 * The card and the boxes it is put right in stand in one place and change over,
 * so a person putting a card right does not lose sight of where they were.
 */
import { Button } from '@numen/ui'
import { SquarePen, Undo2, X } from '@lucide/vue'
import Answers from './Answers.vue'
import Card from './Card.vue'
import Editing from './Editing.vue'
import { deckName } from './core'
import type { Asked, Held, Said } from './core'

defineProps<{
  card: Asked
  /** How many are left to ask, this one among them. */
  left: number
  /** Whether the answer is being shown. */
  shown: boolean
  /** Whether an answer stands that can be taken back. */
  answered: boolean
  /** The card is open to be put right, and what it holds while it is. */
  editing: boolean
  values: readonly Held[]
  writing: boolean
}>()

defineEmits<{
  (event: 'show'): void
  (event: 'answer', how: Said): void
  (event: 'edit'): void
  (event: 'take-back'): void
  (event: 'leave'): void
  (event: 'write', field: string, text: string): void
  (event: 'save'): void
  (event: 'close'): void
}>()
</script>

<template>
  <section class="session">
    <header class="session__where">
      <span class="session__deck">{{ deckName(card.deck) }}</span>
      <span v-if="card.section">{{ card.section }}</span>
      <span>{{ card.face }}</span>
      <span class="session__left">{{ left }} left</span>

      <!-- What a person does beside answering, each one mark. They stand at the
           end of the line that says where the card is from. -->
      <Button variant="ghost" size="icon-small" title="Edit this card" @click="$emit('edit')">
        <SquarePen />
      </Button>
      <Button
        variant="ghost"
        size="icon-small"
        title="Take the last answer back"
        :disabled="!answered"
        @click="$emit('take-back')"
      >
        <Undo2 />
      </Button>
      <Button variant="ghost" size="icon-small" title="Leave" @click="$emit('leave')">
        <X />
      </Button>
    </header>

    <!-- The boxes a card is put right in stand above it, out of the frame.
         Opening them lets the column down by their own height, so they come
         into view and the card is pushed under them. -->
    <div class="session__stage">
      <div class="session__column" :class="{ 'session__column--down': editing }">
        <Editing
          class="session__slot"
          :inert="!editing"
          :card="card.card"
          :section="card.section"
          :values="values"
          :writing="writing"
          @write="(field, text) => $emit('write', field, text)"
          @save="$emit('save')"
          @close="$emit('close')"
        />
        <Card
          class="session__slot"
          :inert="editing"
          :front="card.front"
          :back="card.back"
          :shown="shown"
          :fresh="!card.seen"
          @show="$emit('show')"
        />
      </div>
    </div>

    <Answers :shown="shown" @show="$emit('show')" @answer="(how) => $emit('answer', how)" />
  </section>
</template>

<style scoped>
.session {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-block-size: 0;
  gap: var(--numen-inset);
}

/* Where the card stands, said once and quietly: a person answering is reading
   the card, not the line above it. Text and marks are centred against each
   other, because a button has no baseline to put a word on. */
.session__where {
  display: flex;
  align-items: center;
  gap: var(--numen-inset);
  color: var(--numen-edge-label);
  font-size: var(--numen-font-size);
}

.session__deck {
  color: var(--numen-node-fg);
  font-weight: 600;
}

.session__left {
  margin-inline-start: auto;
}

/* One frame, and a column of two standing in it that is twice as tall. Which of
   them is in the frame is where the column stands, so the two never overlap and
   the one going out is the one the other pushed. */
.session__stage {
  flex: 1;
  min-block-size: 0;
  overflow: hidden;
}

.session__column {
  display: flex;
  flex-direction: column;
  block-size: 200%;
  transform: translateY(-50%);
  transition: transform var(--numen-motion) var(--numen-easing);
}

.session__column--down {
  transform: translateY(0);
}

.session__slot {
  flex: none;
  block-size: 50%;
}
</style>
