<script setup lang="ts">
/**
 * A line of work at the foot of the window.
 *
 * Quiet by design: it says what is running and roughly how far, and it is the
 * only thing on screen that moves while nobody is asking for anything.
 *
 * It is a line, and whoever draws it announces it.
 */
import { computed } from 'vue'
import Spinner from '../waiting/Spinner.vue'
import { activity, percentWord, type Tally, type Tone } from './tally'

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
    /** Whether the work named is happening now. */
    working?: boolean
    /** How much longer, on a clock, from whoever is timing the count. */
    left?: string
    /** How the line reads. An alarm is a line that stopped badly. */
    tone?: Tone
  }>(),
  { says: '', about: '', working: false, left: '', tone: 'plain' },
)

const shown = computed(() =>
  activity({
    says: props.says,
    ...(props.tone === 'alarm' ? { trouble: true } : {}),
    ...(props.working ? { working: true } : {}),
    ...(props.tally ? { tally: props.tally } : {}),
  }),
)

const words = computed(() => props.says)

/**
 * How far the work has got, said once.
 *
 * A share is drawn as a percentage and as nothing else.
 */
const percent = computed(() =>
  shown.value.share === undefined ? '' : percentWord(shown.value.share),
)

/** Read off the count, so it is shown only where there is one. */
const left = computed(() => (shown.value.counts ? props.left : ''))

/**
 * How the words give way.
 *
 * What is happening keeps to one line where anything stands beside or under it.
 * A count keeps what it is about to one line too, and without one a path or a
 * reason is read over two.
 */
const givesSays = computed(() =>
  shown.value.counts || props.about ? 'truncate' : 'line-clamp-3',
)
const givesAbout = computed(() => (shown.value.counts ? 'truncate' : 'line-clamp-2'))

/**
 * A line about work is hushed. A line with a tone is drawn in it, and takes the
 * colour of whatever ground that tone put it on.
 */
const strength = computed(() => (props.tone === 'plain' ? 'text-hushed' : ''))
</script>

<template>
  <p
    v-if="shown.state !== 'quiet'"
    class="activity numen flex items-center gap-2 font-sans text-small"
    :class="strength"
    :data-state="shown.state"
    :data-tone="tone"
  >
    <span class="activity__words flex min-w-0 flex-1 flex-col">
      <span class="activity__says" :class="givesSays">{{ words }}</span>
      <span v-if="about" class="activity__about" :class="givesAbout">{{ about }}</span>
    </span>
    <span v-if="percent || left" class="activity__count flex gap-2 tabular-nums opacity-70">
      <span v-if="percent" class="activity__percent">{{ percent }}</span>
      <span v-if="left" class="activity__left">{{ left }}</span>
    </span>
    <Spinner
      v-if="shown.share === undefined && shown.state === 'working'"
      class="activity__spinner"
    />
  </p>
</template>

<style scoped>
/* A path and a reason carry no spaces to break at, so they break anywhere. */
.activity__words {
  overflow-wrap: anywhere;
}

/* What it is happening to stands under what is happening, and is said more
   quietly. */
.activity__about {
  opacity: 0.75;
}

/* Every state of the line stands the same height. */
.activity {
  min-block-size: calc(var(--numen-line-height) * 1em);
}

/* How far and how long stand across from the words, and the words are what
   gives way. */
.activity__count {
  flex: none;
}

/* The share holds the room for three figures, and stands to the end of it. */
.activity__percent {
  min-inline-size: 3em;
  text-align: end;
}
</style>
