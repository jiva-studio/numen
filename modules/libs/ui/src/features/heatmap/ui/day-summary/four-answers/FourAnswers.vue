<script setup lang="ts">
/**
 * The four answers of one day, and how many of each. An answer nobody gave
 * that day is left out, and a day with none of them draws nothing.
 */
import { computed } from 'vue'

import type { Day } from '../../../lib/heatmap'
import type { Words } from '../../../lib/words'

const props = defineProps<{
  day: Day
  /** What each answer is called, in the person's own language. */
  words: Words
}>()

const four = computed(() =>
  [
    { label: props.words.again, count: props.day.again, tone: 'again' },
    { label: props.words.hard, count: props.day.hard, tone: 'hard' },
    { label: props.words.good, count: props.day.good, tone: 'good' },
    { label: props.words.easy, count: props.day.easy, tone: 'easy' },
  ].filter((one) => one.count > 0),
)
</script>

<template>
  <ul v-if="four.length" class="day-summary__four" data-day-summary="four">
    <li v-for="one in four" :key="one.tone" :data-tone="one.tone">
      <span class="day-summary__said" data-day-summary="said">{{ one.label }}</span>
      <span class="day-summary__how-many" data-day-summary="how-many">{{ one.count }}</span>
    </li>
  </ul>
</template>

<style scoped>
.day-summary__four {
  display: flex;
  margin: var(--numen-inset) 0 0;
  padding: 0;
  flex-direction: column;
  gap: 0.125rem;
  list-style: none;
}

.day-summary__four li {
  display: flex;
  justify-content: space-between;
  gap: var(--numen-inset);
}

/* The word for each answer is drawn in what that answer means. */
.day-summary__four li[data-tone='again'] .day-summary__said {
  color: var(--numen-alarm);
}

.day-summary__four li[data-tone='hard'] .day-summary__said {
  color: var(--numen-caution-fg);
}

.day-summary__how-many {
  font-variant-numeric: tabular-nums;
}
</style>
