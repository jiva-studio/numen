<script setup lang="ts">
/**
 * The session: one card, and the four ways of answering it.
 *
 * It holds no state. Which card is current, whether the answer is displayed and
 * what has been written belong to the session model, and this component renders
 * them.
 */
import { Button, KeyCap, keyChord, nav } from '@numen/ui'

import PanelCarousel from './PanelCarousel.vue'
import Card from './Card.vue'
import { AnswerButton } from './answer-button'
import type { PanelPlace } from '../../model/session/carousel'
import { grades } from '../../lib/card/grades'
import { getDeckName } from '../../lib/vault/names'
import { ASK_KEY, READ_KEY } from '../../lib/keyboard/keys'
import type { ReviewCard } from '../../lib/card/cards'
import type { Grade } from '../../lib/card/grades'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  card: ReviewCard
  isShown: boolean
  left: number
  canTakeBack: boolean
}>()

/** Which of the card and the panels either side of it is in the window. */
const currentPlace = defineModel<PanelPlace>('at', { required: true })

/* --------------------------------- Events --------------------------------- */
defineEmits<{
  (event: 'show'): void
  (event: 'answer', how: Grade): void
  (event: 'take-back'): void
  (event: 'leave'): void
  (event: 'ask'): void
  (event: 'read', note: string): void
}>()

/* -------------------------------- Helpers --------------------------------- */
const chord = (letter: string) => keyChord(letter, nav.userAgent)
</script>

<template>
  <section class="session">
    <header class="session__where">
      <span class="session__deck">{{ getDeckName(card.address.deck) }}</span>
      <span v-if="card.address.section">{{ card.address.section }}</span>
      <span>{{ card.address.template }}</span>
      <span v-if="card.isNew" class="session__new">new</span>
      <span class="session__left">{{ left }} left</span>

      <!-- What a person does beside answering, each one carrying the key it is
           done with. They stand at the end of the line that says where the card
           is from. -->
      <Button variant="ghost" size="small" :disabled="!canTakeBack" @click="$emit('take-back')">
        <KeyCap :keys="{ icons: [], letter: 'u' }" />
        Undo
      </Button>
      <!-- The two panels stand in the order they stand in the strip: what is
           read is to the left of the card, and what is asked to the right. -->
      <Button variant="ghost" size="small" @click="$emit('read', '')">
        <KeyCap :keys="chord(READ_KEY)" />
        Read
      </Button>
      <Button variant="ghost" size="small" @click="$emit('ask')">
        <KeyCap :keys="chord(ASK_KEY)" />
        Ask
      </Button>
      <Button variant="ghost" size="small" @click="$emit('leave')">
        <KeyCap :keys="{ icons: [], letter: 'esc' }" />
        Leave
      </Button>
    </header>

    <!-- The card is what changes under a person as they work, so a reader that
         is not looking at the screen is told when the answer appears. -->
    <PanelCarousel v-model:at="currentPlace">
      <template #before><slot name="reading" /></template>
      <div class="session__card" aria-live="polite">
        <Card
          :front="card.front"
          :back="card.back"
          :is-shown="isShown"
          @show="$emit('show')"
          @read="(note: string) => $emit('read', note)"
        />
      </div>
      <template #after><slot name="panel" /></template>
    </PanelCarousel>

    <footer class="session__answers">
      <!-- The key first and the word after it: user review input with the
           keyboard reads down the row of keys, and one answering with the mouse
           reads the words either way. -->
      <template v-if="isShown">
        <AnswerButton
          v-for="(how, i) in grades"
          :key="how"
          class="session__answer"
          :how="how"
          :shortcut-number="i + 1"
          :ahead="card.ahead"
          @answer="$emit('answer', how)"
        />
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
  --new-padding-block: 0.0625rem;
  --new-padding-inline: 0.4rem;
  padding: var(--new-padding-block) var(--new-padding-inline);
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

.session__answer {
  flex: 1;
  block-size: auto;
  padding-block: var(--numen-inset-wide);
}
</style>
