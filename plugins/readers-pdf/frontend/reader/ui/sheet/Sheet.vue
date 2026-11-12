<script setup lang="ts">
/**
 * One page of a document in the reader row: the page image, the highlights over
 * it, and a spinner while it loads.
 *
 * A new picture is loaded in a hidden image and replaces the one shown only once
 * it has loaded, so a page never goes blank while it is drawn again. A picture
 * that is no longer wanted is dropped from the hidden image, which cancels the
 * request. Highlights are placed in fractions of the page, so zoom scales them.
 * A page that fails to load is requested again a few times, each request
 * carrying a distinct `again` number.
 */
import { computed, onBeforeUnmount, ref, useTemplateRef, watch } from 'vue'
import { Spinner } from '@numen/ui'
import type { Rect } from '../../lib/strip'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    at: number
    picture?: string
    highlights?: readonly Rect[]
    otherHighlights?: readonly Rect[]
    page?: string
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

/** Number of retries so far. */
const tries = ref(0)

/** The URL of the image shown. Nothing until the first image has loaded. */
const shownUrl = ref('')
const isLoaded = computed(() => shownUrl.value !== '')
const incoming = useTemplateRef<HTMLImageElement>('incoming')

/** Whether the retries are exhausted. */
const givenUp = computed(() => tries.value > ATTEMPTS)

/** The image URL for the current attempt. */
const imageUrl = computed(() => {
  if (!props.picture || givenUp.value) return ''
  const separator = props.picture.includes('?') ? '&' : '?'
  return tries.value === 0 ? props.picture : `${props.picture}${separator}again=${tries.value}`
})

/** The URL being loaded: the one wanted, unless it is the one shown. */
const loading = computed(() =>
  imageUrl.value === shownUrl.value ? undefined : imageUrl.value || undefined,
)

/* ---------------------------------- Hooks --------------------------------- */
// Reset retry state when the page image URL changes. What is shown stays until the new image loads.
watch(
  () => props.picture,
  () => {
    tries.value = 0
  },
)

// A request in flight when the page is removed is cancelled.
onBeforeUnmount(() => incoming.value?.removeAttribute('src'))

/* --------------------------------- Helpers -------------------------------- */
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
      v-if="isLoaded"
      class="reader__picture block size-full object-contain"
      :src="shownUrl"
      :alt="`${page} ${at + 1}`"
      draggable="false"
    />
    <img
      ref="incoming"
      class="reader__incoming hidden"
      alt=""
      :src="loading"
      draggable="false"
      @error="tries += 1"
      @load="shownUrl = imageUrl"
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

/* A highlight that is not the navigation target is drawn faintly; the
   navigation target is drawn at full opacity. */
.reader__other-highlight {
  opacity: 0.35;
}

.reader__spinner {
  --spinner-size: 1.4rem;
  --spinner-thickness: 0.125rem;
}
</style>
