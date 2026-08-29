<script setup lang="ts">
/**
 * A floating panel: rounded on every side, with what it covers showing
 * through it.
 *
 * A surface to put things on. It stacks what it is given in a column and
 * keeps them clear of its edges. Anything that scrolls scrolls itself, and
 * where the panel sits on the screen belongs to whatever places it.
 */
</script>

<template>
  <div
    class="numen flex min-h-0 flex-col gap-gap overflow-hidden rounded-panel border border-panel-rule bg-panel p-panel shadow-panel backdrop-blur-panel font-sans text-base text-ink"
  >
    <slot />
  </div>
</template>
