<script setup lang="ts">
import { computed, useTemplateRef, type VNode } from 'vue'
import { Cloud } from '@lucide/vue'
import { Spinner } from '@/shared/ui/spinner'
import { useRenameFocus } from '../model/useRenameFocus'
import type { RowId } from '../lib/row'
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
  end(props: { id: RowId }): VNode[]
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
    :data-muted="row.isMuted ? '' : undefined"
    :data-dragged="isDragged || undefined"
    :data-last="row.isLast || undefined"
    :data-into="isDropInside || undefined"
    :data-before="isDropAbove || undefined"
    v-bind="mark"
    :style="rowStyle"
  >
    <span class="tree__icon flex shrink-0 items-center">
      <slot />
      <span
        v-if="row.badge"
        class="tree__badge"
        role="img"
        :aria-label="row.badge.label"
        :data-badge="row.badge.kind"
      >
        <Spinner v-if="row.badge.kind === 'loading'" class="tree__spinner" />
        <Cloud v-else class="tree__cloud" aria-hidden="true" />
      </span>
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
    <span v-else class="tree__name min-w-0 truncate">{{ row.name }}</span>

    <span class="tree__end flex shrink-0 items-center">
      <slot name="end" :id="row.id">
        <component :is="end" v-if="end" :id="row.id" />
      </slot>
    </span>
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
  --tree-row-bg: var(--numen-surface);
  --tree-row-ring: var(--tree-row-bg);
  background: var(--tree-row-bg);
}

.tree__row:hover {
  --tree-row-bg: var(--numen-bubble-bg);
  --tree-row-ring: var(--numen-bubble-bg);
}

.tree__row[data-renaming] {
  --tree-row-bg: var(--numen-bubble-bg);
  --tree-row-ring: var(--numen-bubble-bg);
}

.tree__row[data-selected],
.tree__row[data-selected]:hover {
  --tree-row-bg: var(--numen-accent);
  --tree-row-ring: color-mix(in srgb, var(--numen-accent-ink) 55%, var(--numen-accent));
  color: var(--numen-accent-ink);
}

.tree__row[data-muted]:not([data-selected]) .tree__name {
  color: var(--numen-hushed);
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

.tree__badge {
  --badge: 0.625rem;
  position: absolute;
  inset-inline-end: -0.25rem;
  inset-block-end: -0.25rem;
  display: flex;
  align-items: center;
  justify-content: center;
  inline-size: var(--badge);
  block-size: var(--badge);
  border-radius: var(--numen-radius-pill);
  background: var(--tree-row-bg);
  color: var(--numen-hushed);
}

.tree__row[data-selected] .tree__badge {
  color: var(--numen-accent-ink);
}

.tree__cloud {
  inline-size: var(--badge);
  block-size: var(--badge);
}

.tree__spinner {
  --spinner-size: var(--badge);
  --spinner-thickness: 0.125rem;
}

.tree__icon {
  position: relative;
  margin-inline-end: var(--gap);
}

.tree__end {
  margin-inline-start: auto;
  padding-inline-start: var(--gap);
}
</style>
