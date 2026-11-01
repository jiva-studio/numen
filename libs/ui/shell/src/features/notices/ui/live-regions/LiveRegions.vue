<script setup lang="ts">
/**
 * The two live regions that announce corner notices: one polite, and one
 * assertive that interrupts other announcements. Neither is visible.
 */
defineProps<{
  /** The text announced politely, when the screen reader is idle. */
  polite: string
  /** The text announced immediately. */
  assertive: string
}>()
</script>

<template>
  <span class="sr-only" aria-live="polite">{{ polite }}</span>
  <span class="sr-only" aria-live="assertive">{{ assertive }}</span>
</template>
