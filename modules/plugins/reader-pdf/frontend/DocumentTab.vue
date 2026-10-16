<script setup lang="ts">
/**
 * PDF document viewer tab component.
 * Renders document pages, toolbar controls, and text highlights.
 */
import { computed, onUnmounted, ref } from 'vue'
import { Reader, Waiting } from '@numen/ui'
import { WORDS as words } from './words'
import type { DocumentTabState } from '@/pages/document-viewer/model/useDocumentTab'
import { useDocumentTab } from '@/pages/document-viewer/model/useDocumentTab'
import { useDocumentReader } from '@/pages/document-viewer/model/useDocumentReader'
import { documents } from '@/pages/document-viewer/api/wire'

/* --------------------------------- Props ---------------------------------- */
interface Props {
  readonly state?: DocumentTabState | { readonly path?: string; readonly file?: string; [key: string]: unknown }
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

const internalState = !isDocumentTabState(props.state) && targetPath.value
  ? useDocumentTab(useDocumentReader(documents, targetPath.value))
  : null

const activeState = computed<DocumentTabState | null>(() => {
  if (isDocumentTabState(props.state)) return props.state
  return internalState
})

/* --------------------------------- Hooks ---------------------------------- */
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
</script>

<template>
  <div
    ref="rootElement"
    class="pdf-viewer-tab"
    tabindex="-1"
    @keydown="onPageTurnKey"
  >
    <Reader
      v-if="activeState"
      :ref="(reader: unknown) => activeState?.setPageHandle(reader)"
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
