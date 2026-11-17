<script setup lang="ts">
/**
 * The headings of a note hung from the foot of its node: the first rows that fit, with a mark at the edge the
 * rest is on, and the mouse wheel scrolling through them a row at a time. What choosing one does is the caller's.
 */
import { computed, ref, useTemplateRef } from 'vue'
import { getFurthestOffset, getListStart, scrollBy, type ListLayout } from '../lib/layout'
import { useElementWidth } from '../model/useElementWidth'
import type { HeadingRow } from '../types'
import HeadingEntry from './HeadingEntry.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ rows: readonly HeadingRow[]; layout: ListLayout }>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** A heading was chosen, with the line of its note. */
  (event: 'choose', line: number): void
}>()

/* --------------------------------- State ---------------------------------- */
const list = useTemplateRef<HTMLElement>('list')
const width = useElementWidth(list)

/** The first row the window shows. */
const first = ref(0)

/** What a wheel moved that came to no whole row, held for the next one. */
let carried = 0

const furthest = computed(() => getFurthestOffset(props.rows.length, props.layout))
const shown = computed(() => props.rows.slice(first.value, first.value + props.layout.visibleCount))
const isAbove = computed(() => first.value > 0)
const isBelow = computed(() => first.value < furthest.value)

const style = computed(() => ({
  left: `${getListStart(props.layout, width.value)}px`,
  height: `${props.layout.height}px`,
  minWidth: `${props.layout.nodeWidth}px`,
  maxWidth: `${props.layout.viewportWidth - 2 * props.layout.margin}px`,
  paddingTop: `${props.layout.pad}px`,
  paddingBottom: `${props.layout.pad}px`,
  '--headings-arrow-inset': `${props.layout.pad / 2}px`,
  visibility: width.value > 0 ? undefined : 'hidden',
}))

/* -------------------------------- Handlers -------------------------------- */
const onWheel = (event: WheelEvent) => {
  const { by, left } = scrollBy(
    props.layout,
    { delta: event.deltaY, mode: event.deltaMode },
    carried,
  )
  const direction = by !== 0 ? by : left
  if (direction === 0) return
  if (direction < 0 ? !isAbove.value : !isBelow.value) {
    carried = 0
    return
  }
  carried = left
  event.preventDefault()
  event.stopPropagation()
  first.value = Math.min(Math.max(first.value + by, 0), furthest.value)
}
</script>

<template>
  <div ref="list" class="headings__list" :style="style" aria-hidden="true" @wheel="onWheel">
    <HeadingEntry
      v-for="row in shown"
      :key="row.line"
      :row="row"
      :height="layout.rowHeight"
      @choose="emit('choose', row.line)"
    />
    <svg v-if="isAbove" class="headings__more headings__more--above" viewBox="0 0 8 5">
      <path d="M0 5 L4 0 L8 5" />
    </svg>
    <svg v-if="isBelow" class="headings__more headings__more--below" viewBox="0 0 8 5">
      <path d="M0 0 L4 5 L8 0" />
    </svg>
  </div>
</template>

<style scoped>
.headings__list {
  --headings-pad-inline: 0.625rem;
  --list-radius: 0.25rem;
  --more-width: 0.5rem;
  --more-height: 0.3125rem;
  position: absolute;
  inset-block-start: 100%;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  inline-size: max-content;
  border: var(--numen-stroke) solid color-mix(in oklab, var(--numen-rule), transparent 55%);
  border-radius: var(--list-radius);
  background: color-mix(in oklab, var(--numen-raised), transparent 6%);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-font-size);
  user-select: none;
  -webkit-user-select: none;
}

.headings__more {
  position: absolute;
  inset-inline-start: calc(50% - var(--more-width) / 2);
  inline-size: var(--more-width);
  block-size: var(--more-height);
  fill: none;
  stroke: var(--numen-edge-label);
  stroke-width: var(--numen-stroke);
  stroke-linecap: round;
  stroke-linejoin: round;
  pointer-events: none;
}

.headings__more--above {
  inset-block-start: calc(var(--headings-arrow-inset) - var(--more-height) / 2);
}

.headings__more--below {
  inset-block-end: calc(var(--headings-arrow-inset) - var(--more-height) / 2);
}
</style>
