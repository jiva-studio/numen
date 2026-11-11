<script setup lang="ts">
/**
 * PDF document viewer tab component.
 * Renders document pages, toolbar controls, and text highlights.
 */
import { useTemplateRef, type ComponentPublicInstance } from 'vue'
import { LoadingIndicator } from '@numen/ui'
import { Reader } from '../reader'
import { WORDS as words } from '../words'
import type { ReaderTabState } from '@numen/plugin-sdk'
import type { Documents } from '../types'
import { useOpenDocument } from '../model/useOpenDocument'
import { useReaderAttachment } from '../model/useReaderAttachment'
import { useReaderTabSync } from '../model/useReaderTabSync'
import { readerPreferences } from '../model/preferences'
import { isPageHandle } from '../model/useDocumentTab'

/* --------------------------------- Props ---------------------------------- */
interface Props {
  readonly state: ReaderTabState
  readonly documents?: Documents
}

const props = defineProps<Props>()

/* --------------------------------- State ---------------------------------- */
const rootElement = useTemplateRef<HTMLElement>('rootElement')

const activeState = useOpenDocument(props.state.path, props.documents)

useReaderTabSync(
  () => props.state,
  () => activeState,
)

/* --------------------------------- Hooks ---------------------------------- */
useReaderAttachment(
  () => props.state,
  {
    focus: () => rootElement.value?.focus(),
    measure: () => activeState.measure(),
  },
  () => activeState.document.close(),
)

/* -------------------------------- Handlers -------------------------------- */
function onGoToPage(page: number) {
  activeState.document.actions.goToPage(page).catch(activeState.document.reportFailure)
}

function onWiden(width: number) {
  activeState.document.view.widen(width)
}

function onPageTurnKey(event: KeyboardEvent) {
  if (activeState.handleKeyPress(event)) event.preventDefault()
}

function onSetPageHandle(reader: Element | ComponentPublicInstance | null) {
  activeState.setPageHandle(isPageHandle(reader) ? reader : null)
}
</script>

<template>
  <div ref="rootElement" class="pdf-viewer-tab" tabindex="-1" @keydown="onPageTurnKey">
    <Reader
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
        <LoadingIndicator
          v-if="activeState.document.state.isLoading.value"
          :label="words.loading"
        />
        <p
          v-else-if="activeState.document.state.error.value"
          class="fallback-panel fallback-message"
        >
          {{ activeState.document.state.error.value }}
        </p>
        <p v-else class="fallback-panel fallback-message">PDF reader unavailable or no pages</p>
      </template>
    </Reader>
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
