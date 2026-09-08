<script setup lang="ts">
/**
 * How many cards are due, as one figure and the word for what it counts.
 *
 * A figure not worked out yet is drawn as the shape it will be, in the same
 * box, so the row it stands in does not move when it lands.
 */
import { computed } from 'vue'
import { Skeleton } from '@/shared/ui/skeleton'
import { DUE_WORDS, type DueWords } from './due'

const props = withDefaults(
  defineProps<{
    /** Cards due today: owed, and never asked. Nothing until it is counted. */
    due: number | null
    /**
     * The number alone. Where a list is long and the room is short, the word is
     * said once above the list and not on every row of it.
     */
    bare?: boolean
    /** Drawn on a ground of its own, where it stands on a filled button. */
    over?: boolean
    /** The words it is drawn with. */
    words?: DueWords
  }>(),
  { bare: false, over: false, words: () => DUE_WORDS },
)

/** What it is read out as, where that is not what it draws. */
const label = computed(() => {
  if (props.due === null) return props.words.counting
  return props.bare ? props.words.counted(props.due) : undefined
})
</script>

<template>
  <!-- A generic element carries no name, so the pill takes a role and is read
       out while the figure is still coming. -->
  <span
    class="due-count"
    role="status"
    :class="{ 'due-count--over': over }"
    :aria-label="label"
  >
    <!-- Narrower than the pill's own least width, so the box is the same width
         whether the figure has landed or not. -->
    <Skeleton v-if="due === null" wide="0.8rem" high="0.7em" pill />
    <template v-else-if="bare">{{ due }}</template>
    <template v-else>{{ words.counted(due) }}</template>
  </span>
</template>

<style scoped>
.due-count {
  flex: none;
  min-inline-size: 1.5rem;
  padding: 0.0625rem 0.4rem;
  text-align: center;
  border-radius: var(--numen-radius-pill);
  background: var(--numen-highlight);
  color: var(--numen-caution-fg);
  font-size: var(--numen-text-1);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

/* On a filled button the ground is the button's own, lightened by the text that
   stands on it, so the pill follows whatever the button is painted. */
.due-count--over {
  background: color-mix(in srgb, currentcolor 18%, transparent);
  color: inherit;
}
</style>
