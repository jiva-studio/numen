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

    <!-- The boxes a card is put right in take their own room at the top, and
         the card is pushed down and stays where a person can read it. -->
    <div class="session__stage" :class="{ 'session__stage--editing': editing }">
      <div class="session__slot" :inert="!editing">
        <Editing
          :card="card.card"
          :section="card.section"
          :values="values"
          :writing="writing"
          @write="(field, text) => $emit('write', field, text)"
          @save="$emit('save')"
          @close="$emit('close')"
        />
      </div>
      <div class="session__slot">
        <Card
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

/* Two rows sharing what the frame has. The first is closed until the boxes are
   asked for, and opening it takes its room from the second: the boxes come down
   and the card goes under them by exactly as much, and stays read. */
.session__stage {
  display: grid;
  flex: 1;
  min-block-size: 0;
  grid-template-rows: 0fr 1fr;
  gap: 0;
  transition:
    grid-template-rows var(--numen-motion) var(--numen-easing),
    gap var(--numen-motion) var(--numen-easing);
}

.session__stage--editing {
  grid-template-rows: 1fr 1fr;
  gap: var(--numen-inset);
}

.session__slot {
  display: flex;
  min-block-size: 0;
  overflow: hidden;
}

.session__slot > * {
  flex: 1;
  min-block-size: 0;
}
</style>
