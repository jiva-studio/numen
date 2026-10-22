<script setup lang="ts">
/**
 * and the word for what is happening beside it.
 *
 * is waiting says so at one size, in one colour, in one word.
 */
import { Spinner } from '../spinner'

const props = withDefaults(defineProps<{ label?: string }>(), { label: 'Loading…' })
</script>

<template>
  <div
    class="waiting numen text-small text-hushed flex h-full w-full items-center justify-center gap-2 font-sans"
    role="status"
  >
    <Spinner class="waiting__ring" />
    <span>{{ props.label }}</span>
  </div>
</template>

<style scoped>
/* The ring relies on the line of the word beside it. */
.waiting__ring {
  --spinner-size: 1em;
}
</style>
