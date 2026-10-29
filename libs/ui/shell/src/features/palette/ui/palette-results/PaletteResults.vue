<script setup lang="ts">
/**
 * The search results: rows in named groups, the row with keyboard focus
 * highlighted, and each row's shortcut key shown at its end.
 *
 * A group that is still loading shows a loading indicator next to its name, and
 * a group with no results shows a placeholder. The caller handles keyboard
 * focus and the meaning of a press.
 *
 * `data-palette` names each part: `list`, `title`, `icon`, `name`, `detail`,
 * `hint` and `silence`. A group is drawn as a group and a row as an option.
 */
import { nextTick, useTemplateRef } from 'vue'
import { PaletteGroup } from './palette-group'
import { listId, type PlacedGroup } from '../../lib/place'

/** The handle of a rendered group, which can reveal one of its items. */
interface PaletteGroupHandle {
  readonly reveal: (item: string) => void
}

defineProps<{
  /** The groups, with their rows numbered across the whole list. */
  groups: readonly PlacedGroup[]
  /** The number of the row with keyboard focus, or -1 for none. */
  here: number
  /**
   * The identifier prefix of the list and its rows. The search field refers to
   * the highlighted row, so both derive their identifiers from one prefix.
   */
  uid: string
  /** The accessible name of the list. */
  name: string
}>()

const emit = defineEmits<{
  /** The pointer moved over a row: its number and the pointer event. */
  (event: 'point-at', pos: number, hasMoved: PointerEvent): void
  /** A row was pressed: its number, and whether the secondary action was requested. */
  (event: 'choose', pos: number, second: boolean): void
}>()

defineSlots<{
  /**
   * The icon before a row's text. When the slot is used, space for it is
   * reserved on every row, so the text aligns down the list whether or not a
   * row renders an icon.
   */
  icon(props: { id: string }): unknown
}>()

/** The rendered groups, each with its rows. */
const groupRefs = useTemplateRef<PaletteGroupHandle[]>('group')

/** Scrolls one item into view after the pending render. */
const reveal = async (item: string): Promise<void> => {
  await nextTick()
  for (const group of groupRefs.value ?? []) group.reveal(item)
}

defineExpose({ reveal })
</script>

<template>
  <div
    :id="listId(uid)"
    class="palette__list min-h-0 flex-1"
    data-palette="list"
    role="listbox"
    :aria-label="name"
  >
    <PaletteGroup
      v-for="one in groups"
      ref="group"
      :key="one.group.id"
      :placed="one"
      :here="here"
      :uid="uid"
      @point-at="(pos, moved) => emit('point-at', pos, moved)"
      @choose="(pos, second) => emit('choose', pos, second)"
    >
      <template v-if="$slots.icon" #icon="{ id }">
        <slot name="icon" :id="id" />
      </template>
    </PaletteGroup>
  </div>
</template>

<style scoped>
.palette__list {
  /* Icon size on a row. */
  --icon: 1rem;

  /* Divider between the search field and the list. */
  border-block-start: var(--numen-stroke) solid var(--numen-panel-border);
  max-block-size: var(--tallest);
  padding: var(--numen-field-padding);
  overflow-y: auto;
  overscroll-behavior: contain;
}
</style>
