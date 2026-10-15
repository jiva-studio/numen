<script setup lang="ts">
import { ref, computed } from 'vue'

/* --------------------------------- Props ---------------------------------- */
interface PageInfo {
  index: number
  width: number
  height: number
}

const props = withDefaults(
  defineProps<{
    pages?: PageInfo[]
    currentPage?: number
    pageImageUrl?: (index: number) => string
    isLoading?: boolean
    error?: string
  }>(),
  {
    pages: () => [],
    currentPage: 0,
    pageImageUrl: undefined,
    isLoading: false,
    error: '',
  },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  'page-change': [page: number]
  'zoom-change': [scale: number]
}>()

/* --------------------------------- State ---------------------------------- */
const scale = ref(1.0)
const totalPages = computed(() => props.pages.length)
const currentImageUrl = computed(() => {
  if (!props.pageImageUrl || totalPages.value === 0) return ''
  return props.pageImageUrl(props.currentPage)
})

/* -------------------------------- Handlers -------------------------------- */
function onNextPage() {
  if (props.currentPage < totalPages.value - 1) {
    emit('page-change', props.currentPage + 1)
  }
}

function onPreviousPage() {
  if (props.currentPage > 0) {
    emit('page-change', props.currentPage - 1)
  }
}

function onZoomIn() {
  scale.value = Math.min(scale.value + 0.25, 3.0)
  emit('zoom-change', scale.value)
}

function onZoomOut() {
  scale.value = Math.max(scale.value - 0.25, 0.5)
  emit('zoom-change', scale.value)
}
</script>

<template>
  <div class="pdf-viewer-root">
    <div v-if="isLoading" class="pdf-viewer-status">
      <span>Loading PDF document…</span>
    </div>
    <div v-else-if="error" class="pdf-viewer-status pdf-viewer-error">
      <span>{{ error }}</span>
    </div>
    <div v-else-if="totalPages === 0" class="pdf-viewer-status">
      <span>No pages in document.</span>
    </div>
    <div v-else class="pdf-viewer-content">
      <div class="pdf-toolbar">
        <button type="button" :disabled="currentPage <= 0" @click="onPreviousPage">Previous</button>
        <span class="pdf-page-indicator">Page {{ currentPage + 1 }} of {{ totalPages }}</span>
        <button type="button" :disabled="currentPage >= totalPages - 1" @click="onNextPage">
          Next
        </button>
        <div class="pdf-zoom-controls">
          <button type="button" @click="onZoomOut">-</button>
          <span>{{ Math.round(scale * 100) }}%</span>
          <button type="button" @click="onZoomIn">+</button>
        </div>
      </div>
      <div class="pdf-page-container">
        <img
          v-if="currentImageUrl"
          :src="currentImageUrl"
          :style="{ transform: `scale(${scale})`, transformOrigin: 'top center' }"
          alt="PDF Page"
          class="pdf-page-image"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.pdf-viewer-root {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  overflow: hidden;
}

.pdf-viewer-status {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  font-size: 14px;
  color: var(--text-secondary, #666);
}

.pdf-viewer-error {
  color: var(--text-error, #d32f2f);
}

.pdf-viewer-content {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.pdf-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 16px;
  background: var(--surface-secondary, #f5f5f5);
  border-bottom: 1px solid var(--border-color, #e0e0e0);
}

.pdf-page-indicator {
  font-size: 13px;
}

.pdf-zoom-controls {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

.pdf-page-container {
  flex: 1;
  overflow: auto;
  display: flex;
  justify-content: center;
  padding: 24px;
}

.pdf-page-image {
  max-width: 100%;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  transition: transform 0.15s ease-out;
}
</style>
