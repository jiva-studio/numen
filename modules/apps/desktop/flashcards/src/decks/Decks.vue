<script setup lang="ts">
/**
 * One vault's decks, and what each owes today.
 *
 * The whole vault is the ordinary way to sit down to this, so it stands at the
 * top as one button. A deck below it is for the person who came for that deck.
 */
import { computed } from 'vue'
import { Button, DueCount, KeyCap, Skeleton } from '@numen/ui'
import type { HeatmapTally } from '@numen/ui'
import Progress from './Progress.vue'
import Presets from './Presets.vue'
import { deckName } from '../core'
import { letterOf } from '../keying'
import { beginsNothing, learned, opens, spent, LEARNED, STOPPED } from './scheduling'
import type { DeckCardsDue, VaultCardsDue } from '../core'
import type { Preset } from './scheduling'

const props = defineProps<{
  vault: VaultCardsDue
  /** How many cards were answered on each day, by the day it was. */
  days: ReadonlyMap<string, HeatmapTally>
  /** How many cards fall on each day still to come, by the day they fall on. */
  due: ReadonlyMap<string, number>
  /** The presets this vault's decks are scheduled by. */
  presets: readonly Preset[]
  /** The preset each deck is scheduled by, by the path the deck is filed under. */
  byDeck: ReadonlyMap<string, Preset>
  /** Whether those presets have been read, which a deck missing from them needs. */
  scheduled: boolean
  /** The day this is being read on, as the year, the month and the day. */
  today: string
}>()

defineEmits<{
  (event: 'start', deck: string): void
  /** Sit down to every deck one preset schedules, by the note it stands in. */
  (event: 'startPreset', preset: string): void
  (event: 'back'): void
}>()

/** What the whole vault owes: what is due today and what has never been asked. */
const owed = computed(() => props.vault.due + props.vault.new)

/** Why a deck is not studied today, and empty while its preset schedules it. */
const stopped = (deck: string): string => props.byDeck.get(deck)?.paused ?? ''

/**
 * Whether a deck's day is done, which is the day's work met and not merely
 * nothing standing: something was answered under its preset today, and its
 * preset still had room for more.
 */
const done = (deck: DeckCardsDue): boolean => {
  const one = props.byDeck.get(deck.deck)
  return !!one && !spent(one) && one.answered > 0
}

/**
 * What a deck with nothing left says where the day's work was not what emptied
 * it: the preset's day is spent, nothing here can be begun at all, or the day
 * held nothing of this deck.
 */
const empty = (deck: DeckCardsDue): string => {
  const one = props.byDeck.get(deck.deck)
  if (one && spent(one)) return STOPPED.full
  return beginsNothing(deck, one) ? STOPPED.beginsNothing : STOPPED.nothing
}

/**
 * How much of a deck stands learned, and empty where the share cannot be said:
 * a deck holding no card face is a share of nothing, and a deck whose preset
 * was not read has no rule to be counted by.
 */
const share = (deck: DeckCardsDue): string => {
  if (props.scheduled && !props.byDeck.has(deck.deck)) return LEARNED.unruled
  const of = learned(deck)
  return of === null ? '' : LEARNED.share(of)
}
</script>

<template>
  <section class="decks">
    <h1 class="decks__title">{{ vault.name }}</h1>

    <Progress :days="days" :due="due" />

    <Presets :presets="presets" :today="today" @start="$emit('startPreset', $event)" />

    <p v-if="!vault.decks.length" class="decks__saying">This vault holds no deck.</p>

    <ul v-else class="decks__list">
      <li v-for="(deck, at) in vault.decks" :key="deck.deck">
        <Button
          variant="outline"
          class="decks__deck"
          :disabled="!opens(deck, byDeck)"
          @click="$emit('start', deck.deck)"
        >
          <!-- The letter it is picked by, where the alphabet reaches it: a
               person reads down the list and presses what they see. -->
          <KeyCap v-if="letterOf(at)" :keys="{ icons: [], letter: letterOf(at) }" />
          <span class="decks__name">
            {{ deckName(deck.deck) }}
            <!-- Which preset schedules it. -->
            <span v-if="byDeck.get(deck.deck)" class="decks__by">{{
              byDeck.get(deck.deck)?.name
            }}</span>
          </span>
          <!-- How much of the deck stands learned, under the rule its own
               preset counts by. The word is said with the figure: a bare share
               on this screen is how far through its day a preset stands. -->
          <Skeleton v-if="!vault.counted" class="decks__learned" wide="3.5rem" high="0.7em" />
          <span v-else-if="share(deck)" class="decks__learned">{{ share(deck) }}</span>

          <!-- What is left of this deck today, or why nothing is. Having
               nothing due is not having finished, so the day's work is only
               met where something was answered. A deck holding no cards at all
               says neither. -->
          <span v-if="stopped(deck.deck)" class="decks__stopped">{{ stopped(deck.deck) }}</span>
          <DueCount v-else-if="deck.due + deck.new > 0" :due="deck.due + deck.new" />
          <template v-else-if="deck.faces > 0">
            <span v-if="done(deck)" class="decks__met">Done today</span>
            <span v-else class="decks__stopped">{{ empty(deck) }}</span>
          </template>
        </Button>
      </li>
    </ul>

    <!-- What a person came here to do stands where the hand is, under the list
         they read: sit down to the whole vault, or go and pick another. -->
    <footer class="decks__deeds">
      <Button variant="ghost" @click="$emit('back')">
        <KeyCap :keys="{ icons: [], letter: 'esc' }" />
        Another vault
      </Button>
      <Button class="decks__all" :disabled="owed === 0" @click="$emit('start', '')">
        <KeyCap :keys="{ icons: [], letter: 'enter' }" />
        Review
        <DueCount :due="owed" bare over />
      </Button>
    </footer>
  </section>
</template>

<style scoped>
.decks {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-block-size: 0;
  gap: var(--numen-inset-wide);
}

.decks__title {
  flex: none;
  margin: 0;
  font-size: var(--numen-title-size);
  font-weight: 600;
}

/* What is done stands at the foot of the screen, whatever is above it. */
.decks__deeds {
  display: flex;
  flex: none;
  margin-block-start: auto;
  align-items: center;
  justify-content: space-between;
  gap: var(--numen-inset);
}

/* The whole vault is the daily act, so it is the one button filled in. It
   stands as tall as the one beside it: two things done from one line. */
.decks__all {
  font-weight: 600;
}

.decks__saying {
  margin: 0;
  color: var(--numen-hushed);
}

.decks__list {
  display: flex;
  flex: 1;
  margin: 0;
  padding: 0;
  flex-direction: column;
  min-block-size: 0;
  overflow-y: auto;
  gap: var(--numen-inset);
  list-style: none;
}

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
