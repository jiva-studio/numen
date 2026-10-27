<script setup lang="ts">
/**
 * PDF document viewer tab component.
 * Renders document pages, toolbar controls, and text highlights.
 */
import { computed, onMounted, onUnmounted, useTemplateRef, watch } from 'vue'
import { LoadingIndicator } from '@numen/ui'
import { Reader } from './reader'
import { WORDS as words } from './words'
import type { ReaderTabState, Selector } from '@numen/plugin-sdk'
import type { Span } from './types'
import type { DocumentTabState } from './model/useDocumentTab'
import { useDocumentTab } from './model/useDocumentTab'
import { useDocumentReader } from './model/useDocumentReader'
import { documents } from './api/wire'

/* --------------------------------- Props ---------------------------------- */
interface Props {
  readonly state?:
    DocumentTabState | { readonly path?: string; readonly file?: string; [key: string]: unknown }
  readonly path?: string
  readonly file?: string
}

const props = defineProps<Props>()

/* --------------------------------- State ---------------------------------- */
const rootElement = useTemplateRef<HTMLElement>('rootElement')

function isDocumentTabState(val: unknown): val is DocumentTabState {
  return (
    typeof val === 'object' &&
    val !== null &&
    'pages' in val &&
    'goToPage' in val &&
    'setPageHandle' in val
  )
}

const targetPath = computed(() => {
  if (props.path) return props.path
  if (props.file) return props.file
  const s = props.state as { path?: string; file?: string } | undefined
  return s?.path ?? s?.file ?? ''
})

const internalState =
  !isDocumentTabState(props.state) && targetPath.value
    ? useDocumentTab(useDocumentReader(documents, targetPath.value))
    : null

const activeState = computed<DocumentTabState | null>(() => {
  if (isDocumentTabState(props.state)) return props.state
  return internalState
})

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  rootElement.value?.focus()
  if (props.state && typeof props.state === 'object' && !isDocumentTabState(props.state)) {
    const s = props.state as Record<string, unknown>
    s.focusTab = () => rootElement.value?.focus()
    s.handleKeyPress = (e: KeyboardEvent) => activeState.value?.handleKeyPress(e) ?? false
    s.focusSpans = (...spans: readonly Span[]) => activeState.value?.focusSpans(...spans)
  }
})

/** The tab as the window hands it, where it is a reader tab. */
const readerTab = computed(() => props.state as Partial<ReaderTabState> | undefined)

// The selector is a search hit or the place saved before a restart; `page` counts from 1 (RFC 8118).
watch(
  () => readerTab.value?.selector,
  (selector: Selector | null | undefined) => {
    if (activeState.value && typeof selector?.page === 'number') {
      void activeState.value.goToPage(selector.page)
    }
  },
  { immediate: true },
)

watch(
  () => activeState.value?.pageNumber.value,
  (page) => {
    if (typeof page === 'number' && page > 0) readerTab.value?.savePlace?.({ page })
  },
)

onUnmounted(() => {
  if (internalState) internalState.close()
})

/* -------------------------------- Handlers -------------------------------- */
function onGoToPage(page: number) {
  if (activeState.value) void activeState.value.goToPage(page)
}

function onWiden(width: number) {
  activeState.value?.widen(width)
}

function onPageTurnKey(event: KeyboardEvent) {
  if (activeState.value?.handleKeyPress(event)) event.preventDefault()
}

function onSetPageHandle(reader: unknown) {
  activeState.value?.setPageHandle(reader)
}
</script>

<template>
  <div ref="rootElement" class="pdf-viewer-tab" tabindex="-1" @keydown="onPageTurnKey">
    <Reader
      v-if="activeState"
      :ref="onSetPageHandle"
      :pages="activeState.pages.value"
      :at="activeState.pageNumber.value"
      :picture="activeState.getPageImageUrl"
      :highlights="activeState.getHighlightsOn"
      :other-highlights="activeState.getOtherHighlightsOn"
      :words="words"
      @go="onGoToPage"
      @measure="onWiden"
    >
      <template #silence>
        <LoadingIndicator v-if="activeState.isLoading.value" :label="words.loading" />
        <p v-else-if="activeState.error.value" class="fallback-panel fallback-message">
          {{ activeState.error.value }}
        </p>
        <p v-else class="fallback-panel fallback-message">PDF reader unavailable or no pages</p>
      </template>
    </Reader>
    <div v-else class="fallback-panel">
      <p class="fallback-message">PDF reader unavailable or no pages</p>
    </div>
  </div>
</template>

<style scoped>
.pdf-viewer-tab {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  inline-size: 100%;
  min-block-size: 0;
  outline: none;
}

.fallback-panel {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 32px;
}

.fallback-message {
  font-size: 14px;
  color: var(--text-secondary, #666);
}
</style>
