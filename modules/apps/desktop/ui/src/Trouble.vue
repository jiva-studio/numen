<script setup lang="ts">
/**
 * What the window says above the work: what it lost touch with, what it could
 * not do, and what it is waiting for.
 *
 * A vault that could not be read and a vault that holds nothing both leave the
 * window with nothing to show, and it says which of the two it is.
 */
import { WORDS as words } from './words'

const props = defineProps<{
  /** The vault is there, and the window is not being told when it changes. */
  unwatched: string
  /** The vault itself could not be read. */
  trouble: string
  /** What the window lost touch with, or what the tab in front cannot show. */
  warning: string
  /** What could not be made or joined, in words a person reads. */
  unmade: string
  /** The core could not be reached: nothing else in the window is true. */
  failure: string
  /** The vault is still being read for the first time. */
  indexing: boolean
  /** Whether the vault holds a note to show at all. */
  holds: boolean
}>()
</script>

<template>
  <p v-if="props.unwatched" class="warning">{{ words.unwatched }} — {{ props.unwatched }}</p>
  <p v-if="props.trouble" class="warning">{{ words.unread }} — {{ props.trouble }}</p>
  <p v-if="props.warning" class="warning">{{ props.warning }}</p>

  <p v-if="props.unmade" role="alert" class="warning">{{ props.unmade }}</p>

  <p v-if="props.failure" class="failure">{{ props.failure }}</p>
  <p v-else-if="props.indexing" class="waiting">{{ words.reading }}</p>
  <p v-else-if="!props.holds && props.trouble" class="waiting">{{ words.nothingRead }}</p>
</template>

<style scoped>
.waiting,
.failure {
  margin: auto;
  font-family: var(--numen-font-sans);
  font-size: 0.9rem;
  opacity: 0.6;
}

/* A warning and a failure carry filesystem paths, and a long one breaks where
   it stands. */
.warning {
  margin: 0;
  padding: 0.4rem 1rem;
  font-family: var(--numen-font-sans);
  font-size: 0.8rem;
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  overflow-wrap: break-word;
}

.failure {
  color: var(--numen-alarm);
  opacity: 1;
  max-width: 40rem;
  text-align: center;
  overflow-wrap: break-word;
}
</style>
