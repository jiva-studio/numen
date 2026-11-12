<script setup lang="ts">
import { useTemplateRef } from 'vue'
import { getMarkOf, type RowId } from '../lib/row'
import { useTree } from '../model/useTree'
import { browserClock } from '@/shared/api/clock'
import { DragPreview } from '@/shared/ui/drag-preview'
import TreeRow from './TreeRow.vue'
import type { TreeEvents, TreeProps, TreeSlots } from './Tree.types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<TreeProps>(), {
  open: () => [],
  selected: () => [],
  threshold: 4,
  name: 'Tree',
  counted: (rows: number) => `${rows} rows`,
  marking: undefined,
  clock: () => browserClock,
})

const renamingPath = defineModel<RowId | null>('renamingPath', { default: null })

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<TreeEvents>()

defineSlots<TreeSlots>()

/* --------------------------------- State ---------------------------------- */
const list = useTemplateRef<HTMLElement>('list')

const box = useTemplateRef<HTMLElement>('box')

const { visibleRows, layout, rows, getRowRef, picked, drag, gestures, renaming } = useTree({
  input: props,
  box,
  list,
  renamingPath,
  select: (chosen) => emit('select', chosen),
  emit,
})
const { virtual, virtualRows, rowsStyle } = layout
const { tabbed } = rows
const { into, before, draggedIds, label, at: dropTarget } = drag
const {
  onContextMenu,
  onRowContextMenu,
  onRowFocus,
  onRowPointerDown,
  onRowClick,
  onRowDoubleClick,
  onKeyDown,
} = gestures
const { onRename, onAbandon, onFieldBlur } = renaming
</script>

<template>
  <div
    ref="box"
    class="tree numen bg-surface text-ink min-h-0 font-sans text-base"
    :data-into="dropTarget && 'into' in dropTarget && dropTarget.into === null ? '' : undefined"
    v-bind="getMarkOf(marking, null)"
    @scroll.passive="virtual.onScroll"
    @contextmenu.prevent="onContextMenu"
  >
    <div
      ref="list"
      class="tree__rows"
      role="tree"
      aria-multiselectable="true"
      :aria-label="name"
      :style="rowsStyle"
      @keydown="onKeyDown"
    >
      <TreeRow
        v-for="row in virtualRows"
        :ref="getRowRef(row.id)"
        :key="row.id"
        :row="row"
        :name="name"
        :is-selected="picked.has(row.id)"
        :is-dragged="draggedIds.has(row.id)"
        :is-drop-inside="row.id === into"
        :is-drop-above="row.id === before"
        :is-tab-stop="row.id === tabbed"
        :is-renaming="renamingPath === row.id"
        :mark="getMarkOf(marking, row.id)"
        @focus="onRowFocus(row.id)"
        @pointerdown="onRowPointerDown(row.id, $event)"
        @click="onRowClick(row)"
        @dblclick="onRowDoubleClick(row)"
        @contextmenu.prevent.stop="onRowContextMenu(row, $event)"
        @rename="onRename(row.id, $event)"
        @abandon="onAbandon(row.id)"
        @blur="onFieldBlur"
      >
        <slot name="icon" :id="row.id" :has-children="row.hasChildren" :is-open="row.isOpen" />
      </TreeRow>
    </div>

    <p v-if="!visibleRows.length" class="tree__silence p-inset text-hushed">
      <slot name="silence">Nothing here</slot>
    </p>

    <DragPreview v-if="label" class="tree__dragged" :at="label.at" :label="label.label" />
  </div>
</template>

<style scoped>
.tree {
  --indent: 0.875rem;
  --row: 1.5rem;
  --pad: 0.25rem;
  --gap: 0.25rem;
  --dragged-fade: 0.5;

  display: flex;
  flex-direction: column;
  block-size: 100%;
  overflow: auto;
}

.tree__rows {
  flex: 0 0 auto;
}

.tree[data-into] {
}

.tree__silence {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  margin: 0;
  text-align: center;
}

.tree__dragged {
  z-index: 3;
}
</style>
