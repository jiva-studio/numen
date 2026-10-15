<script setup lang="ts">
/**
 * Document tab view rendering reader pages and highlights.
 */
import { Reader, Waiting } from '@numen/ui'
import { WORDS as words } from '../words'
import type { DocumentTabState } from '../model/useDocumentTab'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ state: DocumentTabState }>()

/* -------------------------------- Handlers -------------------------------- */
function onGoToPage(page: number) {
  void props.state.goToPage(page)
}

function onWiden(wide: number) {
  props.state.widen(wide)
}
</script>

<template>
  <Reader
    :ref="(reader: unknown) => props.state.setPageHandle(reader)"
    :pages="props.state.pages.value"
    :at="props.state.pageNumber.value"
    :picture="props.state.getPageImageUrl"
    :highlights="props.state.getHighlightsOn"
    :other-highlights="props.state.getOtherHighlightsOn"
    :words="words"
    @go="onGoToPage"
    @measure="onWiden"
  >
    <!-- Until the layout has been read, no pages is not the same as no pages
         yet. -->
    <template #silence>
      <Waiting v-if="props.state.isLoading.value" :label="words.loading" />
      <div v-else-if="props.state.error.value" class="fallback-panel">
        <p class="fallback-message">{{ props.state.error.value }}</p>
      </div>
      <div v-else class="fallback-panel">
        <p class="fallback-message">PDF reader unavailable or no pages</p>
      </div>
    </template>
  </Reader>
</template>

<style scoped>
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
