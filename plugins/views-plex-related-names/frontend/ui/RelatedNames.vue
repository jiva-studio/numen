<script setup lang="ts">
/** The names set in one row group beside a node, in small faint text with no pill. Choosing one is the caller's. */
import type { RelatedName } from '../types'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ names: readonly RelatedName[]; showArrows?: boolean }>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** A name was chosen, with the path of its note. */
  (event: 'choose', path: string): void
}>()
</script>

<template>
  <div v-if="names.length > 0" class="related-names">
    <button
      v-for="name in names"
      :key="name.path"
      type="button"
      class="related-names__name"
      :aria-label="name.label"
      @click="emit('choose', name.path)"
    >
      {{ name.title }}
    </button>
  </div>
</template>

<style scoped>
.related-names {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 0 0.9em;
  font-family: var(--numen-font-sans);
  font-size: 0.7rem;
  line-height: 1.3;
}

.related-names__name {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--numen-hushed);
  font: inherit;
  cursor: pointer;
}

.related-names__name:hover {
  color: var(--numen-ink);
}
</style>
