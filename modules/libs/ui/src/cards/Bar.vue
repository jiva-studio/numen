<script setup lang="ts">
/**
 * The strip a tile or a block is carried by.
 *
 * It runs the whole width of what it heads, like the bar across the top of a
 * window: what it holds stands at its start, what it is pressed for at its end,
 * and the strip itself is what a gesture takes hold of.
 */
import { shallowRef } from 'vue'
import Slab from './Slab.vue'
import { wayOf, type Way } from './model'

defineProps<{
  /** What is said of taking hold of it. */
  carry: string
}>()

const emit = defineEmits<{
  /**
   * It was asked to go one place along the order, with the press itself.
   * Whether there is a place that way is the caller's.
   */
  (event: 'step', way: Way, press: KeyboardEvent): void
}>()

/** What a press lands on that is worked rather than carried. */
const WORKED = 'input, textarea, button, a, select, [contenteditable]'

/** It is taken hold of. A press on something worked lets that thing have it. */
const held = shallowRef(true)

const press = (event: PointerEvent): void => {
  held.value = (event.target as HTMLElement | null)?.closest?.(WORKED) == null
}

const release = (): void => {
  held.value = true
}

/**
 * The strip is what a gesture takes hold of, so it is what the keyboard takes
 * hold of too: an arrow along the order carries it one place. A key struck in
 * something the strip holds belongs to that thing.
 */
const carried = (event: KeyboardEvent): void => {
  if (event.target !== event.currentTarget) return
  const way = wayOf(event.key)
  if (way !== null) emit('step', way, event)
}
</script>

<template>
  <Slab
    as="header"
    tone="bar"
    class="bar"
    data-grip
    role="group"
    tabindex="0"
    :draggable="held"
    :aria-label="carry"
    :title="carry"
    @pointerdown="press"
    @pointerup="release"
    @dragend="release"
    @keydown="carried"
  >
    <span class="bar__held min-w-0 flex-1"><slot /></span>
    <span class="bar__deeds flex shrink-0 items-center"><slot name="deeds" /></span>
  </Slab>
</template>

<style scoped>
/* The strip is a row like any other, and reads as something to take hold of. */
.bar {
  cursor: grab;
  user-select: none;
  -webkit-user-select: none;
}

.bar:active {
  cursor: grabbing;
}

/* What stands in the strip is worked, not carried. */
.bar__held :deep(input),
.bar__deeds :deep(button) {
  cursor: auto;
}

.bar__deeds :deep(button) {
  cursor: pointer;
}

/* What the strip is pressed for stands quietly until it is reached for. */
.bar__deeds {
  opacity: 0.55;
  transition: opacity var(--numen-motion-hover) var(--numen-easing);
}

.bar:hover .bar__deeds,
.bar:focus-within .bar__deeds {
  opacity: 1;
}

@media (prefers-reduced-motion: reduce) {
  .bar__deeds {
    transition: none;
  }
}
</style>
