<script setup lang="ts">
/**
 * What a card is written with, read as it is meant to be read.
 *
 * Marks and tags both come out as elements, and what a card may not be drawn
 * with is gone before this is handed anything. Typesetting is the typography
 * plugin's.
 */
import { computed } from 'vue'
import { drawn } from './render'

const props = defineProps<{
  /** Markdown, with tags among the marks. */
  text: string
}>()

const emit = defineEmits<{
  /**
   * A link was pressed, with what it points at and the press itself. Nothing
   * here follows it: what a link means is the caller's.
   */
  (event: 'follow', href: string, press: MouseEvent): void
}>()

const html = computed(() => drawn(props.text))

const pressed = (press: MouseEvent) => {
  const link = (press.target as HTMLElement | null)?.closest?.('a')
  const href = link?.getAttribute('href')
  if (href) emit('follow', href, press)
}
</script>

<template>
  <!-- eslint-disable-next-line vue/no-v-html -- what stands here has been measured against what a card may be drawn with -->
  <div
    class="marks prose prose-sm prose-numen numen max-w-none break-words"
    v-html="html"
    @click="pressed"
  ></div>
</template>

<style scoped>
/* A card is read at the size an answer is read at. */
.marks {
  font-size: calc(var(--numen-reading-size) * 14 / 13);
  user-select: text;
  -webkit-user-select: text;
}

</style>

<!-- The marks below are made from a person's text, so they carry the scope of
     nothing. They are named by the one class this component draws. -->
<style>
/* A table wider than the measure scrolls inside itself. */
.marks table {
  display: block;
  overflow-x: auto;
}

/* A slot naming no field is read where it stands. */
.marks mark {
  border-radius: var(--numen-radius);
  padding-inline: 0.25em;
  background: var(--numen-alarm-bg);
  color: var(--numen-alarm);
}
</style>
