<script setup lang="ts">
/**
 * The slots a plugin draws, each an island with the views placed in it. A switch hides a view, and a
 * grip drags it to a new place in its slot; the plugin keeps the result in its setting of the type slots.
 */
import { IslandCard } from '@numen/ui'
import type { JsonValue } from '@/shared/api/json'
import {
  getSlotRows,
  moveSlotRow,
  toggleSlotRow,
  writeSlotRows,
  type SlotRow,
} from '../../lib/slotLayout'
import type { SlotGroup } from '../../types'
import { useSlotDrag } from '../../model/useSlotDrag'
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
const { settling, onGripPress, onRowTransitionEnd, isHeld, isShifting, getRowStyle } = useSlotDrag({
  getRows: (group) => rowsOf(group),
  move: (group, from, to) => onChange(group, moveSlotRow(rowsOf(group), from, to)),
})

/* -------------------------------- Handlers -------------------------------- */
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
</script>

<template>
  <IslandCard
    v-for="group in props.groups"
    :key="group.slot"
    :title="group.title"
    :data-testid="`slots-${group.slot}`"
  >
    <ul class="plugin-slots__list" :class="{ 'plugin-slots__list--dragging': isShifting(group) }">
      <li
        v-for="(row, at) in rowsOf(group)"
        :key="row.key"
        class="plugin-slots__row"
        :class="{
          'plugin-slots__row--held': isHeld(group, row.key),
          'plugin-slots__row--lifted': isHeld(group, row.key),
          'plugin-slots__row--shifting': isShifting(group) && !isHeld(group, row.key),
          'plugin-slots__row--settling': settling?.key === row.key && settling.isMoving,
        }"
        :style="getRowStyle(group, row.key, at)"
        @transitionend="onRowTransitionEnd"
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

.plugin-slots__list--dragging {
  user-select: none;
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
.plugin-slots__row--settling {
  transition: transform var(--numen-motion) var(--numen-easing);
}

@media (prefers-reduced-motion: reduce) {
  .plugin-slots__row--shifting,
  .plugin-slots__row--settling {
    transition: none;
  }
}
</style>
