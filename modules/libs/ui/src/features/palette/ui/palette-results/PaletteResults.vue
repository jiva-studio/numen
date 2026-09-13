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
import { nextTick, useTemplateRef } from 'vue'
import { PaletteGroup } from './palette-group'
import { listId, type PlacedGroup } from '../../lib/place'

/** What a drawn group answers to, which is the one item it may hold. */
interface PaletteGroupHandle {
  readonly reveal: (item: string) => void
}

defineProps<{
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
  (event: 'pointAt', at: number, moved: PointerEvent): void
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

/** The groups as they are drawn, each holding the rows it was given. */
const shown = useTemplateRef<PaletteGroupHandle[]>('group')

/** One item brought into sight, once whatever moved has been drawn. */
const reveal = async (item: string): Promise<void> => {
  await nextTick()
  for (const group of shown.value ?? []) group.reveal(item)
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
      @point-at="(at, moved) => emit('pointAt', at, moved)"
      @choose="(at, second) => emit('choose', at, second)"
    >
      <template v-if="$slots.icon" #icon="{ id }">
        <slot name="icon" :id="id" />
      </template>
    </PaletteGroup>
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

</style>
