<script setup lang="ts">
/**
 * One card, as it stands in front of a person: the front, and the back once
 * they have said they are ready for it.
 *
 * A face is HTML and is drawn as the HTML it is, measured first against what a
 * card may be drawn with: a deck may have come from another person, and what
 * they wrote is not this window's to run, style or navigate with.
 */
import { computed, ref } from 'vue'
import { safe } from '@numen/ui'

import { swiped } from './swiping'

const props = defineProps<{
  front: string
  back: string
  /** Whether the answer is being shown. */
  shown: boolean
  /** Whether the panel the card is asked about in is up. */
  asking?: boolean
}>()

const emit = defineEmits<{
  (event: 'show'): void
  (event: 'ask'): void
  (event: 'shut'): void
  /** How far the hand has taken the card, while it is still on it. */
  (event: 'dragging', moved: number): void
}>()

const front = computed(() => safe(props.front))
const back = computed(() => safe(props.back))

/** Where the hand went down, and nothing while none is on the card. */
const from = ref<number | null>(null)
const moved = ref(0)

/** The card follows the hand while the hand is on it. */
const taken = computed(() => (from.value === null ? '' : `translateX(${moved.value}px)`))

/**
 * A card is dragged once its answer is showing. One that can be asked about
 * before it is turned is a way not to recall it.
 */
const draggable = () => props.shown

const took = (press: PointerEvent) => {
  if (!draggable()) return
  if ((press.target as HTMLElement | null)?.closest?.('a')) return
  from.value = press.clientX
  moved.value = 0
  ;(press.currentTarget as HTMLElement).setPointerCapture(press.pointerId)
}

const takes = (press: PointerEvent) => {
  if (from.value === null) return
  moved.value = press.clientX - from.value
  emit('dragging', moved.value)
}

const letGo = () => {
  if (from.value === null) return
  const asked = swiped({ moved: moved.value, open: props.asking === true })
  from.value = null
  moved.value = 0
  emit('dragging', 0)
  if (asked?.does === 'open') emit('ask')
  if (asked?.does === 'shut') emit('shut')
}

/**
 * The card is turned over by pressing it. A link inside one is not followed:
 * this window has one page and no way back to it, and where a card points is
 * read in the editor.
 */
const pressed = (press: MouseEvent) => {
  if ((press.target as HTMLElement | null)?.closest?.('a')) {
    press.preventDefault()
    return
  }
  if (!props.shown) emit('show')
}
</script>

<template>
  <article
    class="card"
    :class="{ 'card--taken': from !== null }"
    :style="{ transform: taken }"
    @click="pressed"
    @pointerdown="took"
    @pointermove="takes"
    @pointerup="letGo"
    @pointercancel="letGo"
  >
    <!-- eslint-disable-next-line vue/no-v-html -- measured against what a card may be drawn with -->
    <div class="card__side" v-html="front" />
    <div v-if="shown" class="card__rule" />
    <!-- eslint-disable-next-line vue/no-v-html -- measured against what a card may be drawn with -->
    <div v-if="shown" class="card__side" v-html="back" />
  </article>
</template>

<style scoped>
.card {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-block-size: 0;
  padding: var(--numen-inset-wide);
  overflow-y: auto;
  border: 1px solid var(--numen-node-border);
  border-radius: var(--numen-radius);
  background: var(--numen-node-bg);
  /* A card is read, not scanned, so it is set at the size reading is set at. */
  font-size: var(--numen-reading-size);
  gap: var(--numen-inset-wide);
  /* The card is taken across and the page is scrolled down, so the hand going
     sideways belongs here and the hand going up and down does not. */
  touch-action: pan-y;
  transition: transform var(--numen-motion-hover) var(--numen-easing);
}

/* While the hand is on it the card is where the hand put it, and nothing eases
   it anywhere else. */
.card--taken {
  transition: none;
  cursor: grabbing;
  user-select: none;
}

/* What a person reads off a card is theirs to carry out of the window, so the
   text takes a selection back. */
.card__side {
  user-select: text;
  -webkit-user-select: text;
}

.card__rule {
  flex: none;
  block-size: 1px;
  background: var(--numen-edge);
}
</style>
