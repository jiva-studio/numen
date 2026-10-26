<script setup lang="ts">
/**
 * What a person has been doing in this vault: the days they answered on, the
 * weeks ahead of them, and what each day came to.
 *
 * It is read at a glance and stands above what they came here to do.
 */
import { Heatmap, dayName as heatmapDayName } from '../../../../ui/heatmap'
import type { Tally as HeatmapTally, Words as HeatmapWords } from '../../../../ui/heatmap'

import { GRADE_LABELS } from '../../../entities/card'

/* --------------------------------- Props ---------------------------------- */
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
