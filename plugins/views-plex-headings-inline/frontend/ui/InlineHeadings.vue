<script setup lang="ts">
/** The headings of a note set in one row group under a node, each as `# Heading` in small faint text. Choosing one is the caller's. */
import type { NoteHeading } from '@numen/plugin-sdk'

defineProps<{ headings: readonly NoteHeading[] }>()

const emit = defineEmits<{
  /** A heading was chosen, with the line of the note it is on. */
  (event: 'choose', line: number): void
}>()
</script>

<template>
  <div v-if="headings.length > 0" class="inline-headings">
    <button
      v-for="heading in headings"
      :key="heading.line"
      type="button"
      class="inline-headings__heading"
      @click="emit('choose', heading.line)"
    >
      # {{ heading.text }}
    </button>
  </div>
</template>

<style scoped>
.inline-headings {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-block-start: var(--numen-node-gap);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-1);
  line-height: 1.3;
}

.inline-headings__heading {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--numen-hushed);
  font: inherit;
  cursor: pointer;
}

.inline-headings__heading:hover {
  color: var(--numen-ink);
}
</style>
