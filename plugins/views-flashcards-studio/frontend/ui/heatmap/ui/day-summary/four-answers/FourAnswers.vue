<script setup lang="ts">
/**
 * The four ratings of one day, and how many of each. An answer given zero times
 * that day is omitted, and a day with none of them draws nothing.
 */
import { computed } from 'vue'

import type { Day } from '../../../lib/heatmap'
import type { Words } from '../../../lib/words'

const props = defineProps<{
  day: Day
  /** The name of each answer, in the user's language. */
  words: Words
}>()

const ratings = computed(() =>
  [
    { label: props.words.again, count: props.day.again, tone: 'again' },
    { label: props.words.hard, count: props.day.hard, tone: 'hard' },
    { label: props.words.good, count: props.day.good, tone: 'good' },
    { label: props.words.easy, count: props.day.easy, tone: 'easy' },
  ].filter((one) => one.count > 0),
)
</script>

<template>
  <ul v-if="ratings.length" class="day-summary__ratings" data-day-detail="ratings">
    <li v-for="one in ratings" :key="one.tone" :data-tone="one.tone">
      <span class="day-summary__label" data-day-detail="label">{{ one.label }}</span>
      <span class="day-summary__rating-count" data-day-detail="rating-count">{{ one.count }}</span>
    </li>
  </ul>
</template>

<style scoped>
.day-summary__ratings {
  display: flex;
  margin: var(--numen-inset) 0 0;
  padding: 0;
  flex-direction: column;
  gap: 0.125rem;
  list-style: none;
}

.day-summary__ratings li {
  display: flex;
  justify-content: space-between;
  gap: var(--numen-inset);
}

/* The word for each answer is drawn in what that answer means. */
.day-summary__ratings li[data-tone='again'] .day-summary__label {
  color: var(--numen-alarm);
}

.day-summary__ratings li[data-tone='hard'] .day-summary__label {
  color: var(--numen-caution-fg);
}

.day-summary__rating-count {
  font-variant-numeric: tabular-nums;
}
</style>
