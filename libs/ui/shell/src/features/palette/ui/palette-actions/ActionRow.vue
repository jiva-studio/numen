<script setup lang="ts">
/** One row of the action panel: the action's name, and the key that triggers it. */
import { useTemplateRef } from 'vue'
import { KeyCap } from '@numen/ui'
import type { PlacedAction } from '../../lib/actions'

defineProps<{
  /** The action as it is drawn: its name in runs, and the key it carries. */
  row: PlacedAction
  /** Whether the panel relies on this row. */
  isCurrent: boolean
}>()

/** The row itself, which the panel brings into sight. */
const element = useTemplateRef<HTMLElement>('element')

defineExpose({ element })
</script>

<template>
  <div
    ref="element"
    class="actions__item rounded-node flex items-center gap-3 px-2 py-1.5"
    role="option"
    :aria-selected="isCurrent"
    :data-here="isCurrent || undefined"
  >
    <span class="actions__name min-w-0 flex-1" data-actions="name">
      <span v-for="(part, piece) in row.name" :key="piece" :data-hit="part.isHit || undefined">
        {{ part.text }}
      </span>
    </span>
    <KeyCap v-if="row.key" class="actions__hint" data-actions="hint" :keys="row.key" />
  </div>
</template>

<style scoped>
.actions__item {
  cursor: default;
  user-select: none;
  -webkit-user-select: none;
}

.actions__item[data-here] {
  background: var(--numen-bubble-bg);
}

/* One line, then an ellipsis. A list is read down its leading edge. */
.actions__name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Highlights the matched part of the name. It is drawn under text, so it
   is a tint with the text legible over it. */
.actions__name [data-hit] {
  border-radius: 2px;
  background: var(--numen-highlight);
  font-weight: 600;
}

/* A key written on a row is the last thing on it, and is read after the name. */
.actions__hint {
  flex: none;
}
</style>
