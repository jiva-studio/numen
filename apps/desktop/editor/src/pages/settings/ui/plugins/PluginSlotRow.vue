<script setup lang="ts">
/**
 * One row of a slot. An entry has the grip that drags it where the slot holds more than one row, its name,
 * the plugin it comes from and the switch that shows it; a separator has its grip, the line and a button
 * that removes it, centred in the cell the switch fills.
 */
import { GripVertical, X } from '@lucide/vue'
import { Switch } from '@numen/ui'
import type { SlotRow } from '../../lib/slotLayout'
import RowAction from '../setting-row/RowAction.vue'
import { WORDS as words } from '../../words'

const props = defineProps<{
  row: SlotRow
  isMovable: boolean
  place: { readonly position: number; readonly total: number }
}>()

const emit = defineEmits<{
  (e: 'grip-press', event: PointerEvent): void
  (e: 'grip-key', event: KeyboardEvent): void
  (e: 'toggle'): void
  (e: 'remove'): void
}>()
</script>

<template>
  <button
    type="button"
    class="plugin-slots__grip"
    :disabled="!props.isMovable"
    :aria-label="
      props.row.kind === 'entry'
        ? words.slotsGrip(props.row.label)
        : words.slotsSeparatorGrip(props.place.position, props.place.total)
    "
    aria-keyshortcuts="ArrowUp ArrowDown"
    @pointerdown="emit('grip-press', $event)"
    @keydown="emit('grip-key', $event)"
  >
    <GripVertical class="plugin-slots__grip-icon" aria-hidden="true" />
  </button>
  <template v-if="props.row.kind === 'entry'">
    <span class="plugin-slots__name">
      <span class="plugin-slots__label">{{ props.row.label }}</span>
      <span class="plugin-slots__owner" aria-hidden="true">·</span>
      <span class="plugin-slots__owner">{{ props.row.pluginName }}</span>
    </span>
    <span class="plugin-slots__trail" data-testid="slot-trail">
      <Switch
        :model-value="!props.row.isHidden"
        :aria-label="words.slotsShow(props.row.label)"
        @update:model-value="emit('toggle')"
      />
    </span>
  </template>
  <template v-else>
    <hr class="plugin-slots__separator" />
    <span class="plugin-slots__trail" data-testid="slot-trail">
      <RowAction
        :label="words.slotsRemoveSeparator(props.place.position, props.place.total)"
        @click="emit('remove')"
      >
        <X :size="14" aria-hidden="true" />
      </RowAction>
    </span>
  </template>
</template>

<style scoped>
.plugin-slots__grip {
  --icon: 0.75rem;
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: flex-start;
  inline-size: 1rem;
  block-size: 1.5rem;
  padding: 0;
  border: none;
  border-radius: var(--numen-radius-tight);
  background: none;
  color: var(--numen-hushed);
  cursor: grab;
  touch-action: none;
}

.plugin-slots__grip:disabled {
  color: var(--numen-edge);
  cursor: default;
}

.plugin-slots__grip:focus-visible {
  outline: none;
}

.plugin-slots__grip-icon {
  inline-size: var(--icon);
  block-size: var(--icon);
}

.plugin-slots__separator {
  flex: 1 1 auto;
  margin: 0;
  border: none;
  border-block-start: var(--numen-stroke) solid var(--numen-rule);
}

.plugin-slots__trail {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  inline-size: 2.25rem;
}

.plugin-slots__name {
  display: flex;
  flex: 1 1 auto;
  align-items: baseline;
  gap: 0.375rem;
  min-inline-size: 0;
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
}

.plugin-slots__owner {
  font-size: var(--numen-text-1);
  color: var(--numen-hushed);
}
</style>
