<script setup lang="ts">
/**
 * One deck of a vault: what it is called, which preset schedules it, how much
 * of it stands learned, and what it owes today.
 */
import { computed } from 'vue'
import { Button, DueCount, KeyCap, Skeleton } from '@numen/ui'

import { getDeckName } from '@/entities/vault'
import { getLetter } from '@/features/keyboard'
import { getLearnedShare, hasNothingToBegin, isSpent } from '../../lib/progress'
import { LEARNED, STOPPED } from '../../words'
import type { DeckCardsDue } from '@/entities/vault'
import type { Preset } from '../../types'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  deck: DeckCardsDue
  /** at its current position in the list, which is the letter it is picked by. */
  index: number
  /** The preset scheduling it, and nothing where none was read. */
  preset: Preset | undefined
  /** Whether the vault's presets have been read, which a deck missing from them needs. */
  hasPresets: boolean
  /** Whether the vault has been counted. */
  isCounted: boolean
  /** Whether it can be sat down to. */
  canOpen: boolean
}>()

/* --------------------------------- Events --------------------------------- */
defineEmits<{
  (event: 'start'): void
}>()

/* --------------------------------- State ---------------------------------- */
/** Why the deck is not studied today, and empty while its preset schedules it. */
const paused = computed(() => props.preset?.paused ?? '')

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
const stoppedLabel = computed(() => {
  if (props.by && isSpent(props.by)) return STOPPED.full
  return hasNothingToBegin(props.deck, props.by) ? STOPPED.noneToBegin : STOPPED.nothing
})

/**
 * How much of the deck stands learned, and empty where the share cannot be
 * said: a deck holding no card face is a share of nothing, and a deck whose
 * preset was not read has no rule to be counted by.
 */
const share = computed(() => {
  if (props.hasPresets && !props.by) return LEARNED.unruled
  const of = getLearnedShare(props.deck)
  return of === null ? '' : LEARNED.share(of)
})
</script>

<template>
  <Button variant="outline" class="decks__deck" :disabled="!canOpen" @click="$emit('start')">
    <!-- The letter it is picked by, where the alphabet reaches it: a
         person reads down the list and presses what they see. -->
    <KeyCap v-if="getLetter(at)" :keys="{ icons: [], letter: getLetter(at) }" />
    <span class="decks__name">
      {{ getDeckName(deck.deck) }}
      <!-- Which preset schedules it. -->
      <span v-if="by" class="decks__by">{{ by.name }}</span>
    </span>
    <!-- How much of the deck stands learned, under the rule its own
         preset counts by. The word is said with the figure: a bare share
         on this screen is how far through its day a preset stands. -->
    <Skeleton v-if="!isCounted" class="decks__learned" wide="3.5rem" high="0.7em" />
    <span v-else-if="share" class="decks__learned">{{ share }}</span>

    <!-- What is left of this deck today, or why nothing is. Having
         nothing due is not having finished, so the day's work is only
         met where something was answered. A deck holding no cards at all
         says neither. -->
    <span v-if="paused" class="decks__stopped">{{ paused }}</span>
    <DueCount v-else-if="deck.due + deck.new > 0" :due="deck.due + deck.new" />
    <template v-else-if="deck.faces > 0">
      <span v-if="isDone" class="decks__met">Done today</span>
      <span v-else class="decks__stopped">{{ stoppedLabel }}</span>
    </template>
  </Button>
</template>

<style scoped>
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
