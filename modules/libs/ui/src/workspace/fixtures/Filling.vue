<script setup lang="ts">
/**
 * A plain panel standing in for whatever a tab holds.
 *
 * The workspace is told nothing about its contents, and neither is this: it
 * shows a name and fills the room it is given.
 */
defineProps<{ name: string }>()
</script>

<template>
  <div class="filling numen grid h-full place-items-center font-sans text-base text-hushed">
    {{ name }}
  </div>
</template>

<style scoped>
.filling {
  background: repeating-linear-gradient(
    135deg,
    transparent 0 12px,
    color-mix(in oklab, var(--numen-node-fg) 4%, transparent) 12px 24px
  );
}
</style>
