<script setup lang="ts">
/** The views of controls a plugin places at the top or the bottom of the board, in a row of their own; a place with no view takes no room. */
import type { PlexWidget } from '../types'

defineProps<{
  place: 'top' | 'bottom'
  entries: readonly PlexWidget[]
}>()
</script>

<template>
  <div v-if="entries.length > 0" class="plex-board" :class="`plex-board--${place}`">
    <component :is="entry.component" v-for="entry in entries" :key="entry.key" />
  </div>
</template>

<style scoped>
.plex-board {
  display: flex;
  flex: none;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
  padding: 0.25rem 0.5rem;
}
</style>
