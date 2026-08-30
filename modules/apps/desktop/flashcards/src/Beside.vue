<script setup lang="ts">
/**
 * One thing, and a second beside it that is scrolled to.
 *
 * The pair is a strip the width of both, and which of them is in the window is
 * where that strip is scrolled to. A hand takes it there and lets go where it
 * likes; the strip settles on one or the other.
 *
 * The first is never drawn narrower. Narrow, the second is the width of the
 * window and the first is scrolled out of it whole.
 */
import { nextTick, onBeforeUnmount, ref, useTemplateRef, watch } from 'vue'

/** How long a strip taken somewhere by a key has to get there. */
const TAKES = 600

/** How long a strip left alone has to have stopped before it is settled. */
const SETTLES = 140

const props = defineProps<{ open: boolean }>()

const emit = defineEmits<{ (event: 'update:open', open: boolean): void }>()

const window_ = useTemplateRef<HTMLElement>('window')

/** A hand is on the strip, so nothing takes it anywhere else. */
const taking = ref(false)

/**
 * The strip is being taken somewhere it was asked to go. It passes the halfway
 * mark on the way, and that is not a person asking for the other one.
 */
let sending = 0

/** The strip has stopped somewhere of its own, and is waiting to be settled. */
let settling = 0

/** How far the strip can go, which is the width of the second. */
const most = () => {
  const at = window_.value
  return at ? at.scrollWidth - at.clientWidth : 0
}

const goes = (open: boolean) => {
  window.clearTimeout(settling)
  window.clearTimeout(sending)
  sending = window.setTimeout(() => {
    sending = 0
  }, TAKES)
  const at = window_.value
  if (at) at.scrollLeft = open ? most() : 0
}

watch(
  () => props.open,
  (open) => {
    if (!taking.value) goes(open)
  },
)

/**
 * Where a hand has taken the strip. Past the halfway mark it has asked for the
 * second, and the window is told the moment it crosses.
 */
const scrolled = () => {
  const at = window_.value
  if (!at || sending) return
  const open = at.scrollLeft > most() / 2
  if (open !== props.open) emit('update:open', open)

  // A wheel or a trackpad leaves the strip wherever it ran out, and it is taken
  // the rest of the way once it has stopped. A hand still on it is not done.
  if (taking.value) return
  window.clearTimeout(settling)
  settling = window.setTimeout(() => goes(open), SETTLES)
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
  const open = at ? at.scrollLeft > most() / 2 : false
  taking.value = false
  // A hand moves the strip with the smoothing off, and it is taken the rest of
  // the way with it on, so the letting go is waited for.
  await nextTick()
  goes(open)
}

onBeforeUnmount(() => {
  window.clearTimeout(sending)
  window.clearTimeout(settling)
})
</script>

<template>
  <div
    ref="window"
    class="beside"
    :class="{ 'beside--taking': taking }"
    @scroll="scrolled"
    @pointerdown="took"
    @pointermove="takes"
    @pointerup="letGo"
    @pointercancel="letGo"
  >
    <div class="beside__one" :inert="open || undefined">
      <slot />
    </div>
    <div class="beside__other" :inert="!open || undefined">
      <slot name="other" />
    </div>
  </div>
</template>

<style scoped>
/* The window the strip is seen through, and the strip is scrolled inside it.
   The bar itself is not drawn: what moves the strip is the hand and the keys. */
.beside {
  --beside-other: min(28rem, 100%);

  display: flex;
  flex: 1;
  min-inline-size: 0;
  min-block-size: 0;
  /* The two stand apart the way everything else in the window does, and the
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

.beside::-webkit-scrollbar {
  display: none;
}

/* A hand on the strip is where the strip is, and nothing pulls it to a stop of
   its own while that hand is down. */
.beside--taking {
  scroll-behavior: auto;
  cursor: grabbing;
  user-select: none;
}

.beside__one,
.beside__other {
  display: flex;
  min-inline-size: 0;
}

.beside__one {
  flex: 0 0 100%;
}

.beside__other {
  flex: 0 0 var(--beside-other);
}

/* Too narrow for the two of them: the second takes the window, and the first is
   scrolled out of it rather than squeezed into what is left. */
@media (max-width: 68rem) {
  .beside {
    --beside-other: 100%;
  }
}
</style>
