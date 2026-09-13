<script setup lang="ts">
/**
 * What a person turns and zooms a document with.
 *
 * It floats over the page, and all the room there is belongs to the page. The
 * page in front and how close it is drawn are the reader's, held here.
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
const at = defineModel<number>('at', { default: 0 })

/** How close the page is drawn, which is never past either end. */
const zoom = defineModel<number>('zoom', { default: 1, set: clampZoom })
</script>

<template>
  <div class="pointer-events-none absolute inset-x-0 bottom-inset flex justify-center">
    <!-- Standing over the page, so it carries a panel's own ground and lets
         what is behind it through. -->
    <div
      class="pointer-events-auto flex items-center gap-1 rounded-pill border border-panel-rule bg-panel p-1 shadow-panel backdrop-blur-panel"
    >
      <PageControls v-model:at="at" :page-count="pageCount" :words="words" />
      <ZoomControls v-model:zoom="zoom" :words="words" />
    </div>
  </div>
</template>
