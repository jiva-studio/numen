<script setup lang="ts">
/** A spinner and its label, drawn while a pane is loading its content. */
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { Spinner } from '../spinner'

const props = withDefaults(defineProps<{ label?: string; delay?: number }>(), {
  label: 'Loading…',
  delay: 400,
})

// What arrives within the delay replaces an empty pane, so no ring flashes before it.
const isShown = ref(props.delay <= 0)
let timer: ReturnType<typeof setTimeout> | undefined
onMounted(() => {
  if (!isShown.value) timer = setTimeout(() => (isShown.value = true), props.delay)
})
onBeforeUnmount(() => clearTimeout(timer))
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
