<script setup lang="ts">
import { useTemplateRef } from 'vue'
import { Book } from '@numen/ui'
import { ListTree } from '@lucide/vue'
import type { Span } from '@/shared/span'
import type { BookHandle } from '../types'
import { WORDS as words } from '../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  markup: string
  path: string
  reading: Span
  bookSpan: Span
  offset: number
  highlights: readonly Span[]
  otherHighlights: readonly Span[]
  chapter: string
  isListingOpen: boolean
}>()

const emit = defineEmits<{
  move: [offset: number]
  follow: [path: string]
  'toggle-listing': []
}>()

/* --------------------------------- State ---------------------------------- */
const book = useTemplateRef<BookHandle>('book')
const way = useTemplateRef<HTMLElement>('way')

/* -------------------------------- Handlers -------------------------------- */
function onMove(offset: number) {
  emit('move', offset)
}

function onFollow(targetPath: string) {
  emit('follow', targetPath)
}

function onToggleListing() {
  emit('toggle-listing')
}

/* -------------------------------- Helpers --------------------------------- */
function measure() {
  book.value?.measure()
}

function handleKeyPress(event: KeyboardEvent): boolean {
  return book.value?.handleKey(event) ?? false
}

function focusWay() {
  way.value?.focus()
}

defineExpose({
  measure,
  handleKey: handleKeyPress,
  focusWay,
})
</script>

<template>
  <Book
    ref="book"
    :markup="props.markup"
    :path="props.path"
    :span="props.reading"
    :book="props.bookSpan"
    :at="props.offset"
    :highlights="props.highlights"
    :other-highlights="props.otherHighlights"
    :chapter="props.chapter"
    :words="words"
    @move="onMove"
    @follow="onFollow"
  >
    <template #way>
      <button
        ref="way"
        type="button"
        class="book-tab__list"
        :aria-label="props.isListingOpen ? words.hideContents : words.showContents"
        :aria-pressed="props.isListingOpen"
        :aria-expanded="props.isListingOpen"
        @click="onToggleListing"
      >
        <ListTree class="size-4" aria-hidden="true" />
      </button>
    </template>
  </Book>
</template>
