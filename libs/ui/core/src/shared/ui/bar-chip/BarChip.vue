<script setup lang="ts">
/**
 * The chip of the title bar and the status bar: a button with a face, and a popover that opens
 * below it, or above it where there is no room below.
 *
 * Without a `popover` slot a press opens nothing.
 */
import { useId, useSlots } from 'vue'
import { PopoverContent, PopoverPortal, PopoverRoot, PopoverTrigger } from 'reka-ui'
import type { BarChipTone } from './barChip.types'
import { useDialogName } from './useDialogName'
import { useSideOffset } from './useSideOffset'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    label: string
    tooltip?: string | undefined
    tone?: BarChipTone
    popoverLabel?: string | undefined
  }>(),
  { tooltip: undefined, tone: 'neutral', popoverLabel: undefined },
)

/* --------------------------------- State ---------------------------------- */
const isOpen = defineModel<boolean>('open', { default: false })
const slots = useSlots()
const chipId = useId()
const sideOffset = useSideOffset(isOpen)

/* --------------------------------- Hooks ---------------------------------- */
useDialogName(chipId, () => isOpen.value && hasPopover() && props.popoverLabel !== undefined)

/* -------------------------------- Handlers -------------------------------- */
function onUpdateOpen(value: boolean): void {
  if (value && !hasPopover()) return
  isOpen.value = value
}

/* -------------------------------- Helpers --------------------------------- */
function hasPopover(): boolean {
  return slots.popover !== undefined
}
</script>

<template>
  <PopoverRoot :open="isOpen && hasPopover()" @update:open="onUpdateOpen">
    <PopoverTrigger as-child>
      <button
        type="button"
        tabindex="-1"
        class="bar-chip"
        :class="`bar-chip--${props.tone}`"
        :aria-label="props.label"
        :title="props.tooltip"
        aria-haspopup="dialog"
        :aria-expanded="isOpen && hasPopover()"
      >
        <slot />
      </button>
    </PopoverTrigger>
    <PopoverPortal v-if="hasPopover()">
      <PopoverContent
        class="bar-chip__popover"
        side="bottom"
        align="end"
        :side-offset="sideOffset"
        avoid-collisions
        :aria-label="props.popoverLabel"
        :data-bar-chip-popover="chipId"
      >
        <slot name="popover" />
      </PopoverContent>
    </PopoverPortal>
  </PopoverRoot>
</template>

<style scoped>
.bar-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: calc(var(--numen-chip-inset) / 2);
  box-sizing: border-box;
  height: var(--numen-chip-height);
  padding: 0 var(--numen-chip-inset);
  border: none;
  border-radius: var(--numen-radius);
  outline: none;
  font: inherit;
  font-size: var(--numen-chip-text);
  font-variant-numeric: tabular-nums;
  font-weight: 500;
  line-height: 1;
  white-space: nowrap;
  cursor: pointer;
}

/* WebKit sends no click when the node under a press is replaced before the release. */
.bar-chip > :deep(*) {
  pointer-events: none;
}

.bar-chip--neutral {
  background: var(--numen-surface);
  color: var(--numen-ink);
}

.bar-chip--neutral:hover,
.bar-chip--neutral[aria-expanded='true'] {
  background: color-mix(in oklab, var(--numen-surface), var(--numen-ink) 10%);
}

.bar-chip--accent {
  background: var(--numen-accent);
  color: var(--numen-accent-ink);
}

.bar-chip--accent:hover,
.bar-chip--accent[aria-expanded='true'] {
  filter: brightness(1.08);
}

.bar-chip--alarm {
  background: var(--numen-alarm-bg);
  color: var(--numen-alarm);
}
</style>

<style>
.bar-chip__popover {
  box-sizing: border-box;
  max-width: calc(100vw - 1rem);
  background: var(--numen-raised);
  color: var(--numen-ink);
  border: 1px solid var(--numen-panel-border);
  border-radius: var(--numen-radius-popup);
  box-shadow: var(--numen-panel-shadow);
  overflow: hidden;
  z-index: var(--numen-lift-menu);
}
</style>
