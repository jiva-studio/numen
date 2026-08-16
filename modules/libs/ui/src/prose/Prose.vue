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

const drawn = computed(() => render(props.text))
const Drawn = () => drawn.value
</script>

<template>
  <div
    class="prose prose-sm prose-numen numen max-w-none break-words"
    :class="{ 'prose--arriving': arriving }"
  >
    <Drawn />
  </div>
</template>
