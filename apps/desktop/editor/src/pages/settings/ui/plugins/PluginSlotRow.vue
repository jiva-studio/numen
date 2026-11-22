<script setup lang="ts">
/** One entry of a slot: the grip that drags it where the slot holds more than one, its name, the plugin it comes from and the switch that shows it. */
import { GripVertical } from '@lucide/vue'
import { Switch } from '@numen/ui'
import type { EntryRow } from '../../lib/slotLayout'
import { WORDS as words } from '../../words'

const props = defineProps<{
  row: EntryRow
  isMovable: boolean
}>()

const emit = defineEmits<{
  (e: 'grip-press', event: PointerEvent): void
  (e: 'grip-key', event: KeyboardEvent): void
  (e: 'toggle'): void
}>()
</script>

<template>
  <button
    type="button"
    class="plugin-slots__grip"
    :disabled="!props.isMovable"
    :aria-label="words.slotsGrip(props.row.label)"
    @pointerdown="emit('grip-press', $event)"
    @keydown="emit('grip-key', $event)"
  >
    <GripVertical class="plugin-slots__grip-icon" aria-hidden="true" />
  </button>
  <span class="plugin-slots__name">
    <span class="plugin-slots__label">{{ props.row.label }}</span>
    <span class="plugin-slots__owner" aria-hidden="true">·</span>
    <span class="plugin-slots__owner">{{ props.row.pluginName }}</span>
  </span>
  <Switch
    :model-value="!props.row.isHidden"
    :aria-label="words.slotsShow(props.row.label)"
    @update:model-value="emit('toggle')"
  />
</template>

<style scoped>
.plugin-slots__grip {
  --icon: 0.75rem;
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
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
  opacity: 0.35;
  cursor: default;
}

.plugin-slots__grip:focus-visible {
  outline: none;
}

.plugin-slots__grip-icon {
  inline-size: var(--icon);
  block-size: var(--icon);
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
