<script setup lang="ts">
/**
 * What a person has been doing in this vault: the days they answered on, the
 * weeks ahead of them, and what each day came to.
 *
 * It is read at a glance and stands above what they came here to do.
 */
import { Heatmap, heatmapDayName } from '@numen/ui'
import type { HeatmapTally, HeatmapWords } from '@numen/ui'

import { called } from './core'

defineProps<{
  /** What was answered on each day, by the day it was. */
  days: ReadonlyMap<string, HeatmapTally>
  /** How many cards fall on each day still to come, by the day they fall on. */
  due: ReadonlyMap<string, number>
}>()

/** What the grid says about a day, in this window's words. */
const words: HeatmapWords = {
  names: heatmapDayName,
  answered: 'answered',
  nothing: 'Nothing answered',
  toCome: 'to come',
  again: called.again,
  hard: called.hard,
  good: called.good,
  easy: called.easy,
  recalled: 'recalled',
}
</script>

<template>
  <section class="progress">
    <Heatmap :did="days" :due="due" :words="words" />
  </section>
</template>

<style scoped>
/* The grid keeps its own height, whatever the list below it does. */
.progress {
  display: flex;
  flex: none;
  flex-direction: column;
  gap: var(--numen-inset);
}
</style>
