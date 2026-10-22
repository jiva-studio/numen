<script setup lang="ts">
/**
 * Indented rows, some of which hold other rows.
 *
 * What a row stands for is the caller's: this takes names and hands identities
 * back. What is drawn beside a name comes from a slot, and which rows are open
 * and which are selected are the caller's to hold: the tree works out what a
 * press with a modifier means and says the selection it came to.
 */
import { computed, onBeforeUnmount, onMounted, useTemplateRef } from 'vue'
import { flatten, getMarkOf, type Row, type RowMarker, type RowId } from '../lib/row'
import type { RowDropTarget } from '../lib/drop'
import { useRowDrag } from '../model/drag'
import { useTreeGestures } from '../model/gestures'
import { useDrawnRows } from '../model/rows'
import { useRowSelection } from '../model/selection'
import { useVirtualTree } from '../model/virtual'
import type { Position } from '@/shared/lib/geometry'
import { browserClock, type Clock } from '@/shared/lib/clock'
import { useRefRegistry } from '@/shared/lib/refs'
import { DragPreview } from '@/shared/ui/drag-preview'
import TreeRow from './TreeRow.vue'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    /** What is drawn, nested. */
    rows: readonly Row[]
    /** The rows whose contents are drawn. */
    open?: readonly RowId[]
    /** The rows the selection relies on. */
    selected?: readonly RowId[]
    /** How far the pointer travels before a press becomes a drag. */
    threshold?: number
    /** What the tree is announced as. */
    name?: string
    /** How many rows are being dragged, said at the pointer. */
    counted?: (rows: number) => string
    /**
     * An attribute written onto the rows and onto the tree, for something
     * outside to find them by. Both the name and the values are the caller's,
     * and a row answered with nothing is left unmarked; the tree itself is
     * asked about as no row at all.
     */
    marking?: RowMarker | undefined
    /** The clock. Browser by default; a test hands in its own. */
    clock?: Clock
  }>(),
  {
    open: () => [],
    selected: () => [],
    threshold: 4,
    name: 'Tree',
    counted: (rows: number) => `${rows} rows`,
    marking: undefined,
    clock: () => browserClock,
  },
)

/** The row whose name is in a field. */
const renamingPath = defineModel<RowId | null>('renamingPath', { default: null })

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'open', row: RowId): void
  (event: 'close', row: RowId): void
  /** The rows the selection now relies on. */
  (event: 'select', rows: readonly RowId[]): void
  /** A row acted on: a double press, or Enter. A row that holds turns as well. */
  (event: 'activate', row: RowId): void
  /** A name typed and committed. */
  (event: 'rename', row: RowId, name: string): void
  /** The rows let go somewhere, all of them drop target in the one place. */
  (event: 'move', rows: readonly RowId[], target: RowDropTarget): void
  /**
   * The rows lifted clear of the tree, on their way across whatever is drawn
   * beside it. Where they end up there is not the tree's to say.
   */
  (event: 'drag', rows: readonly RowId[]): void
  /** The rows release, wherever the pointer had got to. */
  (event: 'drop'): void
  /** The selection asked to go. */
  (event: 'remove', rows: readonly RowId[]): void
  /** A menu asked for, and where the pointer was. Nothing for a press off every row. */
  (event: 'menu', row: RowId | null, pos: Position): void
}>()

defineSlots<{
  /** What is drawn between a row's disclosure and its name. */
  icon(props: { id: RowId; hasChildren: boolean; isOpen: boolean }): unknown
  /** What is said when there is nothing to draw. */
  silence(): unknown
}>()

/* --------------------------------- State ---------------------------------- */
const list = useTemplateRef<HTMLElement>('list')

/** What the tree takes up on screen: a drop lands only over it. */
const box = useTemplateRef<HTMLElement>('box')

const visibleRows = computed(() => flatten(props.rows, new Set(props.open)))

const virtual = useVirtualTree(box, () => visibleRows.value.length, getRowHeight)

const virtualRows = computed(() =>
  visibleRows.value.slice(virtual.window.value.startIndex, virtual.window.value.endIndex),
)

const rowsStyle = computed(() => ({
  blockSize: `${virtual.window.value.totalHeight}px`,
  paddingBlockStart: `${virtual.window.value.offsetY}px`,
  boxSizing: 'border-box' as const,
}))

const rows = useDrawnRows(
  () => visibleRows.value,
  () => props.selected,
  (index) => virtual.scrollToRow(index),
)
const { tabbed, setRowElement } = rows
const { getRef: getRowRef } = useRefRegistry<RowId>(setRowElement)

const selection = useRowSelection(
  () => visibleRows.value,
  () => props.selected,
  (chosen) => emit('select', chosen),
)
const { picked } = selection

const drag = useRowDrag({
  getRows: () => props.rows,
  getShownRows: () => visibleRows.value,
  measure: () => {
    const listEl = list.value
    const overBox = box.value?.getBoundingClientRect()
    if (!listEl || !overBox) return null
    const anyRow = listEl.querySelector('[data-tree-row]')
    const rowHeight = anyRow ? anyRow.getBoundingClientRect().height : 24
    return { hoverRect: overBox, top: listEl.getBoundingClientRect().top, height: rowHeight }
  },
  getThreshold: () => props.threshold,
  getClock: () => props.clock,
  getCountWords: () => props.counted,
  tell: emit,
})
const { into, before, lifted, label, at: dropTarget } = drag

const {
  onContextMenu,
  onRowContextMenu,
  onRowFocus,
  onRowPointerDown,
  onRowClick,
  onRowDoubleClick,
  onRename,
  onAbandon,
  onFieldBlur,
  onKeyDown,
} = useTreeGestures({
  getShownRows: () => visibleRows.value,
  getSelected: () => props.selected,
  renamingPath,
  rows,
  selection,
  drag,
  getMenuAt: (row) => {
    const targetBox = rows.getRowElement(row)?.getBoundingClientRect()
    return targetBox ? { x: targetBox.left, y: targetBox.bottom } : null
  },
  tell: emit,
})

let observer: ResizeObserver | null = null

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  virtual.measureViewport()
  if (typeof ResizeObserver !== 'undefined' && box.value) {
    observer = new ResizeObserver(() => {
      virtual.measureViewport()
    })
    observer.observe(box.value)
  }
})

onBeforeUnmount(() => {
  observer?.disconnect()
})

/* -------------------------------- Helpers --------------------------------- */
function getRowHeight(): number {
  const listEl = list.value
  const anyRow = listEl?.querySelector('[data-tree-row]')
  const height = anyRow?.getBoundingClientRect().height
  return height && height > 0 ? height : 24
}
</script>

<template>
  <div
    ref="box"
    class="tree numen bg-surface text-ink min-h-0 font-sans text-base"
    :data-into="dropTarget && 'into' in dropTarget && dropTarget.into === null ? '' : undefined"
    v-bind="getMarkOf(marking, null)"
    @scroll.passive="virtual.onScroll"
    @contextmenu.prevent="onContextMenu"
  >
    <div
      ref="list"
      class="tree__rows"
      role="tree"
      aria-multiselectable="true"
      :aria-label="name"
      :style="rowsStyle"
      @keydown="onKeyDown"
    >
      <TreeRow
        v-for="row in virtualRows"
        :ref="getRowRef(row.id)"
        :key="row.id"
        :row="row"
        :name="name"
        :is-selected="picked.has(row.id)"
        :is-lifted="lifted.has(row.id)"
        :is-drop-inside="row.id === into"
        :is-drop-above="row.id === before"
        :is-tab-stop="row.id === tabbed"
        :is-renaming="renamingPath === row.id"
        :mark="getMarkOf(marking, row.id)"
        @focus="onRowFocus(row.id)"
        @pointerdown="onRowPointerDown(row.id, $event)"
        @click="onRowClick(row)"
        @dblclick="onRowDoubleClick(row)"
        @contextmenu.prevent.stop="onRowContextMenu(row, $event)"
        @rename="onRename(row.id, $event)"
        @abandon="onAbandon(row.id)"
        @blur="onFieldBlur"
      >
        <slot name="icon" :id="row.id" :has-children="row.hasChildren" :is-open="row.isOpen" />
      </TreeRow>
    </div>

    <p v-if="!visibleRows.length" class="tree__silence p-inset text-hushed">
      <slot name="silence">Nothing here</slot>
    </p>

    <DragPreview v-if="label" class="tree__dragged" :at="label.at" :label="label.label" />
  </div>
</template>

<style scoped>
.tree {
  /* How far one level is set in, how tall a row stands, the room at the edges
     of a row, and the room between a mark and the name beside it. */
  --indent: 0.875rem;
  --row: 1.5rem;
  --pad: 0.25rem;
  --gap: 0.25rem;
  /* How plainly a row on its way somewhere is drawn. */
  --dragged-fade: 0.5;

  display: flex;
  flex-direction: column;
  block-size: 100%;
  overflow: auto;
}

.tree__rows {
  flex: 0 0 auto;
}

/* What a drop at the top level would land inside: the whole tree. */
.tree[data-into] {
}

.tree__silence {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  margin: 0;
  text-align: center;
}

/* The rows being dragged stand over the tree. */
.tree__dragged {
  z-index: 3;
}
</style>
