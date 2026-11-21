<script setup lang="ts">
/** One entry of a slot: the grip that drags it where the slot holds more than one, its name, the plugin it comes from and the switch that shows it. */
import { GripVertical } from '@lucide/vue'
import { Switch } from '@numen/ui'
import type { SlotRow } from '../../lib/slotLayout'
import { WORDS as words } from '../../words'

const props = defineProps<{
  row: SlotRow
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
    v-if="props.isMovable"
    type="button"
    class="plugin-slots__grip"
    :aria-label="words.slotsGrip(props.row.label)"
    @pointerdown="emit('grip-press', $event)"
    @keydown="emit('grip-key', $event)"
  >
    <GripVertical :size="14" aria-hidden="true" />
  </button>
  <span class="plugin-slots__label">{{ props.row.label }}</span>
  <span class="plugin-slots__owner">{{ props.row.pluginName }}</span>
  <Switch
    :model-value="!props.row.isHidden"
    :aria-label="words.slotsShow(props.row.label)"
    @update:model-value="emit('toggle')"
  />
</template>

<style scoped>
.plugin-slots__grip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  inline-size: 1.5rem;
  block-size: 2rem;
  padding: 0;
  border: none;
  border-radius: var(--numen-radius);
  background: none;
  color: var(--numen-hushed);
  cursor: grab;
  touch-action: none;
}

.plugin-slots__grip:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.plugin-slots__label {
  flex: 1 1 auto;
  min-inline-size: 0;
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
}

.plugin-slots__owner {
  font-size: var(--numen-text-1);
  color: var(--numen-hushed);
}
</style>
