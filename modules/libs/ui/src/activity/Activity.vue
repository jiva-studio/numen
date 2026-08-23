<script setup lang="ts">
/**
 * A line of work at the foot of the window.
 *
 * Quiet by design: it says what is running and roughly how far, and it is the
 * only thing on screen that moves while nobody is asking for anything.
 */
import { computed } from 'vue'
import Waiting from '../waiting/Waiting.vue'
import { activity, percentWord, tallyWord, type Counting, type Tally } from './model'

const props = withDefaults(
  defineProps<{
    /** What is happening, in the words it is to be shown by. */
    says?: string
    /** What it is happening to, when that is worth saying. */
    about?: string
    /**
     * Where the work has got to, when that is countable.
     *
     * Absent and undefined mean the same thing here, and both are spelled,
     * because whoever renders this derives the tally from a count that may not
     * exist yet.
     */
    tally?: Tally | undefined
    /** What the tally counts. */
    counting?: Counting
    /** Whether the work named is happening now. */
    working?: boolean
    /**
     * How fast the count is moving, in words, from whoever is timing it. Empty
     * until there is enough movement to say: a rate needs a clock and this has
     * none.
     */
    rate?: string
    /** How much longer, in words, from the same place. */
    left?: string
    /** Something is wrong and this is what it says. */
    trouble?: string
  }>(),
  { says: '', about: '', counting: 'things', working: false, rate: '', left: '', trouble: '' },
)

const shown = computed(() =>
  activity({
    says: props.trouble || props.says,
    ...(props.trouble ? { trouble: true } : {}),
    ...(props.working ? { working: true } : {}),
    ...(props.tally ? { tally: props.tally } : {}),
  }),
)

const words = computed(() => props.trouble || props.says)
const count = computed(() =>
  shown.value.counts && props.tally ? tallyWord(props.tally, props.counting) : '',
)
const percent = computed(() =>
  shown.value.share === undefined ? '' : percentWord(shown.value.share),
)
/** Both are only shown beside a count, since both are read off one. */
const rate = computed(() => (shown.value.counts ? props.rate : ''))
const left = computed(() => (shown.value.counts ? props.left : ''))
</script>

<template>
  <p
    v-if="shown.state !== 'quiet'"
    class="activity numen flex items-center gap-2 font-sans text-small text-hushed"
    :data-state="shown.state"
    role="status"
    aria-live="polite"
  >
    <span class="activity__mark" />
    <span class="activity__says min-w-0 truncate">{{ words }}</span>
    <span v-if="about" class="activity__about min-w-0 flex-1 truncate opacity-70">{{ about }}</span>
    <span v-else class="activity__gap flex-1" />
    <span v-if="count" class="activity__count tabular-nums opacity-70">{{ count }}</span>
    <span v-if="percent" class="activity__percent tabular-nums opacity-70">{{ percent }}</span>
    <span v-if="rate" class="activity__rate tabular-nums opacity-70">{{ rate }}</span>
    <span v-if="left" class="activity__left truncate opacity-70">{{ left }}</span>
    <span
      v-if="shown.share !== undefined"
      class="activity__bar"
      :style="{ '--activity-share': shown.share }"
    />
    <Waiting v-else-if="shown.state === 'working'" class="activity__waiting" />
  </p>
</template>

<style scoped>
/* Every state of the line stands the same height. */
.activity {
  min-block-size: calc(var(--numen-line-height) * 1em);
}

.activity__mark {
  flex: none;
  inline-size: 0.4em;
  block-size: 0.4em;
  border-radius: var(--numen-radius-pill);
  background: currentColor;
  opacity: 0.5;
}

.activity[data-state='working'] .activity__mark {
  opacity: 1;
}

/* Resting says something is so, not that it is happening. */
.activity[data-state='resting'] .activity__mark {
  opacity: 0.35;
}

.activity[data-state='trouble'] .activity__mark {
  opacity: 1;
  background: currentColor;
  box-shadow: 0 0 0 0.15em color-mix(in oklab, currentColor 25%, transparent);
}

/* The numbers keep their own line. They are short, and the words beside them
   are what gives way. */
.activity__count,
.activity__percent,
.activity__rate {
  flex: none;
}

.activity__left {
  min-inline-size: 0;
}

.activity__bar {
  flex: none;
  inline-size: 4rem;
  block-size: 0.25em;
  border-radius: var(--numen-radius-pill);
  background: color-mix(in oklab, currentColor 20%, transparent);
  overflow: hidden;
  position: relative;
}

.activity__bar::after {
  content: '';
  position: absolute;
  inset-block: 0;
  inset-inline-start: 0;
  inline-size: calc(var(--activity-share) * 100%);
  background: currentColor;
  opacity: 0.7;
}
</style>
