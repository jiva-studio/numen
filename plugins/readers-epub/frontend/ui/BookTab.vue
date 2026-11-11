<script setup lang="ts">
/**
 * Book tab view rendering reflowable book text and contents navigation.
 */
import { ref, useTemplateRef, watchEffect, type ComponentPublicInstance } from 'vue'
import type { ReaderTabState, Unvalidated } from '@numen/plugin-sdk'
import { LoadingIndicator } from '@numen/ui'
import BookContent from './BookContent.vue'
import { BookListing } from './book-contents'
import { WORDS as words } from '../words'
import { useBookReader, type Books } from '../model/useBookReader'
import { useBookTab, type BookTabState } from '../model/useBookTab'
import { useBookPlace } from '../model/useBookPlace'
import { useBookRanges } from '../model/useBookRanges'

/* --------------------------------- Props ---------------------------------- */
interface Props {
  path?: string
  state?: Partial<ReaderTabState>
  books?: Books
}

const props = defineProps<Props>()

const targetPath = props.path ?? props.state?.path ?? ''

const existingState = isBookTabState(props.state) ? props.state : null
const tabState =
  existingState ??
  useBookTab(
    props.books
      ? useBookReader(props.books, targetPath, { words })
      : useBookReader(targetPath, { words }),
  )

/* --------------------------------- State ---------------------------------- */
const isListingOpen = ref(false)
const bookContent = useTemplateRef<InstanceType<typeof BookContent>>('bookContent')

/** The tab state from the window, when it is a reader tab. */
const readerTab = props.state

useBookPlace(
  {
    offset: tabState.book.navigation.offset,
    isLoading: tabState.book.state.isLoading,
    chapter: tabState.book.progress.chapter,
    getCfi: tabState.book.actions.getCfi,
    goToCfi: tabState.book.actions.goToCfi,
  },
  () => readerTab,
)
useBookRanges(
  {
    isLoading: tabState.book.state.isLoading,
    focusSpans: tabState.book.actions.focusSpans,
    reportFailure: tabState.book.actions.reportFailure,
  },
  () => readerTab,
)

/* ---------------------------------- Hooks --------------------------------- */
watchEffect(() => tabState.setBookHandle(bookContent.value ?? null))

/* -------------------------------- Handlers -------------------------------- */
function onSetTabElement(element: Element | ComponentPublicInstance | null) {
  tabState.setTabElement(element instanceof HTMLElement ? element : null)
}

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
  tabState.book.navigation.goToOffset(offset).catch(tabState.book.actions.reportFailure)
}

function onToggleListing() {
  isListingOpen.value = !isListingOpen.value
}

function onMove(offset: number) {
  tabState.book.navigation.moveTo(offset).catch(tabState.book.actions.reportFailure)
}

function onFollow(destPath: string) {
  tabState.book.navigation.followLink(destPath).catch(tabState.book.actions.reportFailure)
}

/* -------------------------------- Helpers --------------------------------- */
function isBookTabState(s: Unvalidated): s is BookTabState {
  return typeof s === 'object' && s !== null && 'book' in s && 'setBookHandle' in s
}
</script>

<template>
  <div :ref="onSetTabElement" class="book-tab" tabindex="-1" @keydown="onPageTurnKey">
    <!-- Until the book has been read, a blank page and a book with no text in
         it are drawn the same way. -->
    <LoadingIndicator v-if="tabState.book.state.isLoading.value" :label="words.loading" />

    <div v-else class="book-tab__reading">
      <BookListing
        :is-open="isListingOpen"
        :entries="tabState.book.state.contents.value"
        :offset="tabState.book.navigation.offset.value"
        @go="onSelectEntry"
        @dismiss="onDismissListing"
        @escape="onEscapeListing"
      />

      <BookContent
        ref="bookContent"
        :markup="tabState.book.bookDocument.markup.value"
        :path="tabState.book.bookDocument.renderedChapter.value"
        :document-span="tabState.book.bookDocument.documentSpan.value"
        :book-span="tabState.book.state.span.value"
        :offset="tabState.book.navigation.offset.value"
        :target="tabState.book.navigation.target.value"
        :highlights="tabState.book.marks.highlights.value"
        :other-highlights="tabState.book.marks.otherHighlights.value"
        :chapter="tabState.book.progress.chapter.value"
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
