<script setup lang="ts">
/**
 * What one day of the grid came to, told where a person is pointing at it.
 *
 * It holds nothing and decides nothing: the day is handed to it, and where it
 * stands is where the grid says.
 */
import { computed } from 'vue'

import type { Day } from './heatmap'
import type { Words } from './told'

const props = defineProps<{
  day: Day
  /** Where on the page it stands, in pixels from the top left of the window. */
  at: { x: number; y: number }
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
 * How much of what the person had learned came back to them, where anything
 * learned was asked at all. A day of nothing but new cards has no share to
 * give, and says none.
 */
const came = computed(() => {
  if (props.day.asked <= 0) return ''
  return `${Math.round((props.day.recalled / props.day.asked) * 100)}%`
})
</script>

<template>
  <aside class="told" :style="{ insetInlineStart: `${at.x}px`, insetBlockStart: `${at.y}px` }">
    <p class="told__day">{{ words.names(day.day) }}</p>

    <p v-if="day.ahead" class="told__count">
      {{ day.did > 0 ? `${day.did} ${words.toCome}` : words.nothing }}
    </p>
    <template v-else>
      <p class="told__count">
        {{ day.did > 0 ? `${day.did} ${words.answered}` : words.nothing }}
      </p>
      <ul v-if="four.length" class="told__four">
        <li v-for="one in four" :key="one.tone" :data-tone="one.tone">
          <span class="told__said">{{ one.says }}</span>
          <span class="told__how-many">{{ one.count }}</span>
        </li>
      </ul>
      <p v-if="came" class="told__came">{{ came }} {{ words.recalled }}</p>
    </template>
  </aside>
</template>

<style scoped>
/* It stands over whatever it is pointing at, out of the way of the pointer. */
.told {
  position: fixed;
  z-index: 1;
  min-inline-size: 8rem;
  padding: var(--numen-inset);
  border: 1px solid var(--numen-node-border);
  border-radius: var(--numen-radius);
  background: var(--numen-node-bg);
  color: var(--numen-node-fg);
  box-shadow: var(--numen-shadow-raised, 0 2px 8px rgb(0 0 0 / 25%));
  font-size: var(--numen-text-1);
  pointer-events: none;
}

.told__day {
  margin: 0;
  font-weight: 600;
}

.told__count {
  margin: 0;
  color: var(--numen-hushed);
}

.told__four {
  display: flex;
  margin: var(--numen-inset) 0 0;
  padding: 0;
  flex-direction: column;
  gap: 0.125rem;
  list-style: none;
}

.told__four li {
  display: flex;
  justify-content: space-between;
  gap: var(--numen-inset);
}

/* The word for each answer is drawn in what that answer means: a card that did
   not come back is not the same news as one that came back easily. */
.told__four li[data-tone='again'] .told__said {
  color: var(--numen-alarm-fg);
}

.told__four li[data-tone='hard'] .told__said {
  color: var(--numen-caution-fg);
}

.told__how-many {
  font-variant-numeric: tabular-nums;
}

.told__came {
  margin: var(--numen-inset) 0 0;
  color: var(--numen-hushed);
}
</style>
