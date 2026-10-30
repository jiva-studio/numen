<script setup lang="ts">
/**
 * Page navigation and zoom controls for a document.
 *
 * The toolbar is a flat strip at the top of the reader, separated from the pages
 * by a rule. The current page and the zoom level are state of the reader.
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
  <div
    class="border-rule px-gutter min-h-action flex w-full flex-none items-center justify-center gap-2 border-b"
  >
    <PageControls v-model:at="page" :page-count="pageCount" :words="words" />
    <ZoomControls v-model:zoom="zoom" :words="words" />
  </div>
</template>
