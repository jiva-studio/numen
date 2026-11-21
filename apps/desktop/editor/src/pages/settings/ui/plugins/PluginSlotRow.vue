<script setup lang="ts">
/** One entry of a slot: its name and the plugin it comes from, the arrows that move it, and the switch that shows it. */
import { ChevronDown, ChevronUp } from '@lucide/vue'
import { Button, Switch } from '@numen/ui'
import type { SlotRow } from '../../lib/slotLayout'
import { WORDS as words } from '../../words'

const props = defineProps<{
  row: SlotRow
  canMoveUp: boolean
  canMoveDown: boolean
}>()

const emit = defineEmits<{
  (e: 'move', by: number): void
  (e: 'toggle'): void
}>()
</script>

<template>
  <span class="plugin-slots__label">{{ props.row.label }}</span>
  <span class="plugin-slots__owner">{{ props.row.pluginName }}</span>
  <Button
    variant="ghost"
    size="icon"
    :disabled="!props.canMoveUp"
    :class="{ 'plugin-slots__arrow--off': !props.canMoveUp }"
    :tabindex="props.canMoveUp ? 0 : -1"
    :aria-label="words.slotsMoveUp(props.row.label)"
    @click="emit('move', -1)"
  >
    <ChevronUp :size="14" aria-hidden="true" />
  </Button>
  <Button
    variant="ghost"
    size="icon"
    :disabled="!props.canMoveDown"
    :class="{ 'plugin-slots__arrow--off': !props.canMoveDown }"
    :tabindex="props.canMoveDown ? 0 : -1"
    :aria-label="words.slotsMoveDown(props.row.label)"
    @click="emit('move', 1)"
  >
    <ChevronDown :size="14" aria-hidden="true" />
  </Button>
  <Switch
    :model-value="!props.row.isHidden"
    :aria-label="words.slotsShow(props.row.label)"
    @update:model-value="emit('toggle')"
  />
</template>

<style scoped>
.plugin-slots__label {
  flex: 1 1 auto;
  min-inline-size: 0;
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
}

.plugin-slots__arrow--off {
  visibility: hidden;
}

.plugin-slots__owner {
  font-size: var(--numen-text-1);
  color: var(--numen-hushed);
}
</style>
