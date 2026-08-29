<script setup lang="ts">
/**
 * The sitting: one card in front of a person, and the four ways of answering it.
 *
 * It holds nothing. Which card is up, whether the answer is showing and what
 * has been written are the sitting's, and this draws them.
 */
import { Undo2, X } from '@lucide/vue'
import { Button, KeyCap } from '@numen/ui'

import Card from './Card.vue'
import { ahead, called, deckName, said } from './core'
import type { Asked, Said } from './core'

defineProps<{
  card: Asked
  /** Whether the answer is showing. */
  shown: boolean
  /** How many cards are left to ask, this one among them. */
  left: number
  /** Whether there is an answer to take back. */
  takenBack: boolean
}>()

defineEmits<{
  (event: 'show'): void
  (event: 'answer', how: Said): void
  (event: 'takeBack'): void
  (event: 'leave'): void
}>()
</script>

<template>
  <section class="session">
    <header class="session__where">
      <span class="session__deck">{{ deckName(card.deck) }}</span>
      <span v-if="card.section">{{ card.section }}</span>
      <span>{{ card.face }}</span>
      <span v-if="!card.seen" class="session__new">new</span>
      <span class="session__left">{{ left }} left</span>

      <!-- What a person does beside answering, each one mark. They stand at the
           end of the line that says where the card is from. -->
      <Button
        variant="ghost"
        size="icon-small"
        title="Take the last answer back"
        aria-label="Take the last answer back"
        :disabled="!takenBack"
        @click="$emit('takeBack')"
      >
        <Undo2 />
      </Button>
      <Button
        variant="ghost"
        size="icon-small"
        title="Leave"
        aria-label="Leave"
        @click="$emit('leave')"
      >
        <X />
      </Button>
    </header>

    <!-- The card is what changes under a person as they work, so a reader that
         is not looking at the screen is told when the answer appears. -->
    <div class="session__card" aria-live="polite">
      <Card :front="card.front" :back="card.back" :shown="shown" @show="$emit('show')" />
    </div>

    <footer class="session__answers">
      <!-- The key first and the word after it: a person answering with the
           keyboard reads down the row of keys, and one answering with the mouse
           reads the words either way. -->
      <template v-if="shown">
        <Button
          v-for="(how, i) in said"
          :key="how"
          variant="outline"
          class="session__answer"
          @click="$emit('answer', how)"
        >
          <KeyCap :keys="{ marks: [], letter: String(i + 1) }" />
          {{ called[how] }}
          <!-- What the answer does to the card, said where the answer is
               chosen: a person picking between the four is picking between
               these. It is read off the screen and not out of the button's own
               name, which is the word a person means to press. -->
          <span v-if="card.ahead" class="session__ahead" aria-hidden="true">{{
            ahead(card.ahead[how])
          }}</span>
        </Button>
      </template>
      <Button v-else variant="outline" class="session__answer" @click="$emit('show')">
        <KeyCap :keys="{ marks: [], letter: 'space' }" />
        Show the answer
      </Button>
    </footer>
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

/* The card takes what the row of answers leaves, and it is the card itself that
   scrolls. */
.session__card {
  display: flex;
  flex: 1;
  min-block-size: 0;
}

/* Where the card stands, said once and quietly: a person answering is reading
   the card, not the line above it. Text and marks are centred against each
   other, because a button has no baseline to put a word on. */
.session__where {
  display: flex;
  align-items: center;
  gap: var(--numen-inset);
  color: var(--numen-hushed);
  font-size: var(--numen-font-size);
}

.session__deck {
  color: var(--numen-node-fg);
  font-weight: 600;
}

/* A pill in the same row as the one that counts what is waiting, so it is the
   same shape as that one. */
.session__new {
  padding: 0.0625rem 0.4rem;
  border-radius: var(--numen-radius-pill);
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
}

.session__left {
  margin-inline-start: auto;
}

.session__answers {
  display: flex;
  flex: none;
  gap: var(--numen-inset);
}

/* What the answer does, said quietly beside it: it is read once, when a person
   is learning what the four mean, and glanced at after that. */
.session__ahead {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  font-variant-numeric: tabular-nums;
}

/* An answer is a target a person hits without looking, so it takes the whole
   width it can and stands taller than a button in a row of controls. */
.session__answer {
  flex: 1;
  block-size: auto;
  padding-block: var(--numen-inset-wide);
}
</style>
