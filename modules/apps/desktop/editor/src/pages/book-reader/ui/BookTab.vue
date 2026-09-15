<script setup lang="ts">
/**
 * Book tab view rendering reflowable book text and contents navigation.
 */
import { ref, useTemplateRef, watchEffect } from 'vue'
import { Waiting } from '@numen/ui'
import BookContent from './BookContent.vue'
import { BookListing } from './book-listing'
import { WORDS as words } from '../words'
import type { BookTabState } from '../model/useBookTab'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ state: BookTabState }>()

/* --------------------------------- State ---------------------------------- */
const isListingOpen = ref(false)
const bookContent = useTemplateRef<InstanceType<typeof BookContent>>('bookContent')
watchEffect(() => props.state.setBookHandle(bookContent.value ?? null))

/* -------------------------------- Handlers -------------------------------- */
function onDismissListing() {
  isListingOpen.value = false
}

function onEscapeListing() {
  isListingOpen.value = false
  bookContent.value?.focusWay()
}

function onPageTurnKey(event: KeyboardEvent) {
  if (props.state.handleKeyPress(event)) event.preventDefault()
}

function onSelectEntry(offset: number) {
  isListingOpen.value = false
  void props.state.goToOffset(offset)
}

function onToggleListing() {
  isListingOpen.value = !isListingOpen.value
}

function onMove(offset: number) {
  void props.state.goToOffset(offset)
}

function onFollow(targetPath: string) {
  void props.state.followLink(targetPath)
}
</script>

<template>
  <div
    :ref="(held: unknown) => props.state.setTabElement(held as HTMLElement | null)"
    class="book-tab"
    tabindex="-1"
    @keydown="onPageTurnKey"
  >
    <!-- Until the book has been read, a blank page and a book with no text in
         it are drawn the same way. -->
    <Waiting v-if="props.state.isLoading.value" :label="words.loading" />

    <div v-else class="book-tab__reading">
      <BookListing
        :is-open="isListingOpen"
        :entries="props.state.contents.value"
        :at="props.state.offset.value"
        @go="onSelectEntry"
        @dismiss="onDismissListing"
        @escape="onEscapeListing"
      />

      <BookContent
        ref="bookContent"
        :markup="props.state.markup.value"
        :path="props.state.drawn.value"
        :reading="props.state.reading.value"
        :book-span="props.state.span.value"
        :offset="props.state.offset.value"
        :highlights="props.state.highlights.value"
        :other-highlights="props.state.otherHighlights.value"
        :chapter="props.state.chapter.value"
        :is-listing-open="isListingOpen"
        @move="onMove"
        @follow="onFollow"
        @toggle-listing="onToggleListing"
      />
    </div>
  </div>
</template>

<style scoped>
.book-tab {
  block-size: 100%;
  min-block-size: 0;
}

.book-tab:focus,
.book-tab:focus-visible {
  outline: none;
}

.book-tab__reading {
  position: relative;
  block-size: 100%;
  min-inline-size: 0;
}
</style>
