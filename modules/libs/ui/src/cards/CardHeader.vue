<script setup lang="ts">
/**
 * The strip a tile or a block is dragged by.
 *
 * It runs the whole width of what it heads, like the bar across the top of a
 * window: what it holds stands at its start, what it is pressed for at its end,
 * and the strip itself is what a gesture takes hold of.
 *
 * The strip holds a name box and buttons, so it is a container and not a
 * control, and the keyboard does not stop on it. What the keyboard takes hold
 * of is the handle at its start, as it is on every other row that is reordered
 * here: a control, named, and worked by the arrows along the order.
 */
import { onScopeDispose, shallowRef } from 'vue'
import CardRow from './CardRow.vue'
import Icon from './Icon.vue'
import { directionOf, STEP_KEYS, type StepDirection } from './order'

defineProps<{
  /** What is said of taking hold of it. */
  drag: string
}>()

const emit = defineEmits<{
  /** It was taken hold of by the pointer, and let go again. */
  (event: 'dragstart', press: DragEvent): void
  (event: 'dragend', press: DragEvent): void
  /**
   * It was asked to go one place along the order, with the press itself.
   * Whether there is a place that way is the caller's.
   */
  (event: 'step', direction: StepDirection, press: KeyboardEvent): void
}>()

/** What a press lands on that is worked, and lets the press have it. */
const WORKED = 'input, textarea, button, a, select, [contenteditable]'

/** It is taken hold of. A press on something worked lets that thing have it. */
const held = shallowRef(true)

/**
 * The press is over, wherever it was let go. A press that began on something
 * worked may travel outside the strip and end there, so the end of it is heard
 * where every gesture ends.
 */
const release = (): void => {
  held.value = true
  window.removeEventListener('pointerup', release)
  window.removeEventListener('pointercancel', release)
}

const press = (event: PointerEvent): void => {
  if ((event.target as HTMLElement | null)?.closest?.(WORKED) == null) return
  held.value = false
  window.addEventListener('pointerup', release)
  window.addEventListener('pointercancel', release)
}

onScopeDispose(release)

/** The handle asked by an arrow to carry what it heads one place along. */
const step = (event: KeyboardEvent): void => {
  const direction = directionOf(event.key)
  if (direction !== null) emit('step', direction, event)
}
</script>

<template>
  <CardRow
    as="header"
    tone="header"
    class="card-header"
    data-grip
    :draggable="held"
    :title="drag"
    @dragstart="emit('dragstart', $event)"
    @dragend="emit('dragend', $event)"
    @pointerdown="press"
  >
    <!-- The handle is what the keyboard takes hold of. The pointer has the
         whole strip, so it takes no drag of its own. -->
    <span
      class="card-header__grip flex shrink-0 items-center text-hushed"
      role="button"
      tabindex="0"
      :aria-label="drag"
      :aria-keyshortcuts="STEP_KEYS"
      @keydown="step"
    >
      <Icon shows="grip" />
    </span>

    <span class="card-header__held min-w-0 flex-1"><slot /></span>
    <span class="card-header__deeds flex shrink-0 items-center"><slot name="deeds" /></span>
  </CardRow>
</template>

<style scoped>
/* The strip is a row like any other, and reads as something to take hold of.
   What it holds may stand against the strip's own width. */
.card-header {
  position: relative;
  cursor: grab;
  user-select: none;
  -webkit-user-select: none;
}

.card-header:active {
  cursor: grabbing;
}

/* The handle is drawn quietly for as long as nothing is reaching for the
   strip, as the handle on every other row that is reordered here is. */
.card-header__grip {
  cursor: grab;
  opacity: 0.45;
  transition: opacity var(--numen-motion-hover) var(--numen-easing);
}

/* What the strip is pressed for is not drawn until the strip is reached for,
   by the pointer or by the keyboard. It is drawn on a plane of its own, kept
   for as long as the card stands. */
.card-header__deeds {
  opacity: 0;
  will-change: opacity;
  transition: opacity var(--numen-motion-hover) var(--numen-easing);
}

.card-header:hover .card-header__grip,
.card-header:focus-within .card-header__grip,
.card-header:hover .card-header__deeds,
.card-header:focus-within .card-header__deeds {
  opacity: 1;
}

@media (prefers-reduced-motion: reduce) {
  .card-header__grip,
  .card-header__deeds {
    transition: none;
  }
}
</style>
