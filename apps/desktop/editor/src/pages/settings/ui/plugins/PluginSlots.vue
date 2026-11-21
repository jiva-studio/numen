<script setup lang="ts">
/**
 * The slots a plugin draws, each an island with the views placed in it. A switch hides a view, and a
 * grip drags it to a new place in its slot; the plugin keeps the result in its setting of the type slots.
 */
import { browserClock, measureRect, usePressDrag } from '@numen/ui'
import type { JsonValue } from '@/shared/api/json'
import {
  getSlotRows,
  moveSlotRow,
  toggleSlotRow,
  writeSlotRows,
  type SlotRow,
} from '../../lib/slotLayout'
import type { SlotGroup } from '../../types'
import PluginSlotRow from './PluginSlotRow.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  groups: readonly SlotGroup[]
  layout: JsonValue | undefined
}>()

const emit = defineEmits<{
  (e: 'change', layout: JsonValue): void
}>()

/* --------------------------------- State ---------------------------------- */
const DRAG_THRESHOLD = 4
let pressed: { group: SlotGroup; list: HTMLElement | null } | null = null

const drag = usePressDrag<number, number>({
  getThreshold: () => DRAG_THRESHOLD,
  getClock: () => browserClock,
  getDropTargetAt: (_from, point) => findIndexAt(point.y),
  commit: (from, to) => {
    if (pressed && to !== null)
      onChange(pressed.group, moveSlotRow(rowsOf(pressed.group), from, to))
  },
})

/* -------------------------------- Handlers -------------------------------- */
function onGripPress(group: SlotGroup, at: number, event: PointerEvent): void {
  if (event.button !== 0) return
  const list = (event.currentTarget as HTMLElement | null)?.closest('ul') ?? null
  pressed = { group, list }
  drag.startPress(at, event)
}

function onGripKey(group: SlotGroup, at: number, event: KeyboardEvent): void {
  const by = event.key === 'ArrowUp' ? -1 : event.key === 'ArrowDown' ? 1 : 0
  if (by === 0) return
  event.preventDefault()
  onChange(group, moveSlotRow(rowsOf(group), at, at + by))
}

function onChange(group: SlotGroup, rows: readonly SlotRow[]): void {
  emit('change', writeSlotRows(props.layout, group.slot, rows))
}

/* --------------------------------- Helpers -------------------------------- */
const rowsOf = (group: SlotGroup): readonly SlotRow[] => getSlotRows(group, props.layout)

function findIndexAt(y: number): number | null {
  const items = Array.from(pressed?.list?.children ?? [])
  if (items.length === 0) return null
  const found = items.findIndex((one) => y < measureRect(one).bottom)
  return found < 0 ? items.length - 1 : found
}

const isDragging = (group: SlotGroup, at: number): boolean =>
  pressed?.group === group &&
  drag.dragging.value?.hasMoved === true &&
  drag.dragging.value.item === at

const isDropTarget = (group: SlotGroup, at: number): boolean =>
  pressed?.group === group && drag.dragging.value?.hasMoved === true && drag.at.value === at
</script>

<template>
  <section
    v-for="group in props.groups"
    :key="group.slot"
    class="settings-card plugin-slots"
    :data-testid="`slots-${group.slot}`"
  >
    <h3 class="plugin-slots__title">{{ group.title }}</h3>
    <ul class="plugin-slots__list">
      <li
        v-for="(row, at) in rowsOf(group)"
        :key="row.key"
        class="plugin-slots__row"
        :class="{
          'plugin-slots__row--dragged': isDragging(group, at),
          'plugin-slots__row--target': isDropTarget(group, at),
        }"
      >
        <PluginSlotRow
          :row="row"
          :is-movable="group.contributors.length > 1"
          @grip-press="(event) => onGripPress(group, at, event)"
          @grip-key="(event) => onGripKey(group, at, event)"
          @toggle="onChange(group, toggleSlotRow(rowsOf(group), at))"
        />
      </li>
    </ul>
  </section>
</template>

<style scoped>
.plugin-slots {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: var(--settings-card-pad);
}

.plugin-slots__title {
  margin: 0;
  font-size: var(--numen-text-2);
  font-weight: 600;
  color: var(--numen-ink);
}

.plugin-slots__list {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.plugin-slots__row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding-block: 0.25rem;
}

.plugin-slots__row--dragged {
  opacity: 0.5;
}

.plugin-slots__row--target {
  box-shadow: inset 0 calc(-1 * var(--numen-stroke)) 0 var(--numen-accent);
}
</style>
