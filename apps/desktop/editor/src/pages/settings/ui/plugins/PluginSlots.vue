<script setup lang="ts">
/**
 * The slots a plugin draws, each an island with the views placed in it. A switch hides a view, and a
 * grip drags it to a new place in its slot; the plugin keeps the result in its setting of the type slots.
 */
import { browserClock, IslandCard, measureRect, usePressDrag } from '@numen/ui'
import type { JsonValue } from '@/shared/api/json'
import {
  getSlotRows,
  moveSlotRow,
  toggleSlotRow,
  writeSlotRows,
  type SlotRow,
} from '../../lib/slotLayout'
import { clampRowOffset, getDropIndex, getRowShifts, type RowSpan } from '../../lib/slotDrag'
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
interface Pressed {
  group: SlotGroup
  spans: readonly RowSpan[]
  key: string
  startY: number
}
let pressed: Pressed | null = null

const drag = usePressDrag<number, number>({
  getThreshold: () => DRAG_THRESHOLD,
  getClock: () => browserClock,
  getDropTargetAt: (from, point) =>
    pressed ? getDropIndex(pressed.spans, from, point.y - pressed.startY) : null,
  commit: (from, to) => {
    if (pressed && to !== null)
      onChange(pressed.group, moveSlotRow(rowsOf(pressed.group), from, to))
  },
})

/* -------------------------------- Handlers -------------------------------- */
function onGripPress(group: SlotGroup, at: number, event: PointerEvent): void {
  if (event.button !== 0) return
  const grip = event.currentTarget as HTMLElement | null
  const list = grip?.closest('ul') ?? null
  const spans = Array.from(list?.children ?? [], (one) => {
    const { top, height } = measureRect(one)
    return { top, height }
  })
  pressed = { group, spans, key: rowsOf(group)[at]?.key ?? '', startY: event.clientY }
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

const isHeld = (group: SlotGroup, key: string): boolean =>
  pressed?.group === group && pressed.key === key && drag.dragging.value?.hasMoved === true

const isLifted = (group: SlotGroup, key: string): boolean =>
  isHeld(group, key) && drag.position.value !== null

const isShifting = (group: SlotGroup): boolean =>
  pressed?.group === group && drag.position.value !== null

function getOffset(group: SlotGroup, key: string, at: number): number {
  const point = drag.position.value
  const from = drag.dragging.value?.item
  if (!pressed || pressed.group !== group || !point || from === undefined) return 0
  if (key === pressed.key) return clampRowOffset(pressed.spans, from, point.y - pressed.startY)
  return getShift(group, from, at)
}

function getShift(group: SlotGroup, from: number, at: number): number {
  const pitch = pressed?.spans[from]?.height ?? 0
  const shifts = getRowShifts(rowsOf(group).length, from, drag.at.value ?? from, pitch)
  return shifts[at] ?? 0
}

function getRowStyle(
  group: SlotGroup,
  key: string,
  at: number,
): Record<string, string> | undefined {
  const offset = getOffset(group, key, at)
  return offset === 0 ? undefined : { transform: `translateY(${offset}px)` }
}
</script>

<template>
  <IslandCard
    v-for="group in props.groups"
    :key="group.slot"
    :title="group.title"
    :data-testid="`slots-${group.slot}`"
  >
    <TransitionGroup tag="ul" name="plugin-slots" class="plugin-slots__list">
      <li
        v-for="(row, at) in rowsOf(group)"
        :key="row.key"
        class="plugin-slots__row"
        :class="{
          'plugin-slots__row--held': isHeld(group, row.key),
          'plugin-slots__row--lifted': isLifted(group, row.key),
          'plugin-slots__row--shifting': isShifting(group) && !isLifted(group, row.key),
        }"
        :style="getRowStyle(group, row.key, at)"
      >
        <PluginSlotRow
          :row="row"
          :is-movable="group.contributors.length > 1"
          @grip-press="(event) => onGripPress(group, at, event)"
          @grip-key="(event) => onGripKey(group, at, event)"
          @toggle="onChange(group, toggleSlotRow(rowsOf(group), at))"
        />
      </li>
    </TransitionGroup>
  </IslandCard>
</template>

<style scoped>
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
  gap: 0.375rem;
  padding-block: 0.125rem;
}

.plugin-slots__row--held {
  position: relative;
  z-index: 1;
  transition: none;
}

.plugin-slots__row--lifted {
  background: var(--numen-raised);
  box-shadow: var(--numen-shadow-card);
  opacity: 0.9;
  transition: none;
}

.plugin-slots__row--shifting,
.plugin-slots-move {
  transition: transform var(--numen-motion) var(--numen-easing);
}

@media (prefers-reduced-motion: reduce) {
  .plugin-slots__row--shifting,
  .plugin-slots-move {
    transition: none;
  }
}
</style>
