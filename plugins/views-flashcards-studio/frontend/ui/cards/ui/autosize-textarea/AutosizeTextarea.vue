<script setup lang="ts">
/**
 * A box as tall as its content. The cell's `::after` content carries the box's
 * text in the same type at the same width, and the cell is sized by it, so the
 * box never scrolls.
 *
 * Every attribute the caller passes is applied to the box, except `class` and
 * `style`, which are applied to the cell.
 */
import { computed, useAttrs, useId, useTemplateRef } from 'vue'

defineOptions({ inheritAttrs: false })

defineProps<{
  /** The current text of the box. */
  text: string
  /** What a `<label for>` names the box by. Without one the box makes up its own. */
  id?: string
}>()

const emit = defineEmits<{
  /** The current text, emitted after the user types into the box. */
  (event: 'write', text: string): void
}>()

const attrs = useAttrs()

/** What the caller handed the box itself. */
const handed = computed(() => {
  const { class: _class, style: _style, ...rest } = attrs
  return rest
})

const ownId = useId()

const box = useTemplateRef<HTMLTextAreaElement>('box')

defineExpose({
  /** The box itself, for a caller that puts the caret somewhere in what it holds. */
  box,
})
</script>

<template>
  <div class="autosize" :class="attrs.class" :style="attrs.style" :data-autosize="text">
    <textarea
      ref="box"
      :id="id ?? ownId"
      class="autosize__box"
      rows="1"
      :value="text"
      v-bind="handed"
      @input="emit('write', ($event.target as HTMLTextAreaElement).value)"
    ></textarea>
  </div>
</template>

<style scoped>
/* An empty box is as tall as a box holding one line. */
.autosize {
  display: grid;
  min-block-size: calc(var(--autosize-lines) * var(--numen-line-height) * 1em + 2 * var(--box-air));
}

/* The space at the end keeps a line for text ending in a newline, and for no
   text at all. */
.autosize::after {
  content: attr(data-autosize) ' ';
  visibility: hidden;
}

.autosize__box {
  min-inline-size: 0;
}

.autosize__box,
.autosize::after {
  grid-area: 1 / 1;
  padding: var(--box-air) var(--box-pad-inline);
  border: none;
  background: none;
  color: var(--numen-ink);
  font: inherit;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  overflow: hidden;
  resize: none;
}

.autosize__box:focus-visible {
  outline: none;
}
</style>
