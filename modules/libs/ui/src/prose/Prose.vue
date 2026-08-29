<script setup lang="ts">
/**
 * Marked-up prose, read as it is meant to be read.
 *
 * The marks are read by markdown-it and drawn as Vue's own nodes, so what is
 * on the screen stays on the screen while the rest of it arrives. Typesetting
 * is the typography plugin's.
 */
import { computed } from 'vue'
import { render } from './render'

const props = withDefaults(
  defineProps<{
    /** Markdown, whole or as far as it has arrived. */
    text: string
    /** Still being written: a word that has just arrived is shown arriving. */
    arriving?: boolean
  }>(),
  { arriving: false },
)

const emit = defineEmits<{
  /**
   * A link in the prose was pressed, with what it points at and the press
   * itself. Nothing here follows it: what a link means is the caller's, and so
   * is whether the browser should go there.
   */
  (event: 'follow', href: string, press: MouseEvent): void
}>()

const drawn = computed(() => render(props.text))
const Drawn = () => drawn.value

const pressed = (press: MouseEvent) => {
  const link = (press.target as HTMLElement | null)?.closest?.('a')
  const href = link?.getAttribute('href')
  if (href) emit('follow', href, press)
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
/* An answer is set a shade above the size a note is read at, and every size in
   the scale above it is in `em`. */
.prose {
  font-size: calc(var(--numen-reading-size) * 14 / 13);
  user-select: text;
  -webkit-user-select: text;
}

/* A table wider than the measure scrolls inside itself, carrying its own
   scrollbar. */
.prose :deep(table) {
  display: block;
  overflow-x: auto;
}
</style>
