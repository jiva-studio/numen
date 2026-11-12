<script setup lang="ts">
/**
 * How many cards are due, as one figure and the word for what it counts.
 *
 * A figure that is not yet counted is rendered as a skeleton with the same
 * dimensions as the final figure.
 */
import { computed } from 'vue'
import { Skeleton } from '@numen/ui'
import { useDueText } from '../../model/useDueText'
import { DUE_WORDS, type DueWords } from './due'

const props = withDefaults(
  defineProps<{
    due: number | null
    isBare?: boolean
    isOnButton?: boolean
    words?: DueWords
  }>(),
  { isBare: false, isOnButton: false, words: () => DUE_WORDS },
)

const text = useDueText(
  () => props.due,
  () => props.words.formatDue,
)

/** What it is read out as, where that is not what it draws. */
const label = computed(() => {
  if (props.due === null) return props.words.counting
  return props.isBare ? (text.value ?? undefined) : undefined
})
</script>

<template>
  <!-- A generic element carries no name, so the pill takes a role and is read
       out while the figure is still coming. -->
  <span
    class="due-count"
    role="status"
    :class="{ 'due-count--over': isOnButton }"
    :aria-label="label"
  >
    <!-- Narrower than the pill's own least width, so the box is the same width
         whether the figure has landed or not. -->
    <Skeleton v-if="due === null" wide="0.8rem" high="0.7em" is-pill />
    <template v-else-if="isBare">{{ due }}</template>
    <template v-else>{{ text }}</template>
  </span>
</template>

<style scoped>
.due-count {
  --due-count-min-inline-size: 1.5rem;
  --due-count-padding-block: 0.0625rem;
  --due-count-padding-inline: 0.4rem;
  flex: none;
  min-inline-size: var(--due-count-min-inline-size);
  padding: var(--due-count-padding-block) var(--due-count-padding-inline);
  text-align: center;
  border-radius: var(--numen-radius-pill);
  background: var(--numen-highlight);
  color: var(--numen-caution-fg);
  font-size: var(--numen-text-1);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.due-count--over {
  background: color-mix(in srgb, currentcolor 18%, transparent);
  color: inherit;
}
</style>
