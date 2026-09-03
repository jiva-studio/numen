<script setup lang="ts">
/**
 * Marked-up prose, read as it is meant to be read.
 *
 * The marks are read by markdown-it and drawn as Vue's own nodes, so what is
 * on the screen stays on the screen while the rest of it arrives. Typesetting
 * is the typography plugin's.
 */
import { computed } from 'vue'
import { pointsAtNote } from '../linking/address'
import { render } from './render'

const props = withDefaults(
  defineProps<{
    /** Markdown, whole or as far as it has arrived. */
    text: string
    /** Still being written: a word that has just arrived is shown arriving. */
    arriving?: boolean
    /**
     * The addresses this text points at that reach nothing. A link carrying
     * one is drawn as not resolving.
     */
    unresolved?: readonly string[]
  }>(),
  { arriving: false, unresolved: () => [] },
)

const emit = defineEmits<{
  /**
   * A link in the prose was pressed, with what it points at and the press
   * itself. Nothing here follows it: what a link means is the caller's, and so
   * is whether the browser should go there.
   */
  (event: 'follow', href: string, press: MouseEvent): void
}>()

const drawn = computed(() => render(props.text, new Set(props.unresolved)))
const Drawn = () => drawn.value

const pressed = (press: MouseEvent) => {
  const link = (press.target as HTMLElement | null)?.closest?.('a')
  const href = link?.getAttribute('href')
  if (!href) return
  // A note is addressed and not located, so a browser has nowhere to take one.
  if (pointsAtNote(href)) press.preventDefault()
  emit('follow', href, press)
}
</script>

<template>
  <div
    class="prose prose-sm prose-numen numen max-w-none break-words"
    :class="{ 'prose--arriving': arriving }"
    @click="pressed"
  >
    <Drawn />
  </div>
</template>

<style scoped>
/* Every size in the scale above prose's own is in `em`. */
.prose {
  font-size: var(--numen-prose-size);
  user-select: text;
  -webkit-user-select: text;
}

/* A link that reaches nothing is drawn as the words it is, under a broken
   line. */
.prose :deep(a[data-reaches='nothing']) {
  color: var(--numen-hushed);
  text-decoration-line: underline;
  text-decoration-style: dashed;
}

/* A table wider than the measure scrolls inside itself, carrying its own
   scrollbar. */
.prose :deep(table) {
  display: block;
  overflow-x: auto;
}
</style>
