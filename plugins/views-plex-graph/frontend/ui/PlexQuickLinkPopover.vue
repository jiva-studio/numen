<script setup lang="ts">
/**
 * Native Plex-style thought creation combobox: the input and its options in one frame.
 */
import { computed, ref, watch } from 'vue'
import { SearchableList, type ListItem } from '@numen/ui'
import { useSearchResults } from '../model/useSearchResults'
import { WORDS as words } from '../words'
import type { PlexRelation } from '../types'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  at: { readonly x: number; readonly y: number }
  relation: PlexRelation
  search: (query: string) => Promise<readonly { path: string; title: string }[]>
}>()

const emit = defineEmits<{
  (event: 'select-note', path: string): void
  (event: 'create-note', title: string): void
  (event: 'dismiss'): void
}>()

/* --------------------------------- State ---------------------------------- */
const query = ref('')
const { results, isLoading, setQuery } = useSearchResults((text) => props.search(text))

/** The notes found, each named by its path and written by its title. */
const items = computed<readonly ListItem[]>(() =>
  results.value.map((note) => ({ id: note.path, text: note.title || note.path })),
)

const positionStyle = computed(() => ({
  left: `${Math.round(props.at.x)}px`,
  top: `${Math.round(props.at.y)}px`,
  '--relation-color': `var(--numen-relation-${props.relation}, var(--numen-accent))`,
}))

/* --------------------------------- Hooks ---------------------------------- */
watch(query, (text) => setQuery(text))
</script>

<template>
  <div
    class="plex-quick-link"
    :class="`plex-quick-link--${relation}`"
    :style="positionStyle"
    role="dialog"
    aria-label="New thought"
  >
    <SearchableList
      v-model:query="query"
      class="plex-quick-link__frame"
      :items="items"
      :label="words.quickLinkLabel"
      :placeholder="words.quickLinkPlaceholder"
      :is-filtered="false"
      :is-pending="isLoading"
      has-action
      needs-query
      @select="(path) => emit('select-note', path)"
      @act="(title) => emit('create-note', title)"
      @close="emit('dismiss')"
    >
      <template #action="{ query: typed }">{{ words.createNote(typed) }}</template>
    </SearchableList>
  </div>
</template>

<style scoped>
.plex-quick-link {
  position: absolute;
  z-index: 20;
  inline-size: 0;
  block-size: 0;
  pointer-events: auto;
  font-family: var(--numen-font-sans);
  --plex-quick-link-width: 14rem;
}

/* The frame is centered on the point of the node it was asked from, by the middle of its search line. */
.plex-quick-link :deep(.plex-quick-link__frame) {
  position: absolute;
  margin-inline-start: calc(var(--plex-quick-link-width) / -2);
  margin-block-start: calc(var(--numen-action-size) / -2);
  inline-size: var(--plex-quick-link-width);
}

/* The frame takes the color of the relation it makes. */
.plex-quick-link :deep(.plex-quick-link__frame) {
  border-color: var(--relation-color);
}
</style>
