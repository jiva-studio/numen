<script setup lang="ts">
/**
 * The user's activity in this vault: the days they answered on, the coming
 * weeks, and the tally for each day.
 *
 * It is read at a glance and is placed above the main content.
 */
import Heatmap from '../heatmap/Heatmap.vue'
import { dayName as heatmapDayName } from '../../lib/heatmap/dates'
import type { Tally as HeatmapTally } from '../../lib/heatmap/heatmap'
import type { Words as HeatmapWords } from '../../lib/heatmap/words'

import { GRADE_LABELS } from '../../lib/card/grades'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  days: ReadonlyMap<string, HeatmapTally>
  due: ReadonlyMap<string, number>
}>()

/** The grid labels for a day, in this window's wording. */
const words: HeatmapWords = {
  names: heatmapDayName,
  reviewed: 'answered',
  nothing: 'Nothing answered',
  toCome: 'to come',
  again: GRADE_LABELS.again,
  hard: GRADE_LABELS.hard,
  good: GRADE_LABELS.good,
  easy: GRADE_LABELS.easy,
  recalled: 'of cards you are reviewing came back',
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
