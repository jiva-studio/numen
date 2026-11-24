<script setup lang="ts">
/**
 * The slots a plugin draws, each an island with the views placed in it. A switch hides a view, and a
 * grip drags it to a new place in its slot; a separator is added from the heading of the island, dragged
 * like a view and removed from its row, and a slot with a stored list can be given back to the manifests.
 * The plugin keeps the result in its setting of the type slots.
 */
import { RotateCcw, SeparatorHorizontal } from '@lucide/vue'
import { IslandCard } from '@numen/ui'
import type { JsonValue } from '@/shared/api/json'
import {
  addSlotSeparator,
  getSeparatorPlace,
  getSlotRows,
  hasSlotLayout,
  moveSlotRow,
  removeSlotSeparator,
  resetSlotRows,
  toggleSlotRow,
  writeSlotRows,
  type SlotRow,
} from '../../lib/slotLayout'
import type { SlotGroup } from '../../types'
import { useSlotDrag } from '../../model/useSlotDrag'
import { WORDS as words } from '../../words'
import RowAction from '../setting-row/RowAction.vue'
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
const { onGripPress, onRowTransitionRun, onRowTransitionEnd, isPressed, getRowClass, getRowStyle } =
  useSlotDrag({
    getRows: (group) => getRows(group),
    move: (group, from, to) => {
      const rows = moveSlotRow(getRows(group), from, to)
      onChange(group, rows)
      return rows
    },
  })

/* -------------------------------- Handlers -------------------------------- */
function onGripKey(group: SlotGroup, at: number, event: KeyboardEvent): void {
  const by = event.key === 'ArrowUp' ? -1 : event.key === 'ArrowDown' ? 1 : 0
  if (by === 0) return
  event.preventDefault()
  onChange(group, moveSlotRow(getRows(group), at, at + by))
}

function onReset(group: SlotGroup): void {
  emit('change', resetSlotRows(props.layout, group.slot))
}

function onChange(group: SlotGroup, rows: readonly SlotRow[]): void {
  emit('change', writeSlotRows(props.layout, group.slot, rows))
}

/* --------------------------------- Helpers -------------------------------- */
const getRows = (group: SlotGroup): readonly SlotRow[] => getSlotRows(group, props.layout)
</script>

<template>
  <IslandCard
    v-for="group in props.groups"
    :key="group.slot"
    :title="group.title"
    :data-testid="`slots-${group.slot}`"
  >
    <template #aside>
      <RowAction
        :label="words.slotsAddSeparator"
        class="plugin-slots__action"
        @click="onChange(group, addSlotSeparator(getRows(group)))"
      >
        <SeparatorHorizontal :size="14" aria-hidden="true" />
      </RowAction>
      <RowAction
        v-if="hasSlotLayout(props.layout, group.slot)"
        class="plugin-slots__action"
        :label="words.slotsReset(group.title)"
        @click="onReset(group)"
      >
        <RotateCcw :size="14" aria-hidden="true" />
      </RowAction>
    </template>
    <ul class="plugin-slots__list" :class="{ 'plugin-slots__list--pressed': isPressed(group) }">
      <li
        v-for="(row, at) in getRows(group)"
        :key="row.key"
        class="plugin-slots__row"
        :class="getRowClass(group, row.key)"
        :style="getRowStyle(group, row.key, at)"
        @transitionrun="(event) => onRowTransitionRun(event, row.key)"
        @transitionend="(event) => onRowTransitionEnd(event, row.key)"
      >
        <PluginSlotRow
          :row="row"
          :is-movable="getRows(group).length > 1"
          :place="getSeparatorPlace(getRows(group), at)"
          @grip-press="(event) => onGripPress(group, at, event)"
          @grip-key="(event) => onGripKey(group, at, event)"
          @toggle="onChange(group, toggleSlotRow(getRows(group), at))"
          @remove="onChange(group, removeSlotSeparator(getRows(group), at))"
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
  padding: 0.5rem 0;
  list-style: none;
}

.plugin-slots__list--pressed {
  user-select: none;
}

.plugin-slots__list--pressed .plugin-slots__row {
  will-change: transform;
}

.plugin-slots__action {
  margin-block: -0.25rem;
}

.plugin-slots__row {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  box-sizing: border-box;
  block-size: 1.75rem;
  padding-block: 0.125rem;
}

.plugin-slots__row--lifted,
.plugin-slots__row--settling {
  position: relative;
  z-index: 1;
  will-change: transform;
}

.plugin-slots__row--lifted {
  background: var(--numen-raised);
  box-shadow: 0 0.0625rem 0.1875rem color-mix(in srgb, var(--numen-raised) 40%, var(--numen-scrim));
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
