<script setup lang="ts">
/**
 * A placeholder panel for whatever a tab holds.
 *
 * The workspace has no knowledge of its contents, and neither does this: it
 * shows a name and fills the space it is given.
 */
defineProps<{ name: string }>()
</script>

<template>
  <div
    class="tab-stub numen bg-surface text-hushed grid h-full place-items-center font-sans text-base"
  >
    {{ name }}
  </div>
</template>
