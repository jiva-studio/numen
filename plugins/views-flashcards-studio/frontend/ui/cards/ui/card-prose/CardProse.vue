<script setup lang="ts">
/**
 * What a card is written with, read as it is meant to be read.
 *
 * A template is HTML and is drawn as the markup it is, measured first against what
 * a card may be drawn with. Typesetting is the typography plugin's.
 */
import { computed, type Directive } from 'vue'
import { renderCardHtml } from './render'

const props = defineProps<{
  /** HTML, as written by the user. */
  text: string
}>()

const emit = defineEmits<{
  /**
   * A link was pressed, with what it points at and the press itself. Nothing
   * here follows it: what a link means is the caller's.
   */
  (event: 'follow', href: string, press: MouseEvent): void
}>()

/** Writes markup that was checked before it reached this component. */
const vMarkup: Directive<HTMLElement, string> = {
  created: (element, binding) => {
    element.innerHTML = binding.value
  },
  beforeUpdate: (element, binding) => {
    if (binding.value !== binding.oldValue) element.innerHTML = binding.value
  },
}

const html = computed(() => renderCardHtml(props.text))

const onClick = (press: MouseEvent) => {
  const link = (press.target as HTMLElement | null)?.closest?.('a')
  const href = link?.getAttribute('href')
  if (href) emit('follow', href, press)
}
</script>

<template>
  <div
    v-markup="html"
    class="prose prose-sm prose-numen numen max-w-none break-words"
    @click="onClick"
  ></div>
</template>

<style scoped>
.prose {
  font-size: var(--local-prose-size);
  user-select: text;
  -webkit-user-select: text;

  /* A line written with no enclosing tag is not wrapped, so the breaks
     between such lines are kept and each reads as the line it is. */
  white-space: pre-wrap;
}

/* Inside a user-written tag the whitespace belongs to the markup: a list is not
   indented by the spaces the file was laid out with. Preformatted text keeps
   its own. */
.prose :deep(*:not(pre, pre *)) {
  white-space: normal;
}

.prose :deep(table) {
  display: block;
  overflow-x: auto;
}

/* A slot naming no field is read at its current position. */
.prose :deep(mark) {
  border-radius: var(--numen-radius);
  padding-inline: 0.25em;
  background: var(--numen-alarm-bg);
  color: var(--numen-alarm);
}
</style>
