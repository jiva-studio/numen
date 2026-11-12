<script setup lang="ts">
/**
 * The title of one note where it is read, and the secondary text under it.
 */
import { WORDS as words } from '../../../lib/session/notesWords'
import type { Neighbor } from '../../../api/session/notes'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ note: Neighbor }>()

/* -------------------------------- Helpers --------------------------------- */
const getNoteName = (one: Neighbor) => one.title || one.written
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
  --quiet-margin: 0.25rem;
  margin: var(--quiet-margin) 0 0;
  overflow-wrap: anywhere;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

/* Two secondary items of one note sit apart on the same line. */
.reading__quiet > span + span {
  margin-inline-start: var(--numen-inset);
}
</style>
