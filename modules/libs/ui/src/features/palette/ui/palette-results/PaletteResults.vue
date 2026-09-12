<script setup lang="ts">
/**
 * What the words turned up: rows in named groups, the one the keyboard stands
 * on lit, and the key that reaches a row written at the end of it.
 *
 * A group still filling says so beside its name, and one that came back with
 * nothing says what it has instead. Where the keyboard stands and what a press
 * means are the caller's.
 *
 * `data-palette` names each part: `list`, `title`, `icon`, `name`, `detail`,
 * `hint` and `silence`. A group is drawn as a group and a row as an option.
 */
import { nextTick } from 'vue'
import { Spinner } from '@/shared/ui/spinner'
import { KeyCap } from '@/shared/ui/key-cap'
import { listId, optionId, type PlacedGroup } from '../../lib/place'

const props = defineProps<{
  /** The groups as they are drawn, their rows numbered over the whole list. */
  groups: readonly PlacedGroup[]
  /** The number of the row the keyboard stands on, and -1 for none. */
  here: number
  /**
   * What the list and its rows are addressed by. The field beside the list
   * names the row that is lit, so the two are addressed off one prefix.
   */
  uid: string
  /** What the list is announced as. */
  name: string
}>()

const emit = defineEmits<{
  /** The pointer crossed a row: its number, and the move that took it there. */
  (event: 'over', at: number, moved: PointerEvent): void
  /** A row was pressed: its number, and whether the second action was asked for. */
  (event: 'choose', at: number, second: boolean): void
}>()

defineSlots<{
  /**
   * What is drawn before a row's words. The room for it is kept on every row
   * once the slot is filled, so the words line up down the list whether or not
   * each of them draws anything.
   */
  icon(props: { id: string }): unknown
}>()

/** The rows as they are drawn, each under the item it stands for. */
const drawn = new Map<string, HTMLElement>()

const hold = (item: string, row: unknown): void => {
  if (row) drawn.set(item, row as HTMLElement)
  else drawn.delete(item)
}

/** One item brought into sight, once whatever moved has been drawn. */
const reveal = async (item: string): Promise<void> => {
  await nextTick()
  drawn.get(item)?.scrollIntoView?.({ block: 'nearest' })
}

defineExpose({ reveal })

const rowId = (at: number): string => optionId(props.uid, at)
</script>

<template>
  <div
    :id="listId(uid)"
    class="palette__list min-h-0 flex-1"
    data-palette="list"
    role="listbox"
    :aria-label="name"
  >
    <section
      v-for="one in groups"
      :key="one.group.id"
      class="palette__group"
      role="group"
      :aria-labelledby="`${uid}-group-${one.group.id}`"
      :aria-busy="one.group.working || undefined"
    >
      <p
        :id="`${uid}-group-${one.group.id}`"
        class="palette__title caps-numen flex items-center gap-1.5 text-small text-hushed"
        data-palette="title"
      >
        <span>{{ one.group.title }}</span>
        <!-- More of this group is on its way. -->
        <Spinner v-if="one.group.working" />
      </p>
      <div
        v-for="row in one.items"
        :id="rowId(row.at)"
        :ref="(element) => hold(row.item.id, element)"
        :key="row.item.id"
        class="palette__item flex items-center gap-2 rounded-node px-2 py-1.5"
        role="option"
        :aria-selected="row.at === here"
        :aria-disabled="row.item.disabled || undefined"
        :data-here="row.at === here || undefined"
        :data-disabled="row.item.disabled || undefined"
        @pointermove="emit('over', row.at, $event)"
        @pointerdown.prevent
        @click="emit('choose', row.at, $event.shiftKey)"
      >
        <span
          v-if="$slots.icon"
          class="palette__icon flex shrink-0 items-center"
          data-palette="icon"
        >
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
        <KeyCap
          v-if="row.item.keys"
          class="palette__hint"
          data-palette="hint"
          :keys="row.item.keys"
        />
      </div>

      <p
        v-if="!one.items.length && one.group.silence"
        class="palette__silence px-2 py-1.5 text-hushed"
        data-palette="silence"
      >
        {{ one.group.silence }}
      </p>
    </section>
  </div>
</template>

<style scoped>
.palette__list {
  /* How large an icon is drawn on a row. */
  --icon: 1rem;

  /* The line under the field belongs to what stands beneath it. */
  border-block-start: var(--numen-stroke) solid var(--numen-panel-border);
  max-block-size: var(--tallest);
  padding: var(--numen-field-padding);
  overflow-y: auto;
  overscroll-behavior: contain;
}

.palette__group + .palette__group {
  margin-block-start: var(--numen-panel-gap);
}

/* The group's name is small print over what it names. */
.palette__title {
  margin: 0;
  padding: 0.15rem 0.5rem;
}

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

.palette__silence {
  margin: 0;
}

/* A key written on a row is the last thing on it, and is read after the name. */
.palette__hint {
  flex: none;
}
</style>
