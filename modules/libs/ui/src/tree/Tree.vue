<script setup lang="ts">
/**
 * Indented rows, some of which hold other rows.
 *
 * What a row stands for is the caller's: this takes names and hands identities
 * back. What is drawn beside a name comes from a slot, and which rows are open
 * and which are selected are the caller's to hold: the tree works out what a
 * press with a modifier means and says the selection it came to.
 */
import { computed, nextTick, onBeforeUnmount, shallowRef, useTemplateRef, watch } from 'vue'
import {
  carried,
  carries,
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
  type Carried,
  type Landing,
  type Press,
  type Pressed,
  type Row,
  type RowId,
  type ShownRow,
} from './model'
import type { Point } from '../plex/model'

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
    /** How many rows are being carried, said at the pointer. */
    counted?: (rows: number) => string
    /** When the next frame comes. */
    frame?: (run: () => void) => void
  }>(),
  {
    open: () => [],
    selected: () => [],
    threshold: 4,
    name: 'Tree',
    counted: (rows: number) => `${rows} rows`,
    frame: (run: () => void) => {
      requestAnimationFrame(run)
    },
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
  (event: 'move', rows: readonly RowId[], at: Landing): void
  /** The selection asked to go. */
  (event: 'remove', rows: readonly RowId[]): void
  /** A menu asked for, and where the pointer was. Nothing for a press off every row. */
  (event: 'menu', row: RowId | null, at: Point): void
}>()

defineSlots<{
  /** What is drawn between a row's disclosure and its name. */
  icon(props: { id: RowId; holds: boolean; open: boolean }): unknown
  /** What is said when there is nothing to draw. */
  silence(): unknown
}>()

const list = useTemplateRef<HTMLElement>('list')

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

/** The rows under the pointer, once the pointer has gone far enough to mean it. */
interface Dragging {
  readonly rows: readonly RowId[]
  readonly startX: number
  readonly startY: number
  readonly moved: boolean
}

const dragging = shallowRef<Dragging | null>(null)
/** Whether the press being made has said what the selection is already. */
const said = shallowRef(false)
const at = shallowRef<Landing | null>(null)
/** Where the pointer is, for as long as a drag is live. */
const point = shallowRef<Point | null>(null)

const into = computed(() => (at.value && 'into' in at.value ? at.value.into : null))
const before = computed(() => (at.value && 'before' in at.value ? at.value.before : null))

/** The rows a live drag is carrying, for asking one row at a time. */
const lifted = computed(() => new Set(point.value ? (dragging.value?.rows ?? []) : []))

/** What follows the pointer, and nothing until a press has become a drag. */
const carrying = computed<Carried | null>(() => {
  const held = dragging.value
  const where = point.value
  if (!held?.moved || !where) return null
  return carried(shown.value, held.rows, where, props.counted)
})

const rowFor = (row: RowId): HTMLElement | null =>
  [...(list.value?.querySelectorAll<HTMLElement>('[data-tree-row]') ?? [])].find(
    (each) => each.getAttribute('data-tree-row') === row,
  ) ?? null

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
const takes = (pressed: Pressed): readonly RowId[] => {
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
const askMenu = (row: ShownRow | null, point: Point): void => {
  if (row && !picked.value.has(row.id)) {
    takes(selects(shown.value, props.selected, anchor.value, row.id, PLAIN))
  }
  emit('menu', row?.id ?? null, point)
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
 * carried alone; a row standing in the selection carries the whole of it, and
 * a plain press collapses the selection onto it once the pointer has let go
 * without travelling.
 */
function lift(row: RowId, event: PointerEvent): void {
  if (event.button !== 0) return
  event.preventDefault()
  ;(event.currentTarget as HTMLElement).focus()

  const press: Press = { joining: event.ctrlKey || event.metaKey, reaching: event.shiftKey }
  said.value = press.joining || press.reaching || !picked.value.has(row)
  const taken = said.value
    ? takes(selects(shown.value, props.selected, anchor.value, row, press))
    : props.selected

  dragging.value = {
    rows: carries(taken, row),
    startX: event.clientX,
    startY: event.clientY,
    moved: false,
  }
  window.addEventListener('pointermove', drag)
  window.addEventListener('pointerup', drop)
  window.addEventListener('pointercancel', drop)
}

function drag(event: PointerEvent): void {
  const held = dragging.value
  if (!held) return

  const moved =
    held.moved ||
    Math.abs(event.clientX - held.startX) > props.threshold ||
    Math.abs(event.clientY - held.startY) > props.threshold

  dragging.value = { ...held, moved }
  point.value = moved ? { x: event.clientX, y: event.clientY } : null
  at.value = moved ? landingAt(held.rows, event.clientY) : null
}

function drop(): void {
  const held = dragging.value
  const found = at.value

  window.removeEventListener('pointermove', drag)
  window.removeEventListener('pointerup', drop)
  window.removeEventListener('pointercancel', drop)

  if (held?.moved && found) emit('move', held.rows, found)
  at.value = null
  point.value = null
  // Held one frame longer: the click that follows the release reads it and
  // stands down.
  props.frame(() => {
    dragging.value = null
  })
}

/**
 * Where the pointer is, asked of the drawing: the rows are one height each,
 * and the height is whatever they are drawn at.
 */
function landingAt(rows: readonly RowId[], clientY: number): Landing | null {
  const drawn = list.value
  if (!drawn) return null

  const height = drawn.querySelector('[data-tree-row]')?.getBoundingClientRect().height ?? 0
  const found = landing(shown.value, rows, clientY - drawn.getBoundingClientRect().top, height)
  if (!found) return null

  return refuses(props.rows, rows, holderOf(shown.value, found)) ? null : found
}

/** The field, once it is drawn, with the name in it ready to be replaced. */
watch(renaming, (row) => {
  if (row === null) return
  void nextTick(() => {
    const field = list.value?.querySelector<HTMLInputElement>('.tree__field')
    field?.focus()
    field?.select()
  })
})

function onFieldKey(event: KeyboardEvent): void {
  const field = event.currentTarget as HTMLInputElement
  const row = renaming.value

  if (event.key === 'Enter') {
    event.preventDefault()
    renaming.value = null
    if (row !== null) {
      emit('rename', row, field.value)
      void goTo(row)
    }
    return
  }

  if (event.key === 'Escape') {
    event.preventDefault()
    renaming.value = null
    void goTo(row)
  }
}

onBeforeUnmount(() => {
  window.removeEventListener('pointermove', drag)
  window.removeEventListener('pointerup', drop)
  window.removeEventListener('pointercancel', drop)
  at.value = null
  point.value = null
  dragging.value = null
})
</script>

<template>
  <div
    class="tree numen min-h-0 bg-surface font-sans text-base text-ink"
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
        :key="row.id"
        class="tree__row flex min-w-0 items-center gap-1.5"
        role="treeitem"
        :aria-level="row.level"
        :aria-expanded="row.holds ? row.open : undefined"
        :aria-selected="picked.has(row.id)"
        :tabindex="row.id === tabbed ? 0 : -1"
        :data-tree-row="row.id"
        :data-selected="picked.has(row.id) || undefined"
        :data-carried="lifted.has(row.id) || undefined"
        :data-last="row.last || undefined"
        :data-into="row.id === into || undefined"
        :data-before="row.id === before || undefined"
        :style="{ '--level': row.level }"
        @focus="here = row.id"
        @pointerdown="lift(row.id, $event)"
        @click="choose(row)"
        @dblclick="act(row)"
        @contextmenu.prevent.stop="askMenu(row, { x: $event.clientX, y: $event.clientY })"
      >
        <button
          v-if="row.holds"
          class="tree__twist shrink-0"
          type="button"
          tabindex="-1"
          aria-hidden="true"
          data-tree-twist
          @pointerdown.stop
          @click.stop="turn(row)"
        />
        <span v-else class="tree__twist shrink-0" aria-hidden="true" />

        <span class="tree__icon flex shrink-0 items-center">
          <slot name="icon" :id="row.id" :holds="row.holds" :open="row.open" />
        </span>

        <input
          v-if="renaming === row.id"
          class="tree__field min-w-0 grow rounded-node"
          type="text"
          :value="row.name"
          :aria-label="name"
          @pointerdown.stop
          @click.stop
          @dblclick.stop
          @keydown.stop="onFieldKey"
          @blur="renaming = null"
        />
        <!-- The whole name is on the element, for one too long to be drawn. -->
        <span v-else class="tree__name min-w-0 truncate" :title="row.name">{{ row.name }}</span>
      </div>
    </div>

    <p v-if="!shown.length" class="tree__silence p-inset text-hushed">
      <slot name="silence">Nothing here</slot>
    </p>

    <p
      v-if="carrying"
      class="tree__carried font-sans text-small"
      :style="{ left: `${carrying.at.x}px`, top: `${carrying.at.y}px` }"
    >
      {{ carrying.says }}
    </p>
  </div>
</template>

<style scoped>
.tree {
  /* How far one level is set in, how tall a row stands, the reach of the
     disclosure and the mark on it, and the line standing where a dragged row
     would land. */
  --indent: 0.875rem;
  --row: 1.5rem;
  --twist: 0.75rem;
  --twist-mark: 0.4rem;
  --caret: 2px;
  /* What is carried: how far it stands clear of the pointer, how far it
     reaches before the name is cut, the room the name is given, and how
     plainly a row on its way is drawn. */
  --carried-gap: 0.75rem;
  --carried-widest: 15rem;
  --carried-pad: 0.15rem 0.5rem;
  --carried-fade: 0.5;

  block-size: 100%;
  overflow: auto;
}

.tree__row {
  position: relative;
  block-size: var(--row);
  padding-inline: calc(var(--numen-inset) + var(--indent) * (var(--level) - 1))
    var(--numen-inset);
  cursor: default;
  user-select: none;
  touch-action: none;
}

.tree__row:hover {
  background: var(--numen-bubble-bg);
}

.tree__row[data-selected] {
  background: var(--numen-focus-bg);
  color: var(--numen-focus-fg);
}

/* A row on its way somewhere, drawn plainly where it stands. */
.tree__row[data-carried] {
  opacity: var(--carried-fade);
}

.tree__row:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: calc(-1 * var(--numen-ring-width));
}

/* The row a drag would land inside, and the line a drag would land on. */
.tree__row[data-into] {
  box-shadow: inset 0 0 0 var(--numen-ring-width) var(--numen-ring);
}

.tree__row[data-before]::before {
  content: '';
  position: absolute;
  inset-block-start: 0;
  inset-inline: calc(var(--numen-inset) + var(--indent) * (var(--level) - 1)) 0;
  block-size: var(--caret);
  background: var(--numen-ring);
}

/* A triangle lying on its side, standing up as what it holds is drawn. */
.tree__twist {
  inline-size: var(--twist);
  block-size: var(--twist);
}

button.tree__twist::before {
  content: '';
  display: block;
  inline-size: var(--twist-mark);
  block-size: var(--twist-mark);
  margin-inline: auto;
  background: currentColor;
  clip-path: polygon(20% 0%, 100% 50%, 20% 100%);
  transition: transform var(--numen-motion-hover) var(--numen-easing);
}

.tree__row[aria-expanded='true'] button.tree__twist::before {
  transform: rotate(90deg);
}

button.tree__twist:focus-visible {
  outline: none;
}

.tree__field {
  border: var(--numen-stroke) solid var(--numen-field-border);
  background: var(--numen-field-bg);
  color: var(--numen-node-fg);
  font: inherit;
}

.tree__field:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: calc(-1 * var(--numen-ring-width));
}

.tree__silence {
  margin: 0;
}

/* What is being carried, said beside the pointer and catching nothing. One
   line, then an ellipsis. */
.tree__carried {
  position: fixed;
  z-index: 3;
  max-inline-size: var(--carried-widest);
  margin: 0;
  padding: var(--carried-pad);
  translate: var(--carried-gap) var(--carried-gap);
  pointer-events: none;
  overflow: hidden;
  border-radius: var(--numen-radius);
  background: var(--numen-focus-bg);
  color: var(--numen-focus-fg);
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (prefers-reduced-motion: reduce) {
  button.tree__twist::before {
    transition: none;
  }
}
</style>
