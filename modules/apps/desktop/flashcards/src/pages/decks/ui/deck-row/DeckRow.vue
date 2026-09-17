<script setup lang="ts">
/**
 * One deck of a vault: what it is called, which preset schedules it, how much
 * of it stands learned, and what it owes today.
 */
import { computed } from 'vue'
import { Button, DueCount, KeyCap, Skeleton } from '@numen/ui'

import { deckName } from '@/entities/vault'
import { letterOf } from '@/features/keyboard'
import { getLearnedShare, hasNothingToBegin, isSpent } from '../../lib/progress'
import { LEARNED, STOPPED } from '../../words'
import type { DeckCardsDue } from '@/entities/vault'
import type { Preset } from '../../types'

const props = defineProps<{
  deck: DeckCardsDue
  /** Where it stands in the list, which is the letter it is picked by. */
  at: number
  /** The preset scheduling it, and nothing where none was read. */
  by: Preset | undefined
  /** Whether the vault's presets have been read, which a deck missing from them needs. */
  scheduled: boolean
  /** Whether the vault has been counted. */
  counted: boolean
  /** Whether it can be sat down to. */
  canOpen: boolean
}>()

defineEmits<{
  (event: 'start'): void
}>()

/** Why the deck is not studied today, and empty while its preset schedules it. */
const paused = computed(() => props.by?.paused ?? '')

/**
 * Whether the deck's day is done, which is the day's work met and not merely
 * nothing standing: something was answered under its preset today, and its
 * preset still had room for more.
 */
const isDone = computed(() => !!props.by && !isSpent(props.by) && props.by.answered > 0)

/**
 * What a deck with nothing left says where the day's work was not what emptied
 * it: the preset's day is spent, nothing here can be begun at all, or the day
 * held nothing of this deck.
 */
const stopped = computed(() => {
  if (props.by && isSpent(props.by)) return STOPPED.full
  return hasNothingToBegin(props.deck, props.by) ? STOPPED.noneToBegin : STOPPED.nothing
})

/**
 * How much of the deck stands learned, and empty where the share cannot be
 * said: a deck holding no card face is a share of nothing, and a deck whose
 * preset was not read has no rule to be counted by.
 */
const share = computed(() => {
  if (props.scheduled && !props.by) return LEARNED.unruled
  const of = getLearnedShare(props.deck)
  return of === null ? '' : LEARNED.share(of)
})
</script>

<template>
  <Button variant="outline" class="decks__deck" :disabled="!canOpen" @click="$emit('start')">
    <!-- The letter it is picked by, where the alphabet reaches it: a
         person reads down the list and presses what they see. -->
    <KeyCap v-if="letterOf(at)" :keys="{ icons: [], letter: letterOf(at) }" />
    <span class="decks__name">
      {{ deckName(deck.deck) }}
      <!-- Which preset schedules it. -->
      <span v-if="by" class="decks__by">{{ by.name }}</span>
    </span>
    <!-- How much of the deck stands learned, under the rule its own
         preset counts by. The word is said with the figure: a bare share
         on this screen is how far through its day a preset stands. -->
    <Skeleton v-if="!counted" class="decks__learned" wide="3.5rem" high="0.7em" />
    <span v-else-if="share" class="decks__learned">{{ share }}</span>

    <!-- What is left of this deck today, or why nothing is. Having
         nothing due is not having finished, so the day's work is only
         met where something was answered. A deck holding no cards at all
         says neither. -->
    <span v-if="paused" class="decks__stopped">{{ paused }}</span>
    <DueCount v-else-if="deck.due + deck.new > 0" :due="deck.due + deck.new" />
    <template v-else-if="deck.faces > 0">
      <span v-if="isDone" class="decks__met">Done today</span>
      <span v-else class="decks__stopped">{{ stopped }}</span>
    </template>
  </Button>
</template>

<style scoped>
/* A row of the list is a button the width of the list, taller than one in a row
   of controls and reading from its start. What it is painted, how it answers a
   hover and what it looks like with the keyboard on it are the button's own. */
.decks__deck {
  inline-size: 100%;
  block-size: auto;
  justify-content: start;
  padding: var(--numen-inset);
  gap: var(--numen-inset);
  text-align: start;
}

.decks__name {
  display: flex;
  flex: 1;
  align-items: baseline;
  gap: var(--numen-inset);
}

.decks__by {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

/* The share stands to the left of what the deck owes, in the screen's quiet
   voice: it is read on the way to the counts and never instead of them. */
.decks__learned {
  flex: none;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.decks__stopped,
.decks__met {
  flex: none;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  white-space: nowrap;
}
</style>
