<script setup lang="ts">
/**
 * What is running behind the window, as cards in its bottom corner.
 *
 * A card appears once its work has lasted, and goes when the work does. Each
 * can be put away, and one put away stays away until that work has ended and
 * begun again.
 *
 * It stands over what it covers and takes no room from it. What the work is and
 * what it is called belong to whoever draws this.
 */
import { computed, ref, watch, watchEffect } from 'vue'
import Activity from '../activity/Activity.vue'
import { arrivals, remembered, showing, standing, tallyOf, WAIT, type Notice } from './model'

const props = withDefaults(
  defineProps<{
    /** What is running, in the order it is drawn. */
    notices?: readonly Notice[]
    /** What the corner is announced as. */
    name?: string
    /** What the way to put one away is called. */
    putAway?: string
    /** How long work runs before it is worth a card. */
    wait?: number
    /** What the moment is. The window's, unless a story hands in its own. */
    clock?: () => number
  }>(),
  {
    notices: () => [],
    name: 'Background work',
    putAway: 'Put away',
    wait: WAIT,
    clock: () => Date.now(),
  },
)

/** The ones a person has put away, and when each of the rest arrived. */
const away = ref<ReadonlySet<string>>(new Set())
const arrived = ref<ReadonlyMap<string, number>>(new Map())
const now = ref(props.clock())

watch(
  () => props.notices,
  (all) => {
    now.value = props.clock()
    arrived.value = arrivals(arrived.value, all, now.value)
    away.value = remembered(away.value, all)
  },
  { immediate: true },
)

const drawn = computed(() =>
  showing(props.notices, arrived.value, away.value, now.value, props.wait),
)

/** Whether work is standing that has not lasted long enough to be drawn. */
const coming = computed(
  () =>
    standing(props.notices).filter((one) => !away.value.has(one.id)).length > drawn.value.length,
)

// Work becomes worth drawing while nothing else changes, so the moment is
// watched for as long as something is waiting on it.
watchEffect((clean) => {
  if (!coming.value) return
  const beat = setInterval(() => {
    now.value = props.clock()
  }, 250)
  clean(() => clearInterval(beat))
})

const put = (id: string) => {
  away.value = new Set([...away.value, id])
}
</script>

<template>
  <div v-if="drawn.length" class="notices numen font-sans text-small" :aria-label="name">
    <article v-for="one in drawn" :key="one.id" class="notice">
      <Activity
        class="notice__work"
        :says="one.says"
        :about="one.about ?? ''"
        :tally="tallyOf(one)"
        :working="one.working ?? false"
        :left="one.left ?? ''"
      />
      <button type="button" class="notice__away" :aria-label="putAway" @click="put(one.id)">
        <svg viewBox="0 0 12 12" aria-hidden="true" focusable="false">
          <path d="M3 3 L9 9 M9 3 L3 9" />
        </svg>
      </button>
    </article>
  </div>
</template>

<style scoped>
/* Over the corner, never in the way of a pointer that is not on a card. */
.notices {
  --gap: 0.5rem;

  position: fixed;
  inset-block-end: var(--numen-inset-wide);
  inset-inline-start: var(--numen-inset-wide);
  z-index: 40;
  display: flex;
  flex-direction: column;
  gap: var(--gap);
  pointer-events: none;
}

/* As wide as it needs, and never wider than a narrow window. */
.notice {
  --room: 24rem;

  pointer-events: auto;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  inline-size: var(--room);
  max-inline-size: calc(100vw - 2 * var(--numen-inset-wide));
  padding: 0.5rem 0.5rem 0.5rem 0.75rem;
  border: var(--numen-stroke) solid var(--numen-node-border);
  border-radius: var(--numen-radius-panel);
  background: var(--numen-node-bg);
  color: var(--numen-node-fg);
  box-shadow: 0 0.5rem 1.5rem oklch(0% 0 0 / 35%);
}

.notice__work {
  min-inline-size: 0;
  flex: 1;
}

.notice__away {
  flex: none;
  display: grid;
  place-items: center;
  inline-size: 1.25rem;
  block-size: 1.25rem;
  border-radius: var(--numen-radius-pill);
  color: inherit;
  opacity: 0.5;
}

.notice__away:hover,
.notice__away:focus-visible {
  opacity: 1;
  background: color-mix(in oklab, currentColor 12%, transparent);
}

.notice__away svg {
  inline-size: 0.75rem;
  block-size: 0.75rem;
  stroke: currentColor;
  stroke-width: 1.5;
  stroke-linecap: round;
  fill: none;
}
</style>
