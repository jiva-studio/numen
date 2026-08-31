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
  /**
   * What stands under the name: the goal, or why it schedules nothing. A preset
   * scheduling nothing is asked for its reason, not for what it was aiming at.
   */
  readonly goal: string
  /** What stands at the right of the tile: the figure, or words in its place. */
  readonly says: string
  /** Whether the day is past its budget, which is the one thing to catch the eye. */
  readonly over: boolean
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
    const over = done > 1
    return {
      one,
      goal: one.paused || (one.settings ? goalWords(one.settings, props.today) : ''),
      says: over ? 'over budget' : percent(done),
      over,
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
        <span class="presets__said">
          <span class="presets__name">{{ tile.one.name }}</span>
          <span v-if="tile.goal" class="presets__goal">{{ tile.goal }}</span>
        </span>

        <!-- What today comes to under it. A preset scheduling nothing has said
             so under its name, and stands here empty. -->
        <span
          v-if="!tile.one.paused"
          class="presets__done"
          :data-over="tile.over ? '' : undefined"
          >{{ tile.says }}</span
        >
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
   ground, the same rule around it, the same corner and the same inset. It reads
   across, with what the day comes to at its right. */
.presets__preset {
  display: flex;
  flex: 1 1 12rem;
  align-items: center;
  justify-content: space-between;
  padding: var(--numen-inset);
  gap: var(--numen-inset);
  border: 1px solid var(--numen-node-border);
  border-radius: var(--numen-radius);
  background: var(--numen-node-bg);
  color: var(--numen-node-fg);
}

/* The name, with the goal under it. */
.presets__said {
  display: flex;
  min-inline-size: 0;
  flex-direction: column;
  gap: 0.125rem;
}

.presets__name {
  font-weight: 600;
}

.presets__goal {
  color: var(--numen-hushed);
}

/* What a person looks for, so it carries the weight in the tile and stands at
   the end of the line, against the middle of the two above it. */
.presets__done {
  flex: none;
  font-size: var(--numen-title-size);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

/* A day over its budget is the one thing on the tile worth catching the eye,
   and it is words, which read at the size words do. */
.presets__done[data-over] {
  color: var(--numen-caution-fg);
  font-size: var(--numen-font-size);
  text-align: end;
}

/* A preset scheduling nothing today stands with its reason and nothing else. */
.presets__preset--paused .presets__name,
.presets__preset--paused .presets__goal {
  color: var(--numen-hushed);
}
</style>
