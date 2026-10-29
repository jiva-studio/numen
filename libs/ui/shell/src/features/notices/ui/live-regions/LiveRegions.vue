<script setup lang="ts">
/**
 * The two live regions that announce corner notices: one polite, and one
 * assertive that interrupts other announcements. Neither is visible.
 */
defineProps<{
  /** The text announced when the screen reader reaches the region. */
  told: string
  /** The text announced immediately. */
  cried: string
}>()
</script>

<template>
  <span class="sr-only" aria-live="polite">{{ told }}</span>
  <span class="sr-only" aria-live="assertive">{{ cried }}</span>
</template>
