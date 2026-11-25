<script setup lang="ts">
/**
 * The ring of a transfer, drawn over the box of its parent: an arc by the share done, or, when the total is unknown, an arc that orbits. The glyph
 * in the slot is drawn inside the ring.
 */
import { computed } from 'vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ progress: number | null }>()

/* --------------------------------- State ---------------------------------- */
const RING_RADIUS = 20
const RING_LENGTH = 2 * Math.PI * RING_RADIUS
const ORBIT_SHARE = 0.25
const isIndeterminate = computed(() => props.progress === null)
const fraction = computed(() => Math.min(Math.max(props.progress ?? 0, 0), 1))
const arc = computed(() => (isIndeterminate.value ? ORBIT_SHARE : fraction.value) * RING_LENGTH)
</script>

<template>
  <svg
    class="sync-ring"
    :class="{ 'sync-ring--indeterminate': isIndeterminate }"
    viewBox="0 0 44 44"
    aria-hidden="true"
  >
    <circle class="sync-ring__track" cx="22" cy="22" :r="RING_RADIUS" />
    <circle
      class="sync-ring__arc"
      cx="22"
      cy="22"
      :r="RING_RADIUS"
      :stroke-dasharray="`${arc} ${RING_LENGTH}`"
    />
  </svg>
  <slot />
</template>
