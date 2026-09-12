<script setup lang="ts">
// --- Props & Emits ---
import { useTemplateRef } from 'vue'
import { Book } from '@numen/ui'
import { ListTree } from '@lucide/vue'
import type { BookSpan } from '@numen/ui'
import type { BookHandle } from '../types'
import { WORDS as words } from '../words'

const props = defineProps<{
  markup: string
  path: string
  reading: BookSpan
  bookSpan: BookSpan
  offset: number
  highlights: readonly BookSpan[]
  elsewhere: readonly BookSpan[]
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
function onMoved(at: number) {
  emit('moved', at)
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

function pressed(event: KeyboardEvent): boolean {
  return book.value?.pressed(event) ?? false
}

function focusWay() {
  way.value?.focus()
}

defineExpose({
  measure,
  pressed,
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
    :elsewhere="props.elsewhere"
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
