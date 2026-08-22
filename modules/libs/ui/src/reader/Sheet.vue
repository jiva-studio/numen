<script setup lang="ts">
/**
 * One page of a document, in the row: the picture, what is lit over it, and a
 * ring turning while it is on its way.
 *
 * A page arrives drawn, and nothing of it is painted until it has: a picture
 * still coming is the browser's own broken-picture mark and its words, standing
 * where the page will be.
 *
 * A rectangle is a fraction of the page, so it is placed in per cent and the zoom
 * carries it along, and it too waits: over a page not yet arrived it is a mark on
 * nothing.
 *
 * A document is busy while another page of it is drawing, so a page that did
 * not come is asked for again a few times before it says it is not coming. Each
 * ask carries a number the last one did not, because a picture at an address
 * the browser already refused is not asked for again.
 */
import { computed, ref, watch } from 'vue'
import Waiting from '@/waiting/Waiting.vue'

/** Where something sits on the page, in fractions of it. */
interface Lit {
  readonly minX: number
  readonly minY: number
  readonly maxX: number
  readonly maxY: number
}

const props = withDefaults(
  defineProps<{
    /** Which page of the document this is, counted from the first. */
    at: number
    /** Where it is drawn, as an address to point a picture at. */
    picture?: string
    /** What is lit on it, in fractions of it. */
    lit?: readonly Lit[]
    /** The other places on it, each of them somewhere else to look. */
    also?: readonly Lit[]
    /** What the page is called, for whoever cannot see it. */
    page?: string
    /** What is said where it would not come. */
    undrawn?: string
  }>(),
  {
    picture: '',
    lit: () => [],
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
  return tries.value > 0 ? `${props.picture}&again=${tries.value}` : props.picture
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

/** One lit rectangle, as a share of the page it is drawn over. */
const boxOf = (one: Lit) => ({
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
      <Waiting v-else class="text-hushed" />
    </div>
    <template v-if="arrived">
      <div
        v-for="(one, index) in also"
        :key="`also-${index}`"
        class="reader__also pointer-events-none absolute rounded-[2px] bg-(--numen-highlight)"
        :style="boxOf(one)"
      />
      <div
        v-for="(one, index) in lit"
        :key="index"
        class="reader__lit pointer-events-none absolute rounded-[2px] bg-(--numen-highlight)"
        :style="boxOf(one)"
      />
    </template>
  </figure>
</template>

<style scoped>
/* The page stands on the surface, and its own edge is what tells it from it.
   The edge is the page's, so what is lit is placed inside it. */
.reader__page {
  border: var(--numen-stroke) solid var(--numen-node-border);
  background: var(--numen-node-bg);
}

/* A place the person was not sent to is drawn faintly: it says there is
   something here, and the place they were sent to is the one that reads as lit. */
.reader__also {
  opacity: 0.35;
}

/* The ring stands in the middle of a page's worth of nothing, so it is drawn at
   the size of something being waited for and not of a word. */
.reader__waiting :deep(.waiting) {
  --size: 1.4rem;
  --thickness: 2px;
}
</style>
