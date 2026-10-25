<script setup lang="ts">
/**
 * Book tab view rendering reflowable book text and contents navigation.
 */
import { ref, useTemplateRef, watchEffect } from 'vue'
import { Waiting } from '@numen/ui'
import BookContent from './BookContent.vue'
import { BookListing } from './book-contents'
import { WORDS as words } from './words'
import { books } from './api/wire'
import { useBookReader } from './model/useBookReader'
import { useBookTab, type BookTabState } from './model/useBookTab'

/* --------------------------------- Props ---------------------------------- */
interface Props {
  path?: string
  state?: BookTabState | { path?: string; [key: string]: unknown }
}

const props = defineProps<Props>()

function isBookTabState(s: unknown): s is BookTabState {
  return typeof s === 'object' && s !== null && 'isLoading' in s && 'setBookHandle' in s
}

const targetPath =
  props.path ??
  (props.state && 'path' in props.state && typeof props.state.path === 'string'
    ? props.state.path
    : '')

const isExistingState = isBookTabState(props.state)
const internalState = isExistingState
  ? null
  : useBookTab(useBookReader(books, targetPath, { words }))
const tabState = isExistingState ? props.state! : internalState!

/* --------------------------------- State ---------------------------------- */
const isListingOpen = ref(false)
const bookContent = useTemplateRef<InstanceType<typeof BookContent>>('bookContent')
watchEffect(() => tabState.setBookHandle(bookContent.value ?? null))
watchEffect(() => {
  if (!isExistingState && props.state && typeof props.state === 'object' && tabState.title.value) {
    ;(props.state as Record<string, unknown>).title = tabState.title.value
  }
})

/* -------------------------------- Handlers -------------------------------- */
function onDismissListing() {
  isListingOpen.value = false
}

function onEscapeListing() {
  isListingOpen.value = false
  bookContent.value?.focusWay()
}

function onPageTurnKey(event: KeyboardEvent) {
  if (tabState.handleKeyPress(event)) event.preventDefault()
}

function onSelectEntry(offset: number) {
  isListingOpen.value = false
  void tabState.goToOffset(offset)
}

function onToggleListing() {
  isListingOpen.value = !isListingOpen.value
}

function onMove(offset: number) {
  void tabState.goToOffset(offset)
}

function onFollow(destPath: string) {
  void tabState.followLink(destPath)
}
</script>

<template>
  <div
    :ref="(held: unknown) => tabState.setTabElement(held as HTMLElement | null)"
    class="book-tab"
    tabindex="-1"
    @keydown="onPageTurnKey"
  >
    <!-- Until the book has been read, a blank page and a book with no text in
         it are drawn the same way. -->
    <Waiting v-if="tabState.isLoading.value" :label="words.loading" />

    <div v-else class="book-tab__reading">
      <BookListing
        :is-open="isListingOpen"
        :entries="tabState.contents.value"
        :offset="tabState.offset.value"
        @go="onSelectEntry"
        @dismiss="onDismissListing"
        @escape="onEscapeListing"
      />

      <BookContent
        ref="bookContent"
        :markup="tabState.markup.value"
        :path="tabState.renderedChapter.value"
        :document-span="tabState.documentSpan.value"
        :book-span="tabState.span.value"
        :offset="tabState.offset.value"
        :highlights="tabState.highlights.value"
        :other-highlights="tabState.otherHighlights.value"
        :chapter="tabState.chapter.value"
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
