<script setup lang="ts">
/**
 * One row of a tree: what is drawn beside the name, the name, and the field a
 * new name is typed in. What the row stands for, and what a press on it means,
 * are the tree's.
 */
import { computed, nextTick, useTemplateRef, watch } from 'vue'
import type { ShownRow } from '../lib/row'
import { TreeField } from './tree-field'

const props = defineProps<{
  row: ShownRow
  /** What the tree is announced as. */
  name: string
  /** The selection stands on it. */
  isSelected: boolean
  /** It is among the rows being dragged. */
  isLifted: boolean
  /** A drop would land inside it. */
  isDropInside: boolean
  /** A drop would land above it. */
  isDropAbove: boolean
  /** The one row the tab key reaches. */
  isTabStop: boolean
  /** Its name is in a field. */
  isRenaming: boolean
  /** What the row is marked with, written onto it as it stands. */
  mark: Record<string, string>
}>()

const emit = defineEmits<{
  /** A name typed and committed. */
  (event: 'rename', name: string): void
  /** The name left as it was, and the keyboard asked back to the row. */
  (event: 'abandon'): void
  /** The field left, the keyboard already somewhere else. */
  (event: 'blur'): void
}>()

defineSlots<{
  /** What is drawn between the row's disclosure and its name. */
  default(): unknown
}>()

/** The field a name is typed in. */
const field = useTemplateRef<InstanceType<typeof TreeField>>('field')

const rowStyle = computed(() => ({ '--level': props.row.level }))

/** The keyboard into the field once it is drawn. */
watch(
  () => props.isRenaming,
  (on) => {
    if (!on) return
    void nextTick(() => field.value?.focus())
  },
)
</script>

<template>
  <div
    class="tree__row flex min-w-0 items-center"
    role="treeitem"
    :aria-level="row.level"
    :aria-expanded="row.hasChildren ? row.open : undefined"
    :aria-selected="isSelected"
    :tabindex="isTabStop ? 0 : -1"
    :data-tree-row="row.id"
    :data-selected="isSelected || undefined"
    :data-dragged="isLifted || undefined"
    :data-last="row.isLast || undefined"
    :data-into="isDropInside || undefined"
    :data-before="isDropAbove || undefined"
    v-bind="mark"
    :style="rowStyle"
  >
    <span class="tree__icon flex shrink-0 items-center">
      <slot />
    </span>

    <TreeField
      v-if="isRenaming"
      ref="field"
      :value="row.name"
      :name="name"
      @rename="emit('rename', $event)"
      @abandon="emit('abandon')"
      @blur="emit('blur')"
    />
    <!-- The whole name is on the element, for one too long to be drawn. -->
    <span v-else class="tree__name min-w-0 truncate" :title="row.name">{{ row.name }}</span>
  </div>
</template>

<style scoped>
.tree__row {
  position: relative;
  block-size: var(--row);
  padding-inline: calc(var(--pad) + var(--indent) * (var(--level) - 1)) var(--pad);
  cursor: default;
  user-select: none;
  touch-action: none;
}

.tree__row:hover {
  background: var(--numen-bubble-bg);
}

.tree__row[data-selected] {
  background: var(--numen-accent);
  color: var(--numen-accent-ink);
}

/* A row on its way somewhere, drawn plainly where it stands. */
.tree__row[data-dragged] {
  opacity: var(--dragged-fade);
}

/* Where the keyboard stands. */
.tree__row:focus-visible {
  outline: none;
}

/* A row the selection already marks is marked once. */
.tree__row[data-selected]:focus-visible {
  outline: none;
}

/* What a drop would land inside: the row. */
.tree__row[data-into] {
}

.tree__row[data-before]::before {
  content: '';
  position: absolute;
  inset-block-start: 0;
  inset-inline: calc(var(--pad) + var(--indent) * (var(--level) - 1)) 0;
  block-size: var(--numen-caret);
  background: var(--numen-ring);
}

/* A name stands clear of the mark beside it. */
.tree__icon {
  margin-inline-end: var(--gap);
}
</style>
