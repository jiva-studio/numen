<script setup lang="ts">
/**
 * What the goals of this vault come to today: one tile for each preset that
 * schedules something, saying what its goal is and how far through it the day
 * stands.
 *
 * A preset keeps a budget in cards and a budget in time. The day stands as far
 * through as the fuller of the two, which is the one a person is nearest the
 * end of.
 *
 * A preset nothing points at, and one whose decks hold no cards, are no part of
 * anybody's day: they are files being set up, and the editor's preset tab is
 * where they are read.
 */
import { computed } from 'vue'

import { goalWords, through } from './scheduling'
import type { Preset } from './scheduling'

const props = defineProps<{
  presets: readonly Preset[]
  /** The day this is being read on, as the year, the month and the day. */
  today: string
}>()

/** One preset's tile: its goal, and how far through the day it stands. */
interface Tile {
  readonly one: Preset
  readonly goal: string
  /** How much of the day is done, and above one for a day drawn past it. */
  readonly done: number
  /** How far through the day it stands, in the words the tile says it in. */
  readonly says: string
}

/**
 * Whether a preset schedules anything at all: a deck points at it, and those
 * decks hold cards. A preset that schedules nothing today is still one of
 * these, and says why.
 */
const schedules = (one: Preset): boolean => one.named > 0 && one.faces > 0

const tiles = computed<Tile[]>(() =>
  props.presets.filter(schedules).map((one) => {
    const done = through(one)
    return {
      one,
      goal: one.settings ? goalWords(one.settings, props.today) : '',
      done,
      says: done > 1 ? 'over budget' : percent(done),
    }
  }),
)

/** How far through the day a tile stands, as a person reads it. */
const percent = (done: number): string => `${Math.round(done * 100)}%`
</script>

<template>
  <section v-if="tiles.length" class="presets">
    <ul class="presets__list">
      <li
        v-for="tile in tiles"
        :key="tile.one.path"
        class="presets__preset"
        :class="{ 'presets__preset--paused': tile.one.paused }"
      >
        <p class="presets__name">{{ tile.one.name }}</p>
        <p class="presets__goal">{{ tile.one.paused || tile.goal }}</p>

        <!-- A day answered past its budget says so, rather than standing at a
             round number it is already past. -->
        <p
          v-if="!tile.one.paused"
          class="presets__done"
          :data-over="tile.done > 1 ? '' : undefined"
        >
          {{ tile.says }}
        </p>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.presets {
  display: flex;
  flex: none;
  flex-direction: column;
  gap: var(--numen-inset);
}

/* The tiles stand side by side, and wrap onto another line where the window is
   too narrow to hold them. */
.presets__list {
  display: flex;
  margin: 0;
  padding: 0;
  flex-wrap: wrap;
  gap: var(--numen-inset);
  list-style: none;
}

/* A tile is an island of the same make as a deck in the list below it: the same
   ground, the same rule around it, the same corner and the same inset. */
.presets__preset {
  display: flex;
  flex: 1 1 9rem;
  flex-direction: column;
  padding: var(--numen-inset);
  gap: 0.125rem;
  border: 1px solid var(--numen-node-border);
  border-radius: var(--numen-radius);
  background: var(--numen-node-bg);
  color: var(--numen-node-fg);
}

.presets__name {
  margin: 0;
  font-weight: 600;
}

.presets__goal {
  margin: 0;
  color: var(--numen-hushed);
}

/* What a person looks for, so it carries the weight in the tile and stands at
   the foot of it however tall the words above it run. */
.presets__done {
  margin: 0;
  margin-block-start: auto;
  padding-block-start: var(--numen-inset);
  font-size: var(--numen-title-size);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

/* A day over its budget is the one thing on the tile worth catching the eye,
   and it is words, which read at the size words do. */
.presets__done[data-over] {
  color: var(--numen-caution-fg);
  font-size: var(--numen-font-size);
}

/* A preset scheduling nothing today stands with its reason and nothing else. */
.presets__preset--paused .presets__name,
.presets__preset--paused .presets__goal {
  color: var(--numen-hushed);
}
</style>
