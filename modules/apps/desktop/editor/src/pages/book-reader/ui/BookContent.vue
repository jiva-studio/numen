<script setup lang="ts">
// --- Props & Emits ---
import { useTemplateRef } from 'vue'
import { Book } from '@numen/ui'
import { ListTree } from '@lucide/vue'
import type { Span } from '@/shared/span'
import type { BookHandle } from '../types'
import { WORDS as words } from '../words'

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
  moved: [offset: number]
  followed: [path: string]
  toggleListing: []
}>()

// --- State ---
const book = useTemplateRef<BookHandle>('book')
const way = useTemplateRef<HTMLElement>('way')

// --- Handlers ---
function onMoved(offset: number) {
  emit('moved', offset)
}

function onFollowed(targetPath: string) {
  emit('followed', targetPath)
}

function onToggleListing() {
  emit('toggleListing')
}

// --- Helpers ---
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
    :otherHighlights="props.otherHighlights"
    :chapter="props.chapter"
    :words="words"
    @moved="onMoved"
    @followed="onFollowed"
  >
    <template #way>
      <button
        ref="way"
        type="button"
        class="book-tab__list"
        :aria-label="props.isListingOpen ? words.hides : words.shows"
        :aria-pressed="props.isListingOpen"
        :aria-expanded="props.isListingOpen"
        @click="onToggleListing"
      >
        <ListTree class="size-4" aria-hidden="true" />
      </button>
    </template>
  </Book>
</template>
