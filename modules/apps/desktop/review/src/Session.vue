<script setup lang="ts">
/**
 * One sitting: where the card is from, the card itself, and how it came back.
 *
 * The card and the boxes it is put right in stand in one place and change over,
 * so a person putting a card right does not lose sight of where they were.
 */
import { Button } from '@numen/ui'
import { Check, SquarePen, Trash2, Undo2, X } from '@lucide/vue'
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
  /** The stencil that cuts it, as the read of the card gave it. */
  stencil: string
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
  (event: 'remove'): void
}>()
</script>

<template>
  <section class="session">
    <header class="session__where">
      <span class="session__deck">{{ deckName(card.deck) }}</span>
      <span v-if="card.section">{{ card.section }}</span>
      <span>{{ card.face }}</span>
      <span class="session__left">{{ left }} left</span>

      <!-- This line is the sitting's, so what stands at the end of it is what
           is done to the sitting. What is done to a card stands on the card. -->
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
      <div class="session__slot session__slot--boxes" :inert="!editing">
        <Editing
          :card="card.card"
          :section="card.section"
          :stencil="stencil"
          :values="values"
          @write="(field, text) => $emit('write', field, text)"
        >
          <template #deeds>
            <Button
              variant="ghost"
              size="icon-small"
              title="Save this card"
              :disabled="writing"
              @click="$emit('save')"
            >
              <Check />
            </Button>
            <Button
              variant="ghost"
              size="icon-small"
              title="Leave it as it was"
              @click="$emit('close')"
            >
              <X />
            </Button>
            <Button
              variant="ghost"
              size="icon-small"
              title="Take this card out of its deck"
              :disabled="writing"
              @click="$emit('remove')"
            >
              <Trash2 />
            </Button>
          </template>
        </Editing>
      </div>
      <div class="session__slot session__slot--card">
        <Card
          :front="card.front"
          :back="card.back"
          :shown="shown"
          :stencil="stencil"
          :fresh="!card.seen"
          @show="$emit('show')"
        >
          <template #deeds>
            <Button
              variant="ghost"
              size="icon-small"
              title="Edit this card"
              @click="$emit('edit')"
            >
              <SquarePen />
            </Button>
          </template>
        </Card>
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

/* The boxes are closed until they are asked for. Opening them takes the room
   they need and no more, and the card goes under them by exactly as much and
   stays where a person can read it. */
.session__stage {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-block-size: 0;
  gap: 0;
  transition: gap var(--numen-motion) var(--numen-easing);
}

.session__stage--editing {
  gap: var(--numen-inset);
}

.session__slot {
  display: flex;
  min-block-size: 0;
}

.session__slot > * {
  flex: 1;
  min-block-size: 0;
}

/* Half the frame is as far as the boxes go; a card of many fields scrolls
   inside that. */
.session__slot--boxes {
  flex: none;
  max-block-size: 0;
  overflow: hidden;
  transition: max-block-size var(--numen-motion) var(--numen-easing);
}

.session__stage--editing .session__slot--boxes {
  max-block-size: 50%;
}

.session__slot--card {
  flex: 1;
}
</style>
