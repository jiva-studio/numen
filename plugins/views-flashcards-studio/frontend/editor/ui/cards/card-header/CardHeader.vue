<script setup lang="ts">
/**
 * The strip a tile or a block is dragged by.
 *
 * It runs the whole width of what it heads, like the bar across the top of a
 * and the strip itself is what a gesture takes hold of.
 *
 * The strip holds a name box and buttons, so it is a container and not a
 * control, and the keyboard does not stop on it. What the keyboard takes hold
 * of is the handle at its start, as it is on every other row that is reordered
 * here: a control, named, and worked by the arrows along the order.
 */
import { useGripPress } from '../../../model/cards/useGripPress'
import { CardRow } from '../card-row'
import { Icon } from '../icon'
import { parseStepKey, STEP_KEYS, type StepDirection } from '../../../../lib/cards'

defineProps<{
  drag: string
}>()

const emit = defineEmits<{
  /** It was taken hold of by the pointer, and released again. */
  (event: 'dragstart', press: DragEvent): void
  (event: 'dragend', press: DragEvent): void
  /**
   * The card is moved one position along the order, with the key press itself.
   * Whether there is a position in that direction is decided by the caller.
   */
  (event: 'step', direction: StepDirection, press: KeyboardEvent): void
}>()

/** It is taken hold of. A press on something worked lets that thing have it. */
const { isDraggable, press: onPress } = useGripPress()

const onStep = (event: KeyboardEvent): void => {
  const direction = parseStepKey(event.key)
  if (direction !== null) emit('step', direction, event)
}
</script>

<template>
  <CardRow
    as="header"
    tone="header"
    class="card-header"
    data-grip
    :draggable="isDraggable"
    :title="drag"
    @dragstart="emit('dragstart', $event)"
    @dragend="emit('dragend', $event)"
    @pointerdown="onPress"
  >
    <!-- The handle is what the keyboard takes hold of. The pointer has the
         whole strip, so it takes no drag of its own. -->
    <span
      class="card-header__grip text-hushed flex shrink-0 items-center"
      role="button"
      tabindex="0"
      :aria-label="drag"
      :aria-keyshortcuts="STEP_KEYS"
      @keydown="onStep"
    >
      <Icon name="grip" />
    </span>

    <span class="card-header__content min-w-0 flex-1"><slot /></span>
    <span class="card-header__actions flex shrink-0 items-center"><slot name="actions" /></span>
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
.card-header__actions {
  opacity: 0;
  will-change: opacity;
  transition: opacity var(--numen-motion-hover) var(--numen-easing);
}

.card-header:hover .card-header__grip,
.card-header:focus-within .card-header__grip,
.card-header:hover .card-header__actions,
.card-header:focus-within .card-header__actions {
  opacity: 1;
}

@media (prefers-reduced-motion: reduce) {
  .card-header__grip,
  .card-header__actions {
    transition: none;
  }
}
</style>
