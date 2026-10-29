<script setup lang="ts">
/**
 * The progress of this vault's goals today: one tile for each preset that
 * schedules something, showing its goal and how far through it the day is.
 *
 * A preset has a budget in cards and a budget in time. The day's progress is
 * the larger of the two fractions, which is the budget closest to being used.
 *
 * A preset that no deck points at, and one whose decks hold no cards, are not
 * part of the day: they are being set up, and the editor's preset tab is where
 * they are shown.
 */
import { computed } from 'vue'

import { PresetTile } from './preset-tile'
import { getTiles } from '../lib/tiles'
import type { Preset } from '../types'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  presets: readonly Preset[]
  /** The day this is being read on, as the year, the month and the day. */
  today: string
}>()

/* --------------------------------- Events --------------------------------- */
defineEmits<{
  (event: 'start', preset: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const tiles = computed(() => getTiles(props.presets, props.today))
</script>

<template>
  <section v-if="tiles.length" class="presets">
    <ul class="presets__list" aria-label="What the goals of this vault come to today">
      <li v-for="tile in tiles" :key="tile.one.path" class="presets__tile">
        <PresetTile :tile="tile" @start="$emit('start', tile.one.path)" />
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

.presets__tile {
  display: flex;
  flex: 1 1 12rem;
}
</style>
