<script setup lang="ts">
/**
 * Three panels in a row, one of them in the window at a time.
 *
 * The three are a strip the width of all of them, and which of them is in the
 * window is where that strip is scrolled to. A hand takes it there and lets go
 * where it likes; the strip settles on the nearest of the three.
 *
 * The middle is never drawn narrower, and the strip rests on it. Narrow, either
 * side is the width of the window and the middle is scrolled out of it whole.
 */
import { nextTick, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from 'vue'

/** How long a strip taken somewhere by a key has to get there. */
const TAKES = 600

/** How long a strip left alone has to have stopped before it is settled. */
const SETTLES = 140

/** The three the strip stops at, in the order they stand. */
const PLACES = ['before', 'here', 'after'] as const

/** Which of the three is in the window. */
export type PanelPlace = (typeof PLACES)[number]

/** A hand moves this as much as the owner does, so it is a model and not a prop. */
const shown = defineModel<PanelPlace>('at', { required: true })

const window_ = useTemplateRef<HTMLElement>('window')
const middle = useTemplateRef<HTMLElement>('middle')

/** A hand is on the strip, so nothing takes it anywhere else. */
const taking = ref(false)

/**
 * The strip is being taken somewhere it was asked to go. It passes the others
 * on the way, and that is not a person asking for one of them.
 */
let sending = 0

/** The strip has stopped somewhere of its own, and is waiting to be settled. */
let settling = 0

/** Where a move in flight is going, and nothing while none is. */
let aim: PanelPlace | null = null

/**
 * Where the strip stands when each of the three is in the window. They are read
 * off the layout rather than worked out, so the space the three stand apart by
 * is in them already.
 */
const stops = (): Record<PanelPlace, number> => {
  const at = window_.value
  const card = middle.value
  if (!at || !card) return { before: 0, here: 0, after: 0 }
  return { before: 0, here: card.offsetLeft, after: at.scrollWidth - at.clientWidth }
}

/** Which of the three the strip is closest to standing on. */
const nearest = (left: number): PanelPlace => {
  const all = stops()
  let best: PanelPlace = 'before'
  for (const where of PLACES) {
    if (Math.abs(all[where] - left) < Math.abs(all[best] - left)) best = where
  }
  return best
}

const goes = (where: PanelPlace) => {
  window.clearTimeout(settling)
  const at = window_.value
  if (!at) return
  // A strip already standing where it is being sent moves nothing, and a move
  // that moves nothing never arrives: it would hold the strip deaf to the next
  // hand for as long as a move takes.
  if (Math.abs(at.scrollLeft - stops()[where]) <= 1) return
  window.clearTimeout(sending)
  sending = window.setTimeout(() => {
    sending = 0
    aim = null
  }, TAKES)
  aim = where
  at.scrollLeft = stops()[where]
}

/**
 * The strip put where it belongs rather than taken there. The window opens with
 * the card already in it, and a resize moves the stops under a strip that is
 * standing on one, so neither is something to be seen sliding.
 */
const puts = (where: PanelPlace) => {
  const at = window_.value
  if (!at) return
  window.clearTimeout(settling)
  window.clearTimeout(sending)
  sending = 0
  aim = null
  at.style.scrollBehavior = 'auto'
  at.scrollLeft = stops()[where]
  at.style.scrollBehavior = ''
}

const resized = () => puts(shown.value)

watch(
  () => shown.value,
  (where) => {
    if (!taking.value) goes(where)
  },
)

/**
 * Where a hand has taken the strip. Past the halfway mark between two of the
 * stops it has asked for the nearer one, and the window is told as it crosses.
 */
const scrolled = () => {
  const at = window_.value
  if (!at) return
  // A move in flight is over when it arrives, whatever time it took.
  if (aim && Math.abs(at.scrollLeft - stops()[aim]) <= 1) {
    aim = null
    window.clearTimeout(sending)
    sending = 0
  }
  const where = nearest(at.scrollLeft)
  if (!aim && where !== shown.value) shown.value = where

  // A wheel or a trackpad leaves the strip wherever it ran out, and it is taken
  // the rest of the way once it has stopped. A hand still on it is not done.
  //
  // It is armed while a move is in flight too, and aims where that move was
  // going: a wheel turned during one would otherwise leave the strip standing
  // between two of the three, with nothing left to pull it to either.
  //
  // Where it settles is the window's answer and not the strip's: a panel the
  // window refused to open is a panel the strip must not be left standing on.
  if (taking.value) return
  window.clearTimeout(settling)
  const going = aim
  settling = window.setTimeout(() => goes(going ?? shown.value), SETTLES)
}

/** Where the hand went down, and where the strip was under it. */
let from = 0
let was = 0

const took = (press: PointerEvent) => {
  const at = window_.value
  if (!at || press.button !== 0) return
  taking.value = true
  from = press.clientX
  was = at.scrollLeft
  at.setPointerCapture(press.pointerId)
}

const takes = (press: PointerEvent) => {
  const at = window_.value
  if (!at || !taking.value) return
  at.scrollLeft = was - (press.clientX - from)
}

const letGo = async () => {
  if (!taking.value) return
  const at = window_.value
  const where = at ? nearest(at.scrollLeft) : shown.value
  taking.value = false
  // A hand moves the strip with the smoothing off, and it is taken the rest of
  // the way with it on, so the letting go is waited for.
  await nextTick()
  if (where !== shown.value) shown.value = where
  // Where the hand asked for and where the window went are two things: a panel
  // the window refused to open is one the strip goes back off.
  await nextTick()
  goes(shown.value)
}

onMounted(() => {
  puts(shown.value)
  window.addEventListener('resize', resized)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', resized)
  window.clearTimeout(sending)
  window.clearTimeout(settling)
})
</script>

<template>
  <div
    ref="window"
    class="carousel"
    :class="{ 'carousel--taking': taking }"
    @scroll="scrolled"
    @pointerdown="took"
    @pointermove="takes"
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
