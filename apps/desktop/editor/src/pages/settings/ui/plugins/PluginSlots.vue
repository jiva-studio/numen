<script setup lang="ts">
/**
 * The slots a plugin draws, each with the entries placed in it. A switch hides an entry, and the
 * arrows move it up or down; the plugin keeps the result in its setting of the type slots.
 */
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
import { WORDS as words } from '../../words'

const props = defineProps<{
  groups: readonly SlotGroup[]
  layout: JsonValue | undefined
}>()

const emit = defineEmits<{
  (e: 'change', layout: JsonValue): void
}>()

function onChange(group: SlotGroup, rows: readonly SlotRow[]): void {
  emit('change', writeSlotRows(props.layout, group.slot, rows))
}
</script>

<template>
  <section
    v-for="group in props.groups"
    :key="group.slot"
    class="plugin-slots"
    :data-testid="`slots-${group.slot}`"
  >
    <h3 class="plugin-slots__title">{{ group.title }}</h3>
    <p v-if="group.contributors.length === 0" class="plugin-slots__empty">
      {{ words.slotsEmpty }}
    </p>
    <ul v-else class="plugin-slots__list">
      <li
        v-for="(row, at) in getSlotRows(group, props.layout)"
        :key="row.key"
        class="plugin-slots__row"
      >
        <PluginSlotRow
          :row="row"
          :is-first="at === 0"
          :is-last="at === group.contributors.length - 1"
          @move="
            (by) => onChange(group, moveSlotRow(getSlotRows(group, props.layout), at, at + by))
          "
          @toggle="onChange(group, toggleSlotRow(getSlotRows(group, props.layout), at))"
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
  margin-block: 1rem;
}

.plugin-slots__title {
  margin: 0;
  font-size: var(--numen-text-2);
  font-weight: 600;
  color: var(--numen-ink);
}

.plugin-slots__empty {
  margin: 0;
  font-size: var(--numen-text-2);
  color: var(--numen-hushed);
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
</style>
