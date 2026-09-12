<script setup lang="ts">
/**
 * The session: one card in front of a person, and the four ways of answering it.
 *
 * It holds nothing. Which card is up, whether the answer is showing and what
 * has been written are the session's, and this draws them.
 */
import { Button, KeyCap, keyChord } from '@numen/ui'

import PanelCarousel from './PanelCarousel.vue'
import Card from './Card.vue'
import type { PanelPlace } from '../model/carousel'
import { called, getTimeAhead, grades } from '@/entities/card'
import { deckName } from '@/entities/vault'
import { ASKS, READS } from '@/features/keyboard'
import type { CardFace, Grade } from '@/entities/card'

defineProps<{
  card: CardFace
  /** Whether the answer is showing. */
  shown: boolean
  /** How many cards are left to ask, this one among them. */
  left: number
  /** Whether there is an answer to take back. */
  takenBack: boolean
}>()

/** Which of the card and the panels either side of it is in the window. */
const at = defineModel<PanelPlace>('at', { required: true })

/** The panels are held with the overlay key, drawn as this machine's own. */
const chord = (letter: string) => keyChord(letter, navigator.userAgent)

defineEmits<{
  (event: 'show'): void
  (event: 'answer', how: Grade): void
  (event: 'takeBack'): void
  (event: 'leave'): void
  (event: 'ask'): void
  (event: 'read', note: string): void
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

      <!-- What a person does beside answering, each one carrying the key it is
           done with. They stand at the end of the line that says where the card
           is from. -->
      <Button variant="ghost" size="small" :disabled="!takenBack" @click="$emit('takeBack')">
        <KeyCap :keys="{ icons: [], letter: 'u' }" />
        Undo
      </Button>
      <!-- The two panels stand in the order they stand in the strip: what is
           read is to the left of the card, and what is asked to the right. -->
      <Button variant="ghost" size="small" @click="$emit('read', '')">
        <KeyCap :keys="chord(READS)" />
        Read
      </Button>
      <Button variant="ghost" size="small" @click="$emit('ask')">
        <KeyCap :keys="chord(ASKS)" />
        Ask
      </Button>
      <Button variant="ghost" size="small" @click="$emit('leave')">
        <KeyCap :keys="{ icons: [], letter: 'esc' }" />
        Leave
      </Button>
    </header>

    <!-- The card is what changes under a person as they work, so a reader that
         is not looking at the screen is told when the answer appears. -->
    <PanelCarousel v-model:at="at">
      <template #before><slot name="reading" /></template>
      <div class="session__card" aria-live="polite">
        <Card
          :front="card.front"
          :back="card.back"
          :shown="shown"
          @show="$emit('show')"
          @read="(note: string) => $emit('read', note)"
        />
      </div>
      <template #after><slot name="panel" /></template>
    </PanelCarousel>

    <footer class="session__answers">
      <!-- The key first and the word after it: a person answering with the
           keyboard reads down the row of keys, and one answering with the mouse
           reads the words either way. -->
      <template v-if="shown">
        <Button
          v-for="(how, i) in grades"
          :key="how"
          variant="outline"
          class="session__answer"
          @click="$emit('answer', how)"
        >
          <KeyCap :keys="{ icons: [], letter: String(i + 1) }" />
          {{ called[how] }}
          <!-- What the answer does to the card, said where the answer is
               chosen: a person picking between the four is picking between
               these. It is read off the screen and not out of the button's own
               name, which is the word a person means to press. -->
          <span v-if="card.ahead" class="session__ahead" aria-hidden="true">{{
            getTimeAhead(card.ahead[how])
          }}</span>
        </Button>
      </template>
      <Button v-else variant="outline" class="session__answer" @click="$emit('show')">
        <KeyCap :keys="{ icons: [], letter: 'space' }" />
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

/* It is the card itself that scrolls. */
.session__card {
  display: flex;
  flex: 1;
  min-inline-size: 0;
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
  color: var(--numen-ink);
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
