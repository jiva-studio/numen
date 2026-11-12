<script setup lang="ts">
/** A spinner and its label, drawn while a pane is loading its content. */
import { useDelayedFlag } from '@/shared/model/useDelayedFlag'
import { Spinner } from '../spinner'

const props = withDefaults(defineProps<{ label?: string; delay?: number }>(), {
  label: 'Loading…',
  delay: 400,
})

// What arrives within the delay replaces an empty pane, so no ring flashes before it.
const isShown = useDelayedFlag(() => props.delay)
</script>

<template>
  <div
    class="loading-indicator numen text-small text-hushed flex h-full w-full items-center justify-center gap-2 font-sans"
    role="status"
    aria-busy="true"
    :data-shown="isShown"
  >
    <template v-if="isShown">
      <Spinner class="loading-indicator__ring" />
      <span>{{ props.label }}</span>
    </template>
  </div>
</template>

<style scoped>
/* The ring relies on the line of the word beside it. */
.loading-indicator__ring {
  --spinner-size: 1em;
}
</style>
