<script setup lang="ts">
/**
 * A paper read page by page: the pages around the one being read, the marked passages over them, and the
 * controls that turn pages and zoom.
 */
import { computed, ref } from 'vue'
import { ChevronLeft, ChevronRight, Minus, Plus } from '@lucide/vue'
import { PAPER, getHighlights, getPagePicture, getPdfDoc } from '../lib/pdfPage'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ path: string }>()

/* --------------------------------- State ---------------------------------- */
const AROUND = 2
const ZOOM_STEP = 0.15
const doc = computed(() => getPdfDoc(props.path))
const page = ref(doc.value.firstPage)
const zoom = ref(1)
const shown = computed(() =>
  Array.from({ length: AROUND * 2 + 1 }, (_, index) => page.value - AROUND + index).filter(
    (page) => page >= 0 && page < doc.value.leaves,
  ),
)

/* -------------------------------- Handlers -------------------------------- */
function onGo(target: number) {
  page.value = Math.min(Math.max(target, 0), doc.value.leaves - 1)
}

function onTypePage(event: Event) {
  const typed = Number((event.target as HTMLInputElement).value)
  if (Number.isFinite(typed)) onGo(typed - 1)
}

function onZoom(by: number) {
  zoom.value = Math.min(1.6, Math.max(0.6, zoom.value + by))
}
</script>

<template>
  <div class="pdf">
    <div class="pdf__bar">
      <button type="button" class="pdf__action" aria-label="Previous page" @click="onGo(page - 1)">
        <ChevronLeft class="pdf__icon" />
      </button>
      <button type="button" class="pdf__action" aria-label="Next page" @click="onGo(page + 1)">
        <ChevronRight class="pdf__icon" />
      </button>
      <input class="pdf__page" :value="page + 1" inputmode="numeric" aria-label="Page" @change="onTypePage" />
      <span class="pdf__count">/ {{ doc.leaves }}</span>
      <button type="button" class="pdf__action" aria-label="Zoom out" @click="onZoom(-ZOOM_STEP)">
        <Minus class="pdf__icon" />
      </button>
      <button type="button" class="pdf__action" aria-label="Zoom in" @click="onZoom(ZOOM_STEP)">
        <Plus class="pdf__icon" />
      </button>
    </div>

    <div class="pdf__pages">
      <div
        v-for="one in shown"
        :key="one"
        class="pdf__leaf"
        :style="{ blockSize: `${zoom * 88}%`, aspectRatio: `${PAPER.width} / ${PAPER.height}` }"
      >
        <img class="pdf__picture" :src="getPagePicture(path, one)" alt="" draggable="false" />
        <span
          v-for="(mark, index) in getHighlights(path, one)"
          :key="index"
          class="pdf__mark"
          :style="{
            insetInlineStart: `${mark.minX * 100}%`,
            insetBlockStart: `${mark.minY * 100}%`,
            inlineSize: `${(mark.maxX - mark.minX) * 100}%`,
            blockSize: `${(mark.maxY - mark.minY) * 100}%`,
          }"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.pdf {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
}

.pdf__bar {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
  gap: 0.5rem;
  min-block-size: var(--numen-action-size);
}

.pdf__action {
  display: grid;
  place-items: center;
  inline-size: 1.75rem;
  block-size: 1.75rem;
  padding: 0;
  border: 0;
  border-radius: var(--numen-radius-tight);
  background: none;
  color: inherit;
  cursor: pointer;
}

.pdf__action:hover {
  background: var(--numen-selection-away);
}

.pdf__icon {
  inline-size: 1rem;
  block-size: 1rem;
  stroke-width: 1.875;
}

.pdf__page {
  inline-size: 3.25rem;
  padding: 0.1rem 0.25rem;
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius-pill);
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: center;
}

.pdf__count {
  color: var(--numen-hushed);
}

.pdf__pages {
  display: flex;
  flex: 1;
  align-items: flex-start;
  justify-content: center;
  gap: 1rem;
  min-block-size: 0;
  overflow: hidden;
  padding: 0.75rem 0 0;
}

.pdf__leaf {
  position: relative;
  flex: none;
  max-block-size: 100%;
  background: var(--numen-raised);
  box-shadow: var(--numen-shadow-card);
}

.pdf__picture {
  display: block;
  inline-size: 100%;
  block-size: 100%;
}

.pdf__mark {
  position: absolute;
  background: var(--numen-highlight);
  mix-blend-mode: multiply;
  pointer-events: none;
}
</style>
