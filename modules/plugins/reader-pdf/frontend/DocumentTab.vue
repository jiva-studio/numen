<script setup lang="ts">
/**
 * PDF document viewer tab component.
 * Renders document pages, toolbar controls, and text highlights.
 */
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { Reader, Waiting } from '@numen/ui'
import { WORDS as words } from './words'
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
const rootElement = ref<HTMLElement | null>(null)

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
  }
  const spans = (props.state as { spans?: readonly Span[] } | undefined)?.spans
  if (activeState.value && spans && spans.length > 0) {
    void activeState.value.focusSpans(...spans)
  }
})

watch(
  () => (props.state as { spans?: readonly Span[] } | undefined)?.spans,
  (spans) => {
    if (activeState.value && spans && spans.length > 0) {
      void activeState.value.focusSpans(...spans)
    }
  },
)

onUnmounted(() => {
  if (internalState) internalState.close()
})

/* -------------------------------- Handlers -------------------------------- */
function onGoToPage(page: number) {
  if (activeState.value) void activeState.value.goToPage(page)
}

function onWiden(wide: number) {
  activeState.value?.widen(wide)
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
        <Waiting v-if="activeState.isLoading.value" :label="words.loading" />
        <div v-else-if="activeState.error.value" class="fallback-panel">
          <p class="fallback-message">{{ activeState.error.value }}</p>
        </div>
        <div v-else class="fallback-panel">
          <p class="fallback-message">PDF reader unavailable or no pages</p>
        </div>
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
