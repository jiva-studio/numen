<script setup lang="ts">
/**
 * Indented rows, some of which hold other rows.
 *
 * What a row stands for is the caller's: this takes names and hands identities
 * back. What is drawn beside a name comes from a slot, and which rows are open
 * and which are selected are the caller's to hold: the tree works out what a
 * press with a modifier means and says the selection it came to.
 */
import { computed, nextTick, shallowRef, useTemplateRef, watch } from 'vue'
import {
  dragged,
  dragLabel,
  everyRow,
  flatten,
  holderOf,
  isTreeKey,
  landing,
  refuses,
  sameRows,
  selects,
  stepTo,
  PLAIN,
  type DragLabel,
  type RowLanding,
  type Press,
  type Row,
  type RowMarker,
  type RowSelection,
  type RowId,
  type ShownRow,
} from './row'
import type { Position } from '@/shared/lib/geometry'
import { browserClock, type Clock } from '@/shared/lib/clock'
import { DragPreview, usePressDrag } from '@/shared/ui/drag-preview'
import { TreeField } from './tree-field'

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

const list = useTemplateRef<HTMLElement>('list')

/** What the tree takes up on screen: a drop lands only over it. */
const box = useTemplateRef<HTMLElement>('box')

/** The field a name is typed in. One row is renamed at a time. */
const field = useTemplateRef<InstanceType<typeof TreeField>[]>('field')

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
  landingAt,
  settle: (rows, found) => {
    if (found) emit('move', rows, found)
    emit('drop')
  },
  // The rows are clear of the tree the moment the press turns into a drag,
  // and said once for the whole of it.
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

/** What a row is marked with, and nothing where it is marked with nothing. */
const markOf = (row: RowId | null): Record<string, string> => {
  const mark = props.marking
  const value = mark?.valueFor(row)
  return mark && value !== null && value !== undefined ? { [mark.attribute]: value } : {}
}

/** The rows as they are drawn, each under the row it stands for. */
const drawnRows = new Map<RowId, HTMLElement>()

const holdRow = (row: RowId, element: unknown): void => {
  if (element) drawnRows.set(row, element as HTMLElement)
  else drawnRows.delete(row)
}

const rowFor = (row: RowId): HTMLElement | null => drawnRows.get(row) ?? null

/** The keyboard onto a row, once the rows it moved among are drawn. */
const goTo = async (row: RowId | null): Promise<void> => {
  if (row === null) return
  here.value = row
  await nextTick()
  rowFor(row)?.focus()
}

const turn = (row: ShownRow): void => {
  if (row.open) emit('close', row.id)
  else emit('open', row.id)
}

/** A selection a press came to, said, and the anchor put where it names. */
const takes = (pressed: RowSelection): readonly RowId[] => {
  anchor.value = pressed.anchor
  if (!sameRows(pressed.rows, props.selected)) emit('select', pressed.rows)
  return pressed.rows
}

const choose = (row: ShownRow): void => {
  const spoken = said.value
  said.value = false
  if (dragging.value?.moved || spoken) return
  takes(selects(shown.value, props.selected, anchor.value, row.id, PLAIN))
}

const act = (row: ShownRow): void => {
  if (row.holds) turn(row)
  emit('activate', row.id)
}

/** A menu asked for on a row, which the selection takes in first, or off every row. */
const askMenu = (row: ShownRow | null, at: Position): void => {
  if (row && !picked.value.has(row.id)) {
    takes(selects(shown.value, props.selected, anchor.value, row.id, PLAIN))
  }
  emit('menu', row?.id ?? null, at)
}

const onKey = (event: KeyboardEvent): void => {
  const chorded = event.ctrlKey || event.metaKey

  if (chorded && event.key.toLowerCase() === 'a') {
    event.preventDefault()
    takes(everyRow(shown.value, anchor.value))
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
    act(on)
    return
  }

  // The row the keyboard stands on joins the selection, or leaves it.
  if (event.key === ' ') {
    event.preventDefault()
    const press: Press = { joining: true, reaching: false }
    takes(selects(shown.value, props.selected, anchor.value, on.id, press))
    return
  }

  if (event.key === 'ContextMenu' || (event.key === 'F10' && event.shiftKey)) {
    event.preventDefault()
    const box = rowFor(on.id)?.getBoundingClientRect()
    if (box) askMenu(on, { x: box.left, y: box.bottom })
    return
  }

  if (!isTreeKey(event.key)) return
  event.preventDefault()

  const step = stepTo(shown.value, tabbed.value, event.key)
  if (step.turn?.open) emit('open', step.turn.row)
  else if (step.turn) emit('close', step.turn.row)
  if (step.at !== null) {
    const press: Press = { joining: false, reaching: event.shiftKey }
    takes(selects(shown.value, props.selected, anchor.value, step.at, press))
  }
  void goTo(step.at)
}

/**
 * A row is lifted under the primary button and under no other. The press
 * selects no text as it travels, and takes the keyboard itself.
 *
 * A row standing outside the selection is what the press selects, and it is
 * dragged alone; a row standing in the selection drags the whole of it, and
 * a plain press collapses the selection onto it once the pointer has let go
 * without travelling.
 */
function press(row: RowId, event: PointerEvent): void {
  if (event.button !== 0) return
  event.preventDefault()
  ;(event.currentTarget as HTMLElement).focus()

  const how: Press = { joining: event.ctrlKey || event.metaKey, reaching: event.shiftKey }
  said.value = how.joining || how.reaching || !picked.value.has(row)
  const taken = said.value
    ? takes(selects(shown.value, props.selected, anchor.value, row, how))
    : props.selected

  lift(dragged(taken, row), event)
}

/**
 * Where the pointer is, asked of the drawing: the rows are one height each,
 * and the height is whatever they are drawn at.
 */
function landingAt(rows: readonly RowId[], at: Position): RowLanding | null {
  const drawn = list.value
  const over = box.value?.getBoundingClientRect()
  if (!drawn || !over) return null

  // A pointer that has left the tree is taking what it holds somewhere else.
  const inside =
    at.x >= over.left && at.x <= over.right && at.y >= over.top && at.y <= over.bottom
  if (!inside) return null

  const first = shown.value[0]
  const height =
    (first && rowFor(first.id)?.getBoundingClientRect().height) ?? 0
  const found = landing(shown.value, rows, at.y - drawn.getBoundingClientRect().top, height)
  if (!found) return null

  return refuses(props.rows, rows, holderOf(shown.value, found)) ? null : found
}

/** The keyboard into the field once it is drawn. */
watch(renaming, (row) => {
  if (row === null) return
  void nextTick(() => field.value?.[0]?.focus())
})

/** A name taken, and the keyboard back on the row it belongs to. */
const rename = (row: RowId, name: string): void => {
  renaming.value = null
  emit('rename', row, name)
  void goTo(row)
}

/** A name left as it was, and the keyboard back on the row. */
const abandon = (row: RowId): void => {
  renaming.value = null
  void goTo(row)
}
</script>

<template>
  <div
    ref="box"
    class="tree numen min-h-0 bg-surface font-sans text-base text-ink"
    :data-into="at && 'into' in at && at.into === null ? '' : undefined"
    v-bind="markOf(null)"
    @contextmenu.prevent="askMenu(null, { x: $event.clientX, y: $event.clientY })"
  >
    <div
      ref="list"
      class="tree__rows"
      role="tree"
      aria-multiselectable="true"
      :aria-label="name"
      @keydown="onKey"
    >
      <div
        v-for="row in shown"
        :ref="(element) => holdRow(row.id, element)"
        :key="row.id"
        class="tree__row flex min-w-0 items-center"
        role="treeitem"
        :aria-level="row.level"
        :aria-expanded="row.holds ? row.open : undefined"
        :aria-selected="picked.has(row.id)"
        :tabindex="row.id === tabbed ? 0 : -1"
        :data-tree-row="row.id"
        :data-selected="picked.has(row.id) || undefined"
        :data-dragged="lifted.has(row.id) || undefined"
        :data-last="row.last || undefined"
        :data-into="row.id === into || undefined"
        :data-before="row.id === before || undefined"
        v-bind="markOf(row.id)"
        :style="{ '--level': row.level }"
        @focus="here = row.id"
        @pointerdown="press(row.id, $event)"
        @click="choose(row)"
        @dblclick="act(row)"
        @contextmenu.prevent.stop="askMenu(row, { x: $event.clientX, y: $event.clientY })"
      >
        <span class="tree__icon flex shrink-0 items-center">
          <slot name="icon" :id="row.id" :holds="row.holds" :open="row.open" />
        </span>

        <TreeField
          v-if="renaming === row.id"
          ref="field"
          :value="row.name"
          :name="name"
          @rename="rename(row.id, $event)"
          @abandon="abandon(row.id)"
          @blur="renaming = null"
        />
        <!-- The whole name is on the element, for one too long to be drawn. -->
        <span v-else class="tree__name min-w-0 truncate" :title="row.name">{{ row.name }}</span>
      </div>
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

.tree__row {
  position: relative;
  block-size: var(--row);
  padding-inline: calc(var(--pad) + var(--indent) * (var(--level) - 1)) var(--pad);
  cursor: default;
  user-select: none;
  touch-action: none;
}

.tree__row:hover {
  background: var(--numen-bubble-bg);
}

.tree__row[data-selected] {
  background: var(--numen-accent);
  color: var(--numen-accent-ink);
}

/* A row on its way somewhere, drawn plainly where it stands. */
.tree__row[data-dragged] {
  opacity: var(--dragged-fade);
}

/* Where the keyboard stands. */
.tree__row:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: calc(-1 * var(--numen-ring-width));
}

/* A row the selection already marks is marked once. */
.tree__row[data-selected]:focus-visible {
  outline: none;
}

/* What a drop would land inside: the row, or the whole tree for the top level. */
.tree__row[data-into],
.tree[data-into] {
  box-shadow: inset 0 0 0 var(--numen-ring-width) var(--numen-ring);
}

.tree__row[data-before]::before {
  content: '';
  position: absolute;
  inset-block-start: 0;
  inset-inline: calc(var(--pad) + var(--indent) * (var(--level) - 1)) 0;
  block-size: var(--numen-caret);
  background: var(--numen-ring);
}

/* A name stands clear of the mark beside it. */
.tree__icon {
  margin-inline-end: var(--gap);
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
