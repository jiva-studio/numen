<script setup lang="ts">
/**
 * One page of a document in the reader row: the page image, the highlights over
 * it, and a spinner while it loads.
 *
 * Nothing is painted until the page image has loaded, and highlights are placed
 * in fractions of the page, so zoom scales them. A page that fails to load is
 * requested again a few times, each request carrying a distinct `again` number.
 */
import { computed, ref, watch } from 'vue'
import { Spinner } from '@numen/ui'
import type { Rect } from '../../lib/strip'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    /** Which page of the document this is, counted from the first. */
    at: number
    /**
     * Image URL of the page. A retry appends an `again` parameter, joined with `&`
     * when the URL already has a query string and with `?` otherwise.
     */
    picture?: string
    /** What is highlighted on it, in fractions of it. */
    highlights?: readonly Rect[]
    /** The other places named on it, apart from the one opened at. */
    otherHighlights?: readonly Rect[]
    /** Accessible name of the page. */
    page?: string
    /** Error message displayed when a page fails to load. */
    undrawn?: string
  }>(),
  {
    picture: '',
    highlights: () => [],
    otherHighlights: () => [],
    page: 'Page',
    undrawn: 'Failed to load page',
  },
)

/* ---------------------------------- State --------------------------------- */
/** Maximum number of retries after a page fails to load. */
const ATTEMPTS = 3

/** Number of retries so far, and whether the image loaded. */
const tries = ref(0)
const isLoaded = ref(false)

/** Whether the retries are exhausted. */
const givenUp = computed(() => tries.value > ATTEMPTS)

/** The image URL for the current attempt. */
const imageUrl = computed(() => {
  if (!props.picture || givenUp.value) return ''
  const separator = props.picture.includes('?') ? '&' : '?'
  return tries.value === 0 ? props.picture : `${props.picture}${separator}again=${tries.value}`
})

/* ---------------------------------- Hooks --------------------------------- */
// Reset retry state when the page image URL changes.
watch(
  () => props.picture,
  () => {
    tries.value = 0
    isLoaded.value = false
  },
)

/* --------------------------------- Helpers -------------------------------- */
/** One rectangle, as a share of the page it is drawn over. */
const getBoxStyle = (one: Rect) => ({
  insetInlineStart: `${one.minX * 100}%`,
  insetBlockStart: `${one.minY * 100}%`,
  inlineSize: `${(one.maxX - one.minX) * 100}%`,
  blockSize: `${(one.maxY - one.minY) * 100}%`,
})
</script>

<template>
  <figure class="reader__page rounded-node absolute top-0 overflow-hidden" :data-page="at">
    <img
      v-if="imageUrl"
      class="reader__picture block size-full object-contain"
      :class="{ invisible: !isLoaded }"
      :src="imageUrl"
      :alt="`${page} ${at + 1}`"
      draggable="false"
      @error="tries += 1"
      @load="isLoaded = true"
    />
    <div v-if="!isLoaded" class="reader__spinner absolute inset-0 grid place-items-center">
      <span v-if="givenUp" class="px-inset text-small text-hushed text-center">
        {{ undrawn }}
      </span>
      <Spinner v-else class="text-hushed" />
    </div>
    <template v-if="isLoaded">
      <div
        v-for="(one, index) in otherHighlights"
        :key="`other-highlight-${index}`"
        class="reader__other-highlight rounded-tight pointer-events-none absolute bg-(--numen-highlight)"
        :style="getBoxStyle(one)"
      />
      <div
        v-for="(one, index) in highlights"
        :key="index"
        class="reader__highlight rounded-tight pointer-events-none absolute bg-(--numen-highlight)"
        :style="getBoxStyle(one)"
      />
    </template>
  </figure>
</template>

<style scoped>
/* The page border separates the page from the surface.
   The edge is the page's, so a highlight is placed inside it. */
.reader__page {
  border: var(--numen-stroke) solid var(--numen-rule);
  background: var(--numen-raised);
}

/* A place the person was not sent to is drawn faintly: it says there is
   something here, and the place they were sent to is the one drawn full. */
.reader__other-highlight {
  opacity: 0.35;
}

.reader__spinner {
  --spinner-size: 1.4rem;
  --spinner-thickness: 2px;
}
</style>
