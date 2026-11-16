<script setup lang="ts">
import type { VNode } from 'vue'
/**
 * One row of the list: the leading element, the label with its hits marked, and the key
 * that selects it, shown at the end.
 *
 * A row of a step that asks for several items has a dot in the colour of its item for its icon, and
 * a check or a dash at its end where it is on.
 *
 * `data-palette` names each part: `icon`, `name`, `detail`, `hint` and `check`.
 */
import { Check, Minus } from '@lucide/vue'
import { KeyCap } from '@numen/ui'
import type { PlacedItem } from '../../../lib/place'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  row: PlacedItem
  id: string
  isCurrent: boolean
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** The pointer crossed the row, and the move that took it there. */
  (event: 'point-at', hasMoved: PointerEvent): void
  /** The row was pressed; `second` is whether the secondary action was requested. */
  (event: 'choose', second: boolean): void
}>()

defineSlots<{
  icon(props: { id: string }): VNode[]
}>()

const CHECKED = { set: 'true', mixed: 'mixed', unset: 'false' } as const
</script>

<template>
  <div
    :id="id"
    class="palette__item rounded-node flex items-center gap-2 px-2 py-1.5"
    role="option"
    :aria-selected="isCurrent"
    :aria-disabled="row.item.isDisabled || undefined"
    :aria-checked="row.item.state ? CHECKED[row.item.state] : undefined"
    :data-here="isCurrent || undefined"
    :data-disabled="row.item.isDisabled || undefined"
    @pointermove="emit('point-at', $event)"
    @pointerdown.prevent
    @click="emit('choose', $event.shiftKey)"
  >
    <span
      v-if="$slots.icon || row.item.state || row.item.color"
      class="palette__icon flex shrink-0 items-center"
      data-palette="icon"
    >
      <slot v-if="$slots.icon" name="icon" :id="row.item.id" />
      <svg v-else-if="row.item.color" class="palette__dot" viewBox="0 0 16 16" aria-hidden="true">
        <circle cx="8" cy="8" r="5" :fill="row.item.color" />
      </svg>
    </span>

    <span class="palette__lines flex min-w-0 flex-1 flex-col">
      <span class="palette__name min-w-0" data-palette="name">
        <span v-for="(part, piece) in row.name" :key="piece" :data-hit="part.isHit || undefined">
          {{ part.text }}
        </span>
      </span>

      <span
        v-if="row.detail.length"
        class="palette__detail text-small text-hushed min-w-0"
        data-palette="detail"
      >
        <span v-for="(part, piece) in row.detail" :key="piece" :data-hit="part.isHit || undefined">
          {{ part.text }}
        </span>
      </span>
    </span>

    <!-- What reaches this item away from the palette. -->
    <KeyCap v-if="row.item.keys" class="palette__hint" data-palette="hint" :keys="row.item.keys" />
    <span
      v-if="row.item.state === 'set' || row.item.state === 'mixed'"
      class="palette__check flex shrink-0 items-center"
      data-palette="check"
      :data-state="row.item.state"
    >
      <Minus v-if="row.item.state === 'mixed'" class="palette__mark" aria-hidden="true" />
      <Check v-else class="palette__mark" aria-hidden="true" />
    </span>
  </div>
</template>

<style scoped>
.palette__item {
  cursor: default;
  user-select: none;
  -webkit-user-select: none;
}

.palette__lines {
  --gap: 0.1rem;

  gap: var(--gap);
}

.palette__icon {
  /* The height of one line of the row's type, which the icon is centered on. */
  --line: calc(var(--numen-font-size) * var(--numen-line-height));

  align-self: start;
  margin-block-start: calc((var(--line) - var(--icon)) / 2);
  inline-size: var(--icon);
  block-size: var(--icon);
}

.palette__item[data-here] {
  background: var(--numen-bubble-bg);
}

.palette__item[data-disabled] {
  color: var(--numen-edge-label);
}

/* One line, then an ellipsis. A list is read down its leading edge. */
.palette__name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Two lines of what is placed under a name. A passage is drawn for the words its
   hit sits among, and one line holds too few of them to read. */
.palette__detail {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  overflow: hidden;
}

/* Marks why the item matched. It overlays text being read, so it is a
   tint and not a colour. */
.palette__name [data-hit],
.palette__detail [data-hit] {
  --corner: 0.125rem;

  border-radius: var(--corner);
  background: var(--numen-highlight);
  font-weight: 600;
}

.palette__dot {
  inline-size: 100%;
  block-size: 100%;
}

.palette__mark {
  inline-size: var(--icon);
  block-size: var(--icon);
}

/* A key written on a row is the last thing on it, and is read after the name. */
.palette__hint {
  flex: none;
}
</style>
