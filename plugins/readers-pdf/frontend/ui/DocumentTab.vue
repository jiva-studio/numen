<script setup lang="ts">
/**
 * PDF document viewer tab component.
 * Renders document pages, toolbar controls, and text highlights.
 */
import { computed, onMounted, onUnmounted, useTemplateRef } from 'vue'
import { LoadingIndicator } from '@numen/ui'
import { Reader } from '../reader'
import { WORDS as words } from '../words'
import type { ReaderTabState } from '@numen/plugin-sdk'
import type { TextRange } from '../types'
import type { DocumentTabState } from '../model/useDocumentTab'
import { useDocumentTab } from '../model/useDocumentTab'
import { useDocumentReader } from '../model/useDocumentReader'
import { useReaderTabSync } from '../model/useReaderTabSync'
import { readerPreferences } from '../model/preferences'

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
    'document' in val &&
    'setPageHandle' in val
  )
}

const targetPath = computed(() => {
  const s = props.state as { path?: string; file?: string } | undefined
  return props.path || props.file || s?.path || s?.file || ''
})

const internalState =
  !isDocumentTabState(props.state) && targetPath.value
    ? useDocumentTab(useDocumentReader(targetPath.value))
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
    s.focusSpans = (...spans: readonly TextRange[]) => activeState.value?.document.actions.focusSpans(...spans)
  }
})

/** The tab state from the window, when it is a reader tab. */
const readerTab = computed(() => props.state as Partial<ReaderTabState> | undefined)

useReaderTabSync(
  () => readerTab.value,
  () => activeState.value,
)

onUnmounted(() => {
  if (internalState) internalState.document.close()
})

/* -------------------------------- Handlers -------------------------------- */
function onGoToPage(page: number) {
  if (activeState.value) activeState.value.document.actions.goToPage(page).catch(activeState.value.document.reportFailure)
}

function onWiden(width: number) {
  activeState.value?.document.view.widen(width)
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
      :pages="activeState.document.state.pages.value"
      :at="activeState.document.state.pageNumber.value"
      :picture="activeState.document.view.getPageImageUrl"
      :highlights="activeState.document.highlights.getHighlightsOn"
      :other-highlights="activeState.document.highlights.getOtherHighlightsOn"
      :words="words"
      :fit="readerPreferences.defaultZoom"
      @go="onGoToPage"
      @measure="onWiden"
    >
      <template #silence>
        <LoadingIndicator v-if="activeState.document.state.isLoading.value" :label="words.loading" />
        <p v-else-if="activeState.document.state.error.value" class="fallback-panel fallback-message">
          {{ activeState.document.state.error.value }}
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
  --panel-pad: 2rem;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: var(--panel-pad);
}

.fallback-message {
  --message-text: 0.875rem;
  font-size: var(--message-text);
  color: var(--text-secondary);
}
</style>
