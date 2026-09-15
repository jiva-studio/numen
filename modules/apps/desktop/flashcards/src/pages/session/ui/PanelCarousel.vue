<script setup lang="ts">
/**
 * Three panels in a row, one of them in the window at a time. The middle is
 * never drawn narrower; narrow, either side is the width of the window and the
 * middle is scrolled out of it whole.
 */
import { useTemplateRef } from 'vue'

import { useCarouselStrip } from '../model/carousel'
import type { PanelPlace } from '../model/carousel'

/** A hand moves this as much as the owner does, so it is a model and not a prop. */
const shown = defineModel<PanelPlace>('at', { required: true })

const window_ = useTemplateRef<HTMLElement>('window')
const middle = useTemplateRef<HTMLElement>('middle')

const { taking, handleScroll, handlePointerDown, handlePointerMove, letGo } = useCarouselStrip({
  window: window_,
  middle,
  shown,
})
</script>

<template>
  <div
    ref="window"
    class="carousel"
    :class="{ 'carousel--taking': taking }"
    @scroll="handleScroll"
    @pointerdown="handlePointerDown"
    @pointermove="handlePointerMove"
    @pointerup="letGo"
    @pointercancel="letGo"
  >
    <div class="carousel__before" :inert="shown !== 'before' || undefined">
      <slot name="before" />
    </div>
    <div ref="middle" class="carousel__here" :inert="shown !== 'here' || undefined">
      <slot />
    </div>
    <div class="carousel__after" :inert="shown !== 'after' || undefined">
      <slot name="after" />
    </div>
  </div>
</template>

<style scoped>
/* The window the strip is seen through, and the strip is scrolled inside it.
   The bar itself is not drawn: what moves the strip is the hand and the keys.

   Positioned, so that where the middle stands is measured from the strip and
   the stops are the layout's own answer. */
.carousel {
  --carousel-side: min(28rem, 100%);

  position: relative;
  display: flex;
  flex: 1;
  min-inline-size: 0;
  min-block-size: 0;
  /* The three stand apart the way everything else in the window does, and the
     strip is that much longer for it. */
  gap: var(--numen-inset);
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: none;
  overscroll-behavior-x: contain;
  /* Taken there rather than put there, so a key and a hand read as the same
     movement. A hand is already moving it and turns this off.

     Nothing snaps the strip: a snap put back over a strip a hand has already
     moved lands it instantly, and what settles it is the smoothing here. */
  scroll-behavior: smooth;
  /* The strip is taken sideways and a card is read down, so the hand going
     across belongs here and the hand going up and down does not. */
  touch-action: pan-y;
}

.carousel::-webkit-scrollbar {
  display: none;
}

/* A hand on the strip is where the strip is, and nothing pulls it to a stop of
   its own while that hand is down. */
.carousel--taking {
  scroll-behavior: auto;
  cursor: grabbing;
  user-select: none;
}

.carousel__before,
.carousel__here,
.carousel__after {
  display: flex;
  min-inline-size: 0;
}

.carousel__here {
  flex: 0 0 100%;
}

.carousel__before,
.carousel__after {
  flex: 0 0 var(--carousel-side);
}

/* Too narrow for two of them at once: either side takes the window, and the
   middle is scrolled out of it rather than squeezed into what is left. */
@media (max-width: 68rem) {
  .carousel {
    --carousel-side: 100%;
  }
}
</style>
