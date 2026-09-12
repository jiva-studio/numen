<script setup lang="ts">
/**
 * Indented rows, some of which hold other rows.
 *
 * What a row stands for is the caller's: this takes names and hands identities
 * back. What is drawn beside a name comes from a slot, and which rows are open
 * and which are selected are the caller's to hold: the tree works out what a
 * press with a modifier means and says the selection it came to.
 */
import { computed, nextTick, shallowRef, useTemplateRef } from 'vue'
import {
  flatten,
  type Row,
  type RowMarker,
  type RowId,
  type ShownRow,
} from '../lib/row'
import {
  everyRow,
  sameRows,
  resolveSelection,
  PLAIN,
  type Press,
  type RowSelection,
} from '../lib/select'
import { isTreeKey, stepTo } from '../lib/step'
import { getDraggedRows, dragLabel, type DragLabel } from '../lib/drag'
import { holderOf, isRefused, landing, type RowLanding } from '../lib/drop'
import type { Position } from '@/shared/lib/geometry'
import { browserClock, type Clock } from '@/shared/lib/clock'
import { DragPreview, usePressDrag } from '@/shared/ui/drag-preview'
import TreeRow from './TreeRow.vue'

// --- Props & Emits ---
const props = withDefaults(
  defineProps<{
    /** What is drawn, nested. */
    rows: readonly Row[]
    /** The rows whose contents are drawn. */
    open?: readonly RowId[]
    /** The rows the selection stands on. */
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
const renaming = defineModel<RowId | null>('renaming', { default: null })

const emit = defineEmits<{
  (event: 'open', row: RowId): void
  (event: 'close', row: RowId): void
  /** The rows the selection now stands on. */
  (event: 'select', rows: readonly RowId[]): void
  /** A row acted on: a double press, or Enter. A row that holds turns as well. */
  (event: 'activate', row: RowId): void
  /** A name typed and committed. */
  (event: 'rename', row: RowId, name: string): void
  /** The rows let go somewhere, all of them landing in the one place. */
  (event: 'move', rows: readonly RowId[], at: RowLanding): void
  /**
   * The rows lifted clear of the tree, on their way across whatever is drawn
   * beside it. Where they end up there is not the tree's to say.
   */
  (event: 'drag', rows: readonly RowId[]): void
  /** The rows let go of, wherever the pointer had got to. */
  (event: 'drop'): void
  /** The selection asked to go. */
  (event: 'remove', rows: readonly RowId[]): void
  /** A menu asked for, and where the pointer was. Nothing for a press off every row. */
  (event: 'menu', row: RowId | null, at: Position): void
}>()

defineSlots<{
  /** What is drawn between a row's disclosure and its name. */
  icon(props: { id: RowId; holds: boolean; open: boolean }): unknown
  /** What is said when there is nothing to draw. */
  silence(): unknown
}>()

// --- State ---
const list = useTemplateRef<HTMLElement>('list')

/** What the tree takes up on screen: a drop lands only over it. */
const box = useTemplateRef<HTMLElement>('box')

const shown = computed(() => flatten(props.rows, new Set(props.open)))

/** The rows selected, for asking one row at a time. */
const picked = computed(() => new Set(props.selected))

/** The row the keyboard was last on. */
const here = shallowRef<RowId | null>(null)

/** The row a reach is measured from, where a plain or joining press last landed. */
const anchor = shallowRef<RowId | null>(null)

/**
 * The one row the tab key reaches: where the keyboard was left, else the first
 * row of the selection, else the first row of all.
 */
const tabbed = computed<RowId | null>(() => {
  const drawn = (row: RowId | null | undefined) =>
    row != null && shown.value.some((each) => each.id === row) ? row : null
  return drawn(here.value) ?? drawn(props.selected[0]) ?? shown.value[0]?.id ?? null
})

/** Whether the press being made has said what the selection is already. */
const said = shallowRef(false)

const { dragging, at, position, lift } = usePressDrag<readonly RowId[], RowLanding>({
  threshold: () => props.threshold,
  clock: () => props.clock,
  landingAt: getLandingAt,
  settle: (rows, found) => {
    if (found) emit('move', rows, found)
    emit('drop')
  },
  began: (rows) => emit('drag', rows),
})

const into = computed(() => (at.value && 'into' in at.value ? at.value.into : null))
const before = computed(() => (at.value && 'before' in at.value ? at.value.before : null))

/** The rows a live drag holds, for asking one row at a time. */
const lifted = computed(() => new Set(position.value ? (dragging.value?.held ?? []) : []))

/** What follows the pointer, and nothing until a press has become a drag. */
const label = computed<DragLabel | null>(() => {
  const held = dragging.value
  const where = position.value
  if (!held?.moved || !where) return null
  return dragLabel(shown.value, held.held, where, props.counted)
})

/** The rows as they are drawn, each under the row it stands for. */
const drawnRows = new Map<RowId, HTMLElement>()

// --- Handlers ---
function onContextMenu(event: MouseEvent): void {
  requestMenu(null, { x: event.clientX, y: event.clientY })
}

function onRowContextMenu(row: ShownRow, event: MouseEvent): void {
  requestMenu(row, { x: event.clientX, y: event.clientY })
}

function onRowFocus(rowId: RowId): void {
  here.value = rowId
}

function onRowPointerDown(rowId: RowId, event: PointerEvent): void {
  if (event.button !== 0) return
  event.preventDefault()
  ;(event.currentTarget as HTMLElement).focus()

  const how: Press = { joining: event.ctrlKey || event.metaKey, reaching: event.shiftKey }
  said.value = how.joining || how.reaching || !picked.value.has(rowId)
  const taken = said.value
    ? applySelection(resolveSelection(shown.value, props.selected, anchor.value, rowId, how))
    : props.selected

  lift(getDraggedRows(taken, rowId), event)
}

function onRowClick(row: ShownRow): void {
  const spoken = said.value
  said.value = false
  if (dragging.value?.moved || spoken) return
  applySelection(resolveSelection(shown.value, props.selected, anchor.value, row.id, PLAIN))
}

function onRowDoubleClick(row: ShownRow): void {
  activateRow(row)
}

function onRename(rowId: RowId, name: string): void {
  renaming.value = null
  emit('rename', rowId, name)
  void focusRow(rowId)
}

function onAbandon(rowId: RowId): void {
  renaming.value = null
  void focusRow(rowId)
}

function onFieldBlur(): void {
  renaming.value = null
}

function onKeyDown(event: KeyboardEvent): void {
  const chorded = event.ctrlKey || event.metaKey

  if (chorded && event.key.toLowerCase() === 'a') {
    event.preventDefault()
    applySelection(everyRow(shown.value, anchor.value))
    return
  }

  if (event.key === 'Delete' || event.key === 'Backspace') {
    event.preventDefault()
    if (props.selected.length > 0) emit('remove', props.selected)
    return
  }

  const on = shown.value.find((row) => row.id === tabbed.value)
  if (!on) return

  if (event.key === 'Enter') {
    event.preventDefault()
    activateRow(on)
    return
  }

  // The row the keyboard stands on joins the selection, or leaves it.
  if (event.key === ' ') {
    event.preventDefault()
    const press: Press = { joining: true, reaching: false }
    applySelection(resolveSelection(shown.value, props.selected, anchor.value, on.id, press))
    return
  }

  if (event.key === 'ContextMenu' || (event.key === 'F10' && event.shiftKey)) {
    event.preventDefault()
    const elementBox = getRowElement(on.id)?.getBoundingClientRect()
    if (elementBox) requestMenu(on, { x: elementBox.left, y: elementBox.bottom })
    return
  }

  if (!isTreeKey(event.key)) return
  event.preventDefault()

  const step = stepTo(shown.value, tabbed.value, event.key)
  if (step.turn?.open) emit('open', step.turn.row)
  else if (step.turn) emit('close', step.turn.row)
  if (step.at !== null) {
    const press: Press = { joining: false, reaching: event.shiftKey }
    applySelection(resolveSelection(shown.value, props.selected, anchor.value, step.at, press))
  }
  void focusRow(step.at)
}

// --- Helpers ---
/** What a row is marked with, and nothing where it is marked with nothing. */
function getMarkOf(row: RowId | null): Record<string, string> {
  const mark = props.marking
  const value = mark?.valueFor(row)
  return mark && value !== null && value !== undefined ? { [mark.attribute]: value } : {}
}

function setRowElement(row: RowId, element: unknown): void {
  const drawn = (element as { $el?: unknown } | null)?.$el
  if (drawn) drawnRows.set(row, drawn as HTMLElement)
  else drawnRows.delete(row)
}

function getRowElement(row: RowId): HTMLElement | null {
  return drawnRows.get(row) ?? null
}

/** The keyboard onto a row, once the rows it moved among are drawn. */
async function focusRow(row: RowId | null): Promise<void> {
  if (row === null) return
  here.value = row
  await nextTick()
  getRowElement(row)?.focus()
}

function toggleRow(row: ShownRow): void {
  if (row.open) emit('close', row.id)
  else emit('open', row.id)
}

/** A selection a press came to, said, and the anchor put where it names. */
function applySelection(pressed: RowSelection): readonly RowId[] {
  anchor.value = pressed.anchor
  if (!sameRows(pressed.rows, props.selected)) emit('select', pressed.rows)
  return pressed.rows
}

function activateRow(row: ShownRow): void {
  if (row.holds) toggleRow(row)
  emit('activate', row.id)
}

/** A menu asked for on a row, which the selection takes in first, or off every row. */
function requestMenu(row: ShownRow | null, at: Position): void {
  if (row && !picked.value.has(row.id)) {
    applySelection(resolveSelection(shown.value, props.selected, anchor.value, row.id, PLAIN))
  }
  emit('menu', row?.id ?? null, at)
}

/**
 * Where the pointer is, asked of the drawing: the rows are one height each.
 */
function getLandingAt(rows: readonly RowId[], at: Position): RowLanding | null {
  const drawn = list.value
  const over = box.value?.getBoundingClientRect()
  if (!drawn || !over) return null

  // A pointer that has left the tree is taking what it holds somewhere else.
  const inside =
    at.x >= over.left && at.x <= over.right && at.y >= over.top && at.y <= over.bottom
  if (!inside) return null

  const first = shown.value[0]
  const height =
    (first && getRowElement(first.id)?.getBoundingClientRect().height) ?? 0
  const found = landing(shown.value, rows, at.y - drawn.getBoundingClientRect().top, height)
  if (!found) return null

  return isRefused(props.rows, rows, holderOf(shown.value, found)) ? null : found
}
</script>

<template>
  <div
    ref="box"
    class="tree numen min-h-0 bg-surface font-sans text-base text-ink"
    :data-into="at && 'into' in at && at.into === null ? '' : undefined"
    v-bind="getMarkOf(null)"
    @contextmenu.prevent="onContextMenu"
  >
    <div
      ref="list"
      class="tree__rows"
      role="tree"
      aria-multiselectable="true"
      :aria-label="name"
      @keydown="onKeyDown"
    >
      <TreeRow
        v-for="row in shown"
        :ref="(element) => setRowElement(row.id, element)"
        :key="row.id"
        :row="row"
        :name="name"
        :selected="picked.has(row.id)"
        :lifted="lifted.has(row.id)"
        :into="row.id === into"
        :before="row.id === before"
        :tabbed="row.id === tabbed"
        :renaming="renaming === row.id"
        :mark="getMarkOf(row.id)"
        @focus="onRowFocus(row.id)"
        @pointerdown="onRowPointerDown(row.id, $event)"
        @click="onRowClick(row)"
        @dblclick="onRowDoubleClick(row)"
        @contextmenu.prevent.stop="onRowContextMenu(row, $event)"
        @rename="onRename(row.id, $event)"
        @abandon="onAbandon(row.id)"
        @blur="onFieldBlur"
      >
        <slot name="icon" :id="row.id" :holds="row.holds" :open="row.open" />
      </TreeRow>
    </div>

    <p v-if="!shown.length" class="tree__silence p-inset text-hushed">
      <slot name="silence">Nothing here</slot>
    </p>

    <DragPreview
      v-if="label"
      class="tree__dragged"
      :at="label.at"
      :says="label.says"
    />
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

/* What is said in place of the rows stands in the middle of the tree. */
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
