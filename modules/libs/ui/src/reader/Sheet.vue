<script setup lang="ts">
/**
 * One page of a document, in the row: the picture, what is highlighted over it,
 * and a ring turning while it is on its way.
 *
 * Nothing is painted until the page has arrived, and a highlight is placed in
 * fractions of it, so the zoom carries it along. A page that did not come is
 * asked for again a few times, each ask carrying a number the last one did not.
 */
import { computed, ref, watch } from 'vue'
import Spinner from '@/waiting/Spinner.vue'
import type { Rect } from './strip'

const props = withDefaults(
  defineProps<{
    /** Which page of the document this is, counted from the first. */
    at: number
    /**
     * Where it is drawn, as an address to point a picture at. A page asked for
     * again carries an `again` parameter, joined on with `&` where the address
     * already asks something and with `?` where it does not.
     */
    picture?: string
    /** What is highlighted on it, in fractions of it. */
    highlights?: readonly Rect[]
    /** The other places on it, each of them somewhere else to look. */
    also?: readonly Rect[]
    /** What the page is called, for whoever cannot see it. */
    page?: string
    /** What is said where it would not come. */
    undrawn?: string
  }>(),
  {
    picture: '',
    highlights: () => [],
    also: () => [],
    page: 'Page',
    undrawn: 'This page would not come.',
  },
)

/** How many times a page that did not come is asked for again. */
const ATTEMPTS = 3

/** How many times this page has been asked for again, and whether it arrived. */
const tries = ref(0)
const arrived = ref(false)

/** Whether it has been asked for as many times as it is going to be. */
const givenUp = computed(() => tries.value > ATTEMPTS)

/** The address it is asked for at, this attempt. */
const drawing = computed(() => {
  if (!props.picture || givenUp.value) return ''
  if (tries.value === 0) return props.picture
  return `${props.picture}${props.picture.includes('?') ? '&' : '?'}again=${tries.value}`
})

// Another address is another question, and it is asked afresh. What would not
// come is what would not come at that width.
watch(
  () => props.picture,
  () => {
    tries.value = 0
    arrived.value = false
  },
)

/** One rectangle, as a share of the page it is drawn over. */
const boxOf = (one: Rect) => ({
  insetInlineStart: `${one.minX * 100}%`,
  insetBlockStart: `${one.minY * 100}%`,
  inlineSize: `${(one.maxX - one.minX) * 100}%`,
  blockSize: `${(one.maxY - one.minY) * 100}%`,
})
</script>

<template>
  <figure class="reader__page absolute top-0 overflow-hidden rounded-node" :data-page="at">
    <img
      v-if="drawing"
      class="reader__picture block size-full object-contain"
      :class="{ invisible: !arrived }"
      :src="drawing"
      :alt="`${page} ${at + 1}`"
      draggable="false"
      @error="tries += 1"
      @load="arrived = true"
    />
    <div v-if="!arrived" class="reader__waiting absolute inset-0 grid place-items-center">
      <span v-if="givenUp" class="px-inset text-center text-small text-hushed">
        {{ undrawn }}
      </span>
      <Spinner v-else class="text-hushed" />
    </div>
    <template v-if="arrived">
      <div
        v-for="(one, index) in also"
        :key="`also-${index}`"
        class="reader__also pointer-events-none absolute rounded-tight bg-(--numen-highlight)"
        :style="boxOf(one)"
      />
      <div
        v-for="(one, index) in highlights"
        :key="index"
        class="reader__highlight pointer-events-none absolute rounded-tight bg-(--numen-highlight)"
        :style="boxOf(one)"
      />
    </template>
  </figure>
</template>

<style scoped>
/* The page stands on the surface, and its own edge is what tells it from it.
   The edge is the page's, so a highlight is placed inside it. */
.reader__page {
  border: var(--numen-stroke) solid var(--numen-rule);
  background: var(--numen-raised);
}

/* A place the person was not sent to is drawn faintly: it says there is
   something here, and the place they were sent to is the one drawn full. */
.reader__also {
  opacity: 0.35;
}

/* The ring stands in the middle of a page's worth of nothing, so it is drawn at
   the size of something being waited for and not of a word. */
.reader__waiting {
  --waiting-size: 1.4rem;
  --waiting-thickness: 2px;
}
</style>
