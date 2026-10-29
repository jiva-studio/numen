<script setup lang="ts">
/**
 * The summary of one day of the grid. The day is passed in, and positioning it
 * is the tooltip's responsibility.
 *
 * `data-day-summary` names each part of the summary: `account` is the whole
 * summary, and `day`, `count`, `four`, `said`, `how-many` and `came` are its
 * lines.
 */
import { computed } from 'vue'

import { FourAnswers } from './four-answers'
import type { Day } from '../../lib/heatmap'
import type { Words } from '../../lib/words'

const props = defineProps<{
  day: Day
  /** The label of each line, in the user's language. */
  words: Words
}>()

/**
 * The share of the reviewed cards that were recalled, where any card was presented.
 * A day with only new cards has no share and shows an empty string.
 */
const came = computed(() => {
  if (props.day.asked <= 0) return ''
  return `${Math.round((props.day.recalled / props.day.asked) * 100)}%`
})
</script>

<template>
  <div class="day-summary" data-day-summary="account">
    <p class="day-summary__day" data-day-summary="day">{{ words.names(day.day) }}</p>

    <p v-if="day.isAhead" class="day-summary__count" data-day-summary="count">
      {{ day.did > 0 ? `${day.did} ${words.toCome}` : words.nothing }}
    </p>
    <template v-else>
      <p class="day-summary__count" data-day-summary="count">
        {{ day.did > 0 ? `${day.did} ${words.answered}` : words.nothing }}
      </p>
      <FourAnswers :day="day" :words="words" />
      <p v-if="came" class="day-summary__came" data-day-summary="came">
        {{ came }} {{ words.recalled }}
      </p>
    </template>
  </div>
</template>

<style scoped>
.day-summary {
  min-inline-size: 8rem;
}

.day-summary__day {
  margin: 0;
  font-weight: 600;
}

.day-summary__count {
  margin: 0;
  color: var(--numen-hushed);
}

.day-summary__came {
  margin: var(--numen-inset) 0 0;
  color: var(--numen-hushed);
}
</style>
