<script setup lang="ts">
/**
 * Page navigation and zoom controls for a document.
 *
 * The toolbar floats over the page and does not take layout space from it. The
 * current page and the zoom level are state of the reader, kept here.
 */
import { PageControls } from './page-controls'
import { ZoomControls } from './zoom-controls'
import { clampZoom, READER_WORDS, type ReaderWords } from '../../lib/strip'

withDefaults(
  defineProps<{
    /** How many pages the document has. */
    pageCount?: number
    /** The words they are drawn with. */
    words?: ReaderWords
  }>(),
  {
    pageCount: 0,
    words: () => READER_WORDS,
  },
)

/** Which page is in front, counted from the first. */
const page = defineModel<number>('at', { default: 0 })

/** How close the page is drawn, which is never past either end. */
const zoom = defineModel<number>('zoom', { default: 1, set: clampZoom })
</script>

<template>
  <div class="bottom-inset pointer-events-none absolute inset-x-0 flex justify-center">
    <!-- Standing over the page, so it carries a panel's own ground and lets
         what is behind it through. -->
    <div
      class="rounded-pill border-panel-rule bg-panel shadow-panel backdrop-blur-panel pointer-events-auto flex items-center gap-1 border p-1"
    >
      <PageControls v-model:at="page" :page-count="pageCount" :words="words" />
      <ZoomControls v-model:zoom="zoom" :words="words" />
    </div>
  </div>
</template>
