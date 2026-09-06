<script setup lang="ts">
/**
 * What a book divides into, as a list to reach any of it by.
 *
 * Every line stands at a byte offset into the book's text, and choosing one
 * asks for that offset. The line the person is reading is marked, and the list
 * is brought to it as the reading moves.
 */
import { computed, ref, useTemplateRef, watch } from 'vue'
import {
  CONTENTS_WORDS,
  matching,
  standingIn,
  type ContentsEntry,
  type ContentsWords,
} from './contents'

const props = withDefaults(
  defineProps<{
    /** What the book divides into, ascending by offset. */
    entries?: readonly ContentsEntry[]
    /** Where the person is reading, in bytes of the book's text. */
    at?: number
    /** The words it is read with. */
    words?: ContentsWords
  }>(),
  {
    entries: () => [],
    at: 0,
    words: () => CONTENTS_WORDS,
  },
)

const emit = defineEmits<{
  /** The offset the person chose, in bytes of the book's text. */
  (event: 'go', at: number): void
}>()

/** What is typed into the field, which the list is narrowed by. */
const typed = ref('')

/** The lines as they are drawn, which the list is brought along by. */
const lines = useTemplateRef<HTMLElement[]>('lines')

const shown = computed(() => matching(props.entries, typed.value))

/** The line the person is reading, and none where they stand before the first. */
const standing = computed(() => props.entries[standingIn(props.entries, props.at)])

/**
 * How deep a line is set in. Past the fourth level it stops: a name set any
 * further in is a column of one word.
 */
const DEEPEST = 4

const depth = (level: number) => Math.min(Math.max(level, 0), DEEPEST)

// The list follows the reading, so a person turning pages finds where they are
// without looking for it.
watch(standing, (one) => {
  if (!one || typed.value !== '') return
  const line = lines.value?.find((drawn) => drawn.dataset['at'] === String(one.at))
  line?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
})
</script>

<template>
  <nav class="contents" :aria-label="words.contents">
    <div class="contents__head">
      <input
        v-model="typed"
        class="contents__field"
        type="text"
        autocomplete="off"
        spellcheck="false"
        :aria-label="words.find"
        :placeholder="words.find"
      />
    </div>

    <ol v-if="shown.length !== 0" class="contents__list">
      <li v-for="one in shown" :key="one.at">
        <button
          ref="lines"
          type="button"
          class="contents__line"
          :data-at="one.at"
          :style="{ '--level': depth(one.level) }"
          :aria-current="one.at === standing?.at ? 'true' : undefined"
          @click="emit('go', one.at)"
        >
          {{ one.title }}
        </button>
      </li>
    </ol>

    <p v-else class="contents__silence">{{ words.nothing }}</p>
  </nav>
</template>

<style scoped>
.contents {
  /* How far one level is set in, and the room at the edges of a line. */
  --indent: 0.875rem;
  --pad: 0.5rem;

  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  font-size: var(--numen-text-1);
}

.contents__head {
  flex: 0 0 auto;
  padding: var(--pad);
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

.contents__field {
  inline-size: 100%;
  padding: 0.25rem 0.5rem;
  border-radius: var(--numen-radius-tight);
  background: var(--numen-bubble-bg);
  color: inherit;
}

.contents__field:focus-visible {
  outline: none;
  box-shadow: inset 0 0 0 var(--numen-ring-width) var(--numen-ring);
}

.contents__list {
  flex: 1 1 auto;
  min-block-size: 0;
  overflow-y: auto;
  padding-block: 0.25rem;
}

.contents__line {
  display: block;
  inline-size: 100%;
  padding-inline: calc(var(--pad) + var(--indent) * var(--level)) var(--pad);
  padding-block: 0.125rem;
  text-align: start;
  /* A name too long for the panel is cut, and the whole of it is in the book. */
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.contents__line:hover {
  background: var(--numen-bubble-bg);
}

.contents__line[aria-current='true'] {
  background: var(--numen-accent);
  color: var(--numen-accent-ink);
}

.contents__line:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
  outline-offset: calc(-1 * var(--numen-ring-width));
}

.contents__silence {
  padding: var(--pad);
  color: var(--numen-hushed);
}
</style>
