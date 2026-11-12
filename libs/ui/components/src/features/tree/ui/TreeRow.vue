<script setup lang="ts">
import { computed, useTemplateRef, type VNode } from 'vue'
import { useRenameFocus } from '../model/useRenameFocus'
import type { TreeRowProps } from './TreeRow.types'
import { TreeField } from './tree-field'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<TreeRowProps>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'rename', name: string): void
  (event: 'abandon'): void
  (event: 'blur'): void
}>()

defineSlots<{
  default(): VNode[]
}>()

/* --------------------------------- State ---------------------------------- */
const field = useTemplateRef<InstanceType<typeof TreeField>>('field')

const rowStyle = computed(() => ({ '--level': props.row.level }))

/* --------------------------------- Hooks ---------------------------------- */
useRenameFocus(() => props.isRenaming, field)
</script>

<template>
  <div
    class="tree__row flex min-w-0 items-center"
    role="treeitem"
    :aria-level="row.level"
    :aria-expanded="row.hasChildren ? row.isOpen : undefined"
    :aria-selected="isSelected"
    :tabindex="isTabStop ? 0 : -1"
    :data-tree-row="row.id"
    :data-selected="isSelected && !isRenaming ? '' : undefined"
    :data-renaming="isRenaming ? '' : undefined"
    :data-dragged="isDragged || undefined"
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
      :is-folder="row.hasChildren"
      @rename="emit('rename', $event)"
      @abandon="emit('abandon')"
      @blur="emit('blur')"
    />
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

.tree__row[data-renaming] {
  background: var(--numen-bubble-bg);
}

.tree__row[data-dragged] {
  opacity: var(--dragged-fade);
}

.tree__row:focus-visible {
  outline: none;
}

.tree__row[data-selected]:focus-visible {
  outline: none;
}

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

.tree__icon {
  margin-inline-end: var(--gap);
}
</style>
