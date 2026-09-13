<script setup lang="ts">
/**
 * One row of the list: what is drawn before its words, the words themselves
 * with their hits marked, and the key that reaches it written at the end.
 *
 * `data-palette` names each part: `icon`, `name`, `detail` and `hint`.
 */
import { KeyCap } from '@/shared/ui/key-cap'
import type { PlacedItem } from '../../../lib/place'

defineProps<{
  /** The row as it is drawn, its lines already split. */
  row: PlacedItem
  /** What this row is addressed by. */
  id: string
  /** Whether the keyboard stands on it. */
  here: boolean
}>()

const emit = defineEmits<{
  /** The pointer crossed the row, and the move that took it there. */
  (event: 'pointAt', moved: PointerEvent): void
  /** The row was pressed, and whether the second action was asked for. */
  (event: 'choose', second: boolean): void
}>()

defineSlots<{
  icon(props: { id: string }): unknown
}>()
</script>

<template>
  <div
    :id="id"
    class="palette__item flex items-center gap-2 rounded-node px-2 py-1.5"
    role="option"
    :aria-selected="here"
    :aria-disabled="row.item.disabled || undefined"
    :data-here="here || undefined"
    :data-disabled="row.item.disabled || undefined"
    @pointermove="emit('pointAt', $event)"
    @pointerdown.prevent
    @click="emit('choose', $event.shiftKey)"
  >
    <span v-if="$slots.icon" class="palette__icon flex shrink-0 items-center" data-palette="icon">
      <slot name="icon" :id="row.item.id" />
    </span>

    <span class="palette__lines flex min-w-0 flex-1 flex-col">
      <span class="palette__name min-w-0" data-palette="name">
        <span
          v-for="(part, piece) in row.name"
          :key="piece"
          :data-hit="part.hit || undefined"
          >{{ part.text }}</span
        >
      </span>

      <span
        v-if="row.detail.length"
        class="palette__detail min-w-0 text-small text-hushed"
        data-palette="detail"
      >
        <span
          v-for="(part, piece) in row.detail"
          :key="piece"
          :data-hit="part.hit || undefined"
          >{{ part.text }}</span
        >
      </span>
    </span>

    <!-- What reaches this item away from the palette. -->
    <KeyCap v-if="row.item.keys" class="palette__hint" data-palette="hint" :keys="row.item.keys" />
  </div>
</template>

<style scoped>
.palette__item {
  cursor: default;
  user-select: none;
  -webkit-user-select: none;
}

.palette__lines {
  gap: 0.1rem;
}

/* The room an icon takes, kept whether or not the row draws one, so the words
   line up down the list. What is drawn in it is the caller's.

   It stands on the name, centred against that one line, so the icons read down
   the list beside the names on a row carrying a second line. */
.palette__icon {
  align-self: start;
  margin-block-start: calc((1lh - var(--icon)) / 2);
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

/* Two lines of what stands under a name. A passage is drawn for the words its
   hit sits among, and one line holds too few of them to read. */
.palette__detail {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  overflow: hidden;
}

/* Why the item is here. It sits under words that are being read, so it is a
   tint and not a colour. */
.palette__name [data-hit],
.palette__detail [data-hit] {
  border-radius: 2px;
  background: var(--numen-highlight);
  font-weight: 600;
}

/* A key written on a row is the last thing on it, and is read after the name. */
.palette__hint {
  flex: none;
}
</style>
