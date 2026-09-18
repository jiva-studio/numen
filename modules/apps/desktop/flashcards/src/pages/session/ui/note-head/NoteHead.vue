<script setup lang="ts">
/**
 * What one note is called where it is read, and what is said quietly under it.
 */
import { WORDS as words } from '../../lib/notesWords'
import type { Neighbour } from '../../api/notes'

const props = defineProps<{ note: Neighbour }>()

/** A note is named by its title, and by how it was written where it has none. */
const getNoteName = (one: Neighbour) => one.title || one.written
</script>

<template>
  <header class="reading__head">
    <h2 class="reading__name">{{ getNoteName(props.note) }}</h2>
    <!-- The path is here because two notes can be called the same thing. -->
    <p class="reading__quiet">
      <span v-if="props.note.path">{{ props.note.path }}</span>
      <span v-if="!props.note.hasPoints">{{ words.pointsHere }}</span>
      <span v-if="props.note.label">{{ props.note.label }}</span>
    </p>
  </header>
</template>

<style scoped>
/* The name, ruled off from the prose under it. */
.reading__head {
  margin-block-end: var(--numen-inset);
  padding-block-end: var(--numen-inset);
  border-block-end: 1px solid var(--numen-rule);
}

.reading__name {
  margin: 0;
  overflow-wrap: anywhere;
  font-size: var(--numen-title-size);
}

.reading__quiet {
  margin: 0.25rem 0 0;
  overflow-wrap: anywhere;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

/* Two things said quietly about one note stand apart on the same line. */
.reading__quiet > span + span {
  margin-inline-start: var(--numen-inset);
}
</style>
