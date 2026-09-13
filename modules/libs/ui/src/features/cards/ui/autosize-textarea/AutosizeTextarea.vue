<script setup lang="ts">
/**
 * A box as tall as what it holds. The ground behind it carries the box's own
 * text in the same type at the same measure, and the cell is sized by the
 * ground, so the box never scrolls.
 *
 * Everything a caller hands it lands on the box, but for a class and a style,
 * which stand on the cell.
 */
import { computed, useAttrs, useTemplateRef } from 'vue'

defineProps<{
  /** What the box holds, as it now reads. */
  text: string
}>()

const emit = defineEmits<{
  /** The text as it now reads, after something was typed into the box. */
  (event: 'write', text: string): void
}>()

defineOptions({ inheritAttrs: false })

const attrs = useAttrs()

/** What the caller handed the box itself. */
const handed = computed(() => {
  const held = { ...attrs }
  delete held['class']
  delete held['style']
  return held
})

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
      class="autosize__box"
      rows="1"
      :value="text"
      v-bind="handed"
      @input="emit('write', ($event.target as HTMLTextAreaElement).value)"
    ></textarea>
  </div>
</template>

<style scoped>
/* A box holding nothing stands as tall as a box holding one line. */
.autosize {
  display: grid;
  min-block-size: calc(
    var(--autosize-lines, 1) * var(--numen-line-height) * 1em + 2 * var(--box-air, 0.5rem)
  );
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
  padding: var(--box-air, 0.5rem) var(--box-pad-inline, var(--numen-box-air));
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
