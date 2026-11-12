<script setup lang="ts">
/**
 * The sections of a book as a list for navigating to any of them.
 *
 * Choosing an entry emits its offset. The entry containing the reading position
 * is marked, and the list scrolls to it as the reading position changes.
 */
import { computed } from 'vue'
import { SearchableList, type ListItem } from '@numen/ui'
import { CONTENTS_WORDS, findLineAt, type ContentsWords } from './contents'
import type { ContentsEntry } from '../../../types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    entries?: readonly ContentsEntry[]
    at?: number
    words?: ContentsWords
  }>(),
  {
    entries: () => [],
    at: 0,
    words: () => CONTENTS_WORDS,
  },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** The offset of the chosen entry, in bytes of the book's text. */
  (event: 'go', pos: number): void
  /** The list asks to be put away. */
  (event: 'close'): void
}>()

/* --------------------------------- State ---------------------------------- */
/** How many lines show before the list scrolls. */
const VISIBLE_LINES = 10

/**
 * How deep a line is set in. Past the fourth level it stops: a name set any
 * further in is a column of one word.
 */
const DEEPEST = 4

/** A line is named by its place in the book, which no other line shares. */
const items = computed<readonly ListItem[]>(() =>
  props.entries.map((one, index) => ({ id: String(index), text: one.title })),
)

/** The line containing the reading position, and none before the first entry. */
const current = computed(() => {
  const index = findLineAt(props.entries, props.at)
  return index < 0 ? null : String(index)
})

/* -------------------------------- Handlers -------------------------------- */
function onSelect(id: string) {
  const entry = props.entries[Number(id)]
  if (entry) emit('go', entry.at)
}

/* -------------------------------- Helpers --------------------------------- */
const getLevel = (id: string): number =>
  Math.min(Math.max(props.entries[Number(id)]?.level ?? 0, 0), DEEPEST)
</script>

<template>
  <SearchableList
    :items="items"
    :model-value="current"
    :label="words.contents"
    :placeholder="words.find"
    :empty-text="words.nothing"
    :max-rows="VISIBLE_LINES"
    @select="onSelect"
    @close="emit('close')"
  >
    <template #leading="{ item }">
      <span
        v-if="getLevel(item.id) > 0"
        class="contents__indent"
        :style="{ '--level': getLevel(item.id) }"
        aria-hidden="true"
      />
    </template>
  </SearchableList>
</template>

<style scoped>
/* The indentation of one level. */
.contents__indent {
  --indent: 0.875rem;

  flex: none;
  inline-size: calc(var(--indent) * var(--level));
}
</style>
