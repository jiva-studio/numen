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

import { PresetTile } from './preset-tile'
import { getTiles } from '../lib/tiles'
import type { Preset } from '../types'

const props = defineProps<{
  presets: readonly Preset[]
  /** The day this is being read on, as the year, the month and the day. */
  today: string
}>()

defineEmits<{
  /** Sit down to every deck this preset schedules, by the note it stands in. */
  (event: 'start', preset: string): void
}>()

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
