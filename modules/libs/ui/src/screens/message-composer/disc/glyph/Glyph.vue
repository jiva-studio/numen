<script setup lang="ts">
/**
 * What stands on the disc: an arrow while it sends, a square while it stops.
 */
import type { ComposerAction } from '../../state'

defineProps<{
  /** What pressing the disc does. */
  action: ComposerAction
}>()
</script>

<template>
  <svg
    v-if="action === 'send'"
    viewBox="0 0 16 16"
    class="size-4"
    fill="none"
    stroke="currentColor"
    stroke-width="1.75"
    stroke-linecap="round"
    stroke-linejoin="round"
  >
    <path d="M8 13V3" />
    <path d="M3.5 7.5 8 3l4.5 4.5" />
  </svg>
  <svg v-else viewBox="0 0 16 16" class="size-4" fill="currentColor">
    <rect x="3" y="3" width="10" height="10" rx="2" />
  </svg>
</template>
