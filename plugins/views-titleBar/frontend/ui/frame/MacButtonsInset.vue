<script setup lang="ts">
/**
 * The room the system buttons of macOS take at the left of the window. In full screen macOS shows
 * no buttons, and the inset takes no room.
 */

/* --------------------------------- Props ---------------------------------- */
withDefaults(defineProps<{ isFullscreen?: boolean }>(), { isFullscreen: false })
</script>

<template>
  <div v-if="!isFullscreen" class="mac-buttons-inset" data-mac-buttons-inset />
</template>
