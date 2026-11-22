<script setup lang="ts">
/** A small round icon button of a settings row or an island heading: the icon is its content, its label is its name and its tooltip. */
import { Tooltip, useTooltip } from '@numen/ui'

const props = defineProps<{ label: string }>()

const tip = useTooltip()
</script>

<template>
  <button
    type="button"
    class="row-action"
    :aria-label="props.label"
    :aria-describedby="tip.describedBy.value"
    @mouseenter="tip.open"
    @mouseleave="tip.close"
    @focus="tip.open"
    @blur="tip.close"
  >
    <slot />
    <Teleport to="body">
      <Tooltip v-if="tip.at.value" :id="tip.id" :at="tip.at.value">{{ props.label }}</Tooltip>
    </Teleport>
  </button>
</template>

<style scoped>
.row-action {
  display: inline-grid;
  place-items: center;
  flex: none;
  inline-size: 1.5rem;
  block-size: 1.5rem;
  padding: 0;
  border: none;
  border-radius: var(--numen-radius-tight);
  background: none;
  color: var(--numen-hushed);
  cursor: pointer;
}

.row-action:hover {
  background-color: var(--settings-hover-bg);
  color: var(--numen-ink);
}
</style>
