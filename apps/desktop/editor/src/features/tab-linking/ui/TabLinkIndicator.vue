<script setup lang="ts">
/**
 * Visual indicator for a linked tab, displaying the link icon and group color badge.
 */
import { computed } from 'vue'
import { Link2 } from '@lucide/vue'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    isLinked?: boolean
    colorIndex?: number | undefined
  }>(),
  { isLinked: false, colorIndex: 1 },
)

/* --------------------------------- State ---------------------------------- */
const LINK_COLORS: Readonly<Record<number, string>> = {
  1: 'var(--numen-blue, #3b82f6)',
  2: 'var(--numen-leaf, #22c55e)',
  3: 'var(--numen-amber, #f59e0b)',
  4: 'var(--numen-purple, #a855f7)',
}

const linkColor = computed(() => LINK_COLORS[props.colorIndex ?? 1] ?? 'var(--numen-blue, #3b82f6)')

const iconStyle = computed(() => ({
  color: linkColor.value,
}))
</script>

<template>
  <span v-if="props.isLinked" class="tab-link-indicator" title="Linked tab" aria-label="Linked tab">
    <Link2 class="tab-link-indicator__icon" :style="iconStyle" aria-hidden="true" />
  </span>
</template>

<style scoped>
.tab-link-indicator {
  display: inline-flex;
  align-items: center;
  padding-inline-start: 2px;
  user-select: none;
  -webkit-user-select: none;
}

.tab-link-indicator__icon {
  inline-size: 0.85rem;
  block-size: 0.85rem;
  stroke-width: 2.25;
}
</style>
