<script setup lang="ts">
/**
 * One deck of a vault: its name, the preset that schedules it, how much of it
 * is learned, and the number of cards due today.
 */
import { computed } from 'vue'
import { Button, KeyCap, Skeleton } from '@numen/ui'
import { DueCount } from '../../../../../ui/cards'

import { getDeckName } from '../../../../entities/vault'
import { getLetter } from '../../../../features/keyboard'
import { getLearnedShare, hasNothingToBegin, isSpent } from '../../lib/progress'
import { LEARNED, STOPPED } from '../../words'
import type { DeckCardsDue } from '../../../../entities/vault'
import type { Preset } from '../../types'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  deck: DeckCardsDue
  /** The position of the deck in the list, which determines the letter that picks it. */
  index: number
  /** The preset scheduling it, and nothing where none was read. */
  preset: Preset | undefined
  /** Whether the vault's presets have been read, which a deck missing from them needs. */
  hasPresets: boolean
  /** Whether the vault has been counted. */
  isCounted: boolean
  /** Whether a session can be started on it. */
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
 * Whether the deck's day is complete: the day's work was met and not merely
 * nothing remains. Something was answered under its preset today, and the
 * preset still had budget for more.
 */
const isDone = computed(() => !!props.preset && !isSpent(props.preset) && props.preset.answered > 0)

/**
 * The label of a deck with no cards remaining where the day's work did not
 * empty it: the preset's budget for the day is spent, nothing in the deck can
 * be begun, or the day had no cards of this deck.
 */
const stoppedLabel = computed(() => {
  if (props.preset && isSpent(props.preset)) return STOPPED.full
  return hasNothingToBegin(props.deck, props.preset) ? STOPPED.noneToBegin : STOPPED.nothing
})

/**
 * How much of the deck is learned, or empty where the share cannot be
 * computed: a deck with no review cards has no share, and a deck whose preset
 * was not read has no rule to count by.
 */
const share = computed(() => {
  if (props.hasPresets && !props.preset) return LEARNED.unruled
  const of = getLearnedShare(props.deck)
  return of === null ? '' : LEARNED.share(of)
})
</script>

<template>
  <Button variant="outline" class="decks__deck" :disabled="!canOpen" @click="$emit('start')">
    <!-- The letter it is picked by, where the alphabet reaches it: a
         person reads down the list and presses what they see. -->
    <KeyCap v-if="getLetter(index)" :keys="{ icons: [], letter: getLetter(index) }" />
    <span class="decks__name">
      {{ getDeckName(deck.deck) }}
      <!-- Which preset schedules it. -->
      <span v-if="preset" class="decks__by">{{ preset.name }}</span>
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
    <template v-else-if="deck.reviewCards > 0">
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
