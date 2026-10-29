<script setup lang="ts">
/**
 * A line of work at the foot of the window.
 *
 * Shows what is running and roughly how far. It is the only element that changes
 * while the user is idle.
 */
import { computed } from 'vue'
import { Spinner } from '@numen/ui'
import { activity, percentWord, type Tally, type Severity } from '../../lib/tally'

const props = withDefaults(
  defineProps<{
    /** What is happening, in the words it is to be shown by. */
    text?: string
    /** What it is happening to, when that is worth saying. */
    about?: string
    /**
     * Where the work has got to, when that is countable.
     *
     * Absent and undefined are equivalent; both are declared because the parent
     * derives the tally from a count that may not exist yet.
     */
    tally?: Tally | undefined
    /** Whether the work named is happening now. */
    isWorking?: boolean
    /** Remaining time as a clock string, from the count timer. */
    left?: string
    /** How the line reads. An error is a line that stopped badly. */
    severity?: Severity
  }>(),
  { text: '', about: '', isWorking: false, left: '', severity: 'info' },
)

const activityView = computed(() =>
  activity({
    text: props.text,
    ...(props.severity === 'error' ? { hasFailed: true } : {}),
    ...(props.isWorking ? { isWorking: true } : {}),
    ...(props.tally ? { tally: props.tally } : {}),
  }),
)

const words = computed(() => props.text)

/**
 * Progress of the work.
 *
 * A share is drawn as a percentage and as nothing else.
 */
const percent = computed(() =>
  activityView.value.share === undefined ? '' : percentWord(activityView.value.share),
)

/** Read off the count, so it is shown only where there is one. */
const remainingText = computed(() => (activityView.value.hasCounts ? props.left : ''))

/**
 * How the words give way.
 *
 * The text truncates to one line when a count or a detail line is shown, and
 * clamps to three lines otherwise. The detail truncates to one line when a
 * count is shown, and clamps to two lines otherwise.
 */
const givesSays = computed(() =>
  activityView.value.hasCounts || props.about ? 'truncate' : 'line-clamp-3',
)
const givesAbout = computed(() => (activityView.value.hasCounts ? 'truncate' : 'line-clamp-2'))

/**
 * A line about work is muted. A line with a severity uses the colour of that
 * severity.
 */
const strength = computed(() => (props.severity === 'info' ? 'text-hushed' : ''))
</script>

<template>
  <p
    v-if="activityView.state !== 'quiet'"
    class="activity numen text-small flex items-center gap-2 font-sans"
    :class="strength"
    :data-state="activityView.state"
    :data-severity="severity"
  >
    <span class="activity__words flex min-w-0 flex-1 flex-col">
      <span class="activity__says" :class="givesSays">{{ words }}</span>
      <span v-if="about" class="activity__about" :class="givesAbout">{{ about }}</span>
    </span>
    <span
      v-if="percent || remainingText"
      class="activity__count flex gap-2 tabular-nums opacity-70"
    >
      <span v-if="percent" class="activity__percent">{{ percent }}</span>
      <span v-if="remainingText" class="activity__left">{{ remainingText }}</span>
    </span>
    <Spinner
      v-if="activityView.share === undefined && activityView.state === 'working'"
      class="activity__spinner"
    />
  </p>
</template>

<style scoped>
/* A path and a reason carry no spaces to break at, so they break anywhere. */
.activity__words {
  overflow-wrap: anywhere;
}

/* The detail line is placed below the text and is rendered more
   quietly. */
.activity__about {
  opacity: 0.75;
}

/* Every state has the same height. */
.activity {
  min-block-size: calc(var(--numen-line-height) * 1em);
}

/* How far and how long stand across from the words, and the words are what
   gives way. */
.activity__count {
  flex: none;
}

/* The percentage reserves the width of three figures and is end-aligned. */
.activity__percent {
  min-inline-size: 3em;
  text-align: end;
}
</style>
