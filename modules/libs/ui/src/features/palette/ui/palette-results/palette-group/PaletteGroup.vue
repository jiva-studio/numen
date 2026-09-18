<script setup lang="ts">
/**
 * One named group of the list: its name, the rows standing under it, and what
 * it says instead where it came back with nothing.
 *
 * A group still filling says so beside its name. `data-palette` names each
 * part: `title` and `silence`.
 */
import { computed, type ComponentPublicInstance } from 'vue'
import { Spinner } from '@/shared/ui/spinner'
import { PaletteRow } from '../palette-row'
import { optionId, type PlacedGroup } from '../../../lib/place'

const props = defineProps<{
  /** The group as it is drawn, its rows numbered over the whole list. */
  placed: PlacedGroup
  /** The number of the row the keyboard stands on, and -1 for none. */
  here: number
  /** What the list and its rows are addressed by. */
  uid: string
}>()

const emit = defineEmits<{
  /** The pointer crossed a row: its number, and the move that took it there. */
  (event: 'point-at', at: number, moved: PointerEvent): void
  /** A row was pressed: its number, and whether the second action was asked for. */
  (event: 'choose', at: number, second: boolean): void
}>()

defineSlots<{
  /** What is drawn before a row's words. */
  icon(props: { id: string }): unknown
}>()

/** The rows as they are drawn, each under the item it stands for. */
const drawn = new Map<string, HTMLElement>()

const hold = (item: string, row: Element | ComponentPublicInstance | null): void => {
  const element = row && '$el' in row ? (row.$el as HTMLElement) : (row as HTMLElement | null)
  if (element) drawn.set(item, element)
  else drawn.delete(item)
}

/** One item brought into sight, where this group is the one holding it. */
const reveal = (item: string): void => {
  drawn.get(item)?.scrollIntoView?.({ block: 'nearest' })
}

defineExpose({ reveal })

/** What this group's name is addressed by, which the list points at. */
const titleId = computed(() => `${props.uid}-group-${props.placed.group.id}`)

const rowId = (at: number): string => optionId(props.uid, at)
</script>

<template>
  <section
    class="palette__group"
    role="group"
    :aria-labelledby="titleId"
    :aria-busy="placed.group.isWorking || undefined"
  >
    <p
      :id="titleId"
      class="palette__title caps-numen text-small text-hushed flex items-center gap-1.5"
      data-palette="title"
    >
      <span>{{ placed.group.title }}</span>
      <!-- More of this group is on its way. -->
      <Spinner v-if="placed.group.isWorking" />
    </p>

    <PaletteRow
      v-for="row in placed.items"
      :ref="(element) => hold(row.item.id, element)"
      :key="row.item.id"
      :row="row"
      :id="rowId(row.at)"
      :is-highlighted="row.at === here"
      @point-at="(moved) => emit('point-at', row.at, moved)"
      @choose="(second) => emit('choose', row.at, second)"
    >
      <template v-if="$slots.icon" #icon="{ id }">
        <slot name="icon" :id="id" />
      </template>
    </PaletteRow>

    <p
      v-if="!placed.items.length && placed.group.silence"
      class="palette__silence text-hushed px-2 py-1.5"
      data-palette="silence"
    >
      {{ placed.group.silence }}
    </p>
  </section>
</template>

<style scoped>
.palette__group + .palette__group {
  margin-block-start: var(--numen-panel-gap);
}

/* The group's name is small print over what it names. */
.palette__title {
  margin: 0;
  padding: 0.15rem 0.5rem;
}

.palette__silence {
  margin: 0;
}
</style>
