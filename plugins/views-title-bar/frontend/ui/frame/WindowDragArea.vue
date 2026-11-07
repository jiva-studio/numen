<script setup lang="ts">
/**
 * An area that moves the window when it is pressed and dragged, its contents included. A button,
 * a field or a link inside it carries `--wails-draggable: no-drag` to take the press itself.
 */
</script>

<template>
  <div class="window-drag-area" data-window-drag style="--wails-draggable: drag">
    <slot />
  </div>
</template>
