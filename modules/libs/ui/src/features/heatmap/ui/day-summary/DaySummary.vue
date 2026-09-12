<script setup lang="ts">
/**
 * What one day of the grid came to. The day is handed to it, and where it
 * stands is the tooltip's business.
 *
 * `data-day-summary` names each part of the account: `account` is the whole of it,
 * and `day`, `count`, `four`, `said`, `how-many` and `came` are its lines.
 */
import { computed } from 'vue'

import type { Day } from '../../lib/heatmap'
import type { Words } from '../../lib/words'

const props = defineProps<{
  day: Day
  /** What each line of it is called, in the person's own language. */
  words: Words
}>()

/** The four, and how many of each, left out where none were said that way. */
const four = computed(() =>
  [
    { says: props.words.again, count: props.day.again, tone: 'again' },
    { says: props.words.hard, count: props.day.hard, tone: 'hard' },
    { says: props.words.good, count: props.day.good, tone: 'good' },
    { says: props.words.easy, count: props.day.easy, tone: 'easy' },
  ].filter((one) => one.count > 0),
)

/**
 * How much of what the person is already reviewing came back to them, where any
 * of it was asked at all. A day of nothing but new cards has no share to give,
 * and says none.
 */
const came = computed(() => {
  if (props.day.asked <= 0) return ''
  return `${Math.round((props.day.recalled / props.day.asked) * 100)}%`
})
</script>

<template>
  <div class="day-summary" data-day-summary="account">
    <p class="day-summary__day" data-day-summary="day">{{ words.names(day.day) }}</p>

    <p v-if="day.ahead" class="day-summary__count" data-day-summary="count">
      {{ day.did > 0 ? `${day.did} ${words.toCome}` : words.nothing }}
    </p>
    <template v-else>
      <p class="day-summary__count" data-day-summary="count">
        {{ day.did > 0 ? `${day.did} ${words.answered}` : words.nothing }}
      </p>
      <ul v-if="four.length" class="day-summary__four" data-day-summary="four">
        <li v-for="one in four" :key="one.tone" :data-tone="one.tone">
          <span class="day-summary__said" data-day-summary="said">{{ one.says }}</span>
          <span class="day-summary__how-many" data-day-summary="how-many">{{ one.count }}</span>
        </li>
      </ul>
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

.day-summary__came {
  margin: var(--numen-inset) 0 0;
  color: var(--numen-hushed);
}
</style>
