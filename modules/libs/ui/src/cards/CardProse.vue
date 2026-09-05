<script setup lang="ts">
/**
 * What a card is written with, read as it is meant to be read.
 *
 * A face is HTML and is drawn as the markup it is, measured first against what
 * a card may be drawn with. Typesetting is the typography plugin's.
 */
import { computed } from 'vue'
import { rendered } from './render'

const props = defineProps<{
  /** HTML, as a person wrote it. */
  text: string
}>()

const emit = defineEmits<{
  /**
   * A link was pressed, with what it points at and the press itself. Nothing
   * here follows it: what a link means is the caller's.
   */
  (event: 'follow', href: string, press: MouseEvent): void
}>()

const html = computed(() => rendered(props.text))

const pressed = (press: MouseEvent) => {
  const link = (press.target as HTMLElement | null)?.closest?.('a')
  const href = link?.getAttribute('href')
  if (href) emit('follow', href, press)
}
</script>

<template>
  <!-- eslint-disable-next-line vue/no-v-html -- what stands here has been measured against what a card may be drawn with -->
  <div
    class="prose prose-sm prose-numen numen max-w-none break-words"
    v-html="html"
    @click="pressed"
  ></div>
</template>

<style scoped>
.prose {
  font-size: var(--numen-prose-size);
  user-select: text;
  -webkit-user-select: text;

  /* Nothing wraps a line a person wrote with no tag around it, so the breaks
     between such lines are kept and each reads as the line it is. */
  white-space: pre-wrap;
}

/* Inside a tag a person wrote, the whitespace is the markup's: a list is not
   indented by the spaces the file was laid out with. Preformatted text keeps
   its own. */
.prose :deep(*:not(pre, pre *)) {
  white-space: normal;
}

/* A table wider than the measure scrolls inside itself, carrying its own
   scrollbar. */
.prose :deep(table) {
  display: block;
  overflow-x: auto;
}

/* A slot naming no field is read where it stands. */
.prose :deep(mark) {
  border-radius: var(--numen-radius);
  padding-inline: 0.25em;
  background: var(--numen-alarm-bg);
  color: var(--numen-alarm);
}
</style>
