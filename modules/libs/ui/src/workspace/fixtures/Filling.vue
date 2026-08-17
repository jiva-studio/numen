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
  <div class="filling numen grid h-full place-items-center bg-surface font-sans text-base text-hushed">
    {{ name }}
  </div>
</template>
