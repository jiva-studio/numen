<script setup lang="ts">
/**
 * The command palette: a search field and the grouped results for its text.
 *
 * It takes groups of items and emits the chosen item and the requested action.
 * Keyboard focus stays in the field, the highlighted item is announced to
 * screen readers, and the content behind the palette is inert while it is open.
 *
 * `data-palette` marks each part of the panel by name: the backdrop, `panel`, `crumb`,
 * `field`, `announcement`, `nothing`, `key` and `more`. The list below the field marks
 * its own parts.
 */
import { computed, useId, useTemplateRef } from 'vue'
import { reportFailure, runInBackground } from '@numen/ui'
import { getUserAgent } from '@/shared/lib/dom'
import PaletteKeyHints from './PaletteKeyHints.vue'
import { PaletteActions } from './palette-actions'
import { PaletteField } from './palette-field'
import { PaletteFrame } from './palette-frame'
import { PaletteResults } from './palette-results'
import { ACTION_WORDS } from '../lib/actions'
import { type PaletteItemId } from '../lib/item'
import { commandKeyChord, getShortcuts } from '../lib/keys'
import { usePaletteKeys } from '../model/keys'
import { useActionPanel } from '../model/panel'
import { usePalettePlaces } from '../model/places'
import type { PaletteProps } from './Palette.types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<PaletteProps>(), {
  groups: () => [],
  isOpen: false,
  placeholder: 'Search',
  crumb: '',
  step: '',
  opensOn: '',
  from: null,
  to: 'body',
  name: 'Palette',
  actionWords: () => ACTION_WORDS,
  actionKey: () => commandKeyChord(getUserAgent()),
})

const searchQuery = defineModel<string>({ default: '' })

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /**
   * An item was chosen with the requested action. Both identifiers belong to
   * the caller and are returned unchanged.
   */
  (event: 'choose', item: string, action: string): void
  /**
   * The item highlighted by the keyboard, emitted whenever it changes. When the
   * list changes under the keyboard, it is emitted once after the list settles.
   */
  (event: 'light', item: PaletteItemId): void
  /**
   * Backspace was pressed in an empty field. The caller keeps the steps and
   * decides what going back means.
   */
  (event: 'back'): void
  /** Requests that the palette close. */
  (event: 'dismiss'): void
}>()

defineSlots<{
  /**
   * Rendered before a row's text. Space for it is reserved on every row once
   * the slot is filled, so the text aligns down the list whether or not a row
   * renders anything.
   */
  icon(props: { id: string }): unknown
  /** Rendered when there is no group to show. */
  silence(): unknown
}>()

/* ---------------------------------- State --------------------------------- */
const uid = useId()

const field = useTemplateRef<InstanceType<typeof PaletteField>>('field')

/** The list, for asking it to bring what is lit into sight. */
const results = useTemplateRef<InstanceType<typeof PaletteResults>>('results')

const places = usePalettePlaces({
  groups: () => props.groups,
  typed: searchQuery,
  onItemActive: (item) => emit('light', item),
  tellChoice: (item, action) => emit('choose', item, action),
})
const { placed, announcement, here, activeItem, actions, selectedId, chooseAt } = places

/** The actions available through keys, shown in the palette footer. */
const hinted = computed(() => getShortcuts(activeItem.value))

const {
  open: isPanelOpen,
  chooseAction,
  onPress,
  onBackdrop,
} = useActionPanel({
  getActions: () => actions.value,
  getLit: () => activeItem.value?.id,
  choose: (item, action) => emit('choose', item, action),
  dismiss: () => emit('dismiss'),
})

const { onKey } = usePaletteKeys({
  field,
  reveal,
  places,
  isPanelOpen,
  getActions: () => actions.value,
  typed: searchQuery,
  isOpen: () => props.isOpen,
  getStep: () => props.step,
  getOpensOn: () => props.opensOn,
  getOpenedFrom: () => props.from,
  goBack: () => emit('back'),
  dismiss: () => emit('dismiss'),
})

/* -------------------------------- Handlers -------------------------------- */
// The action panel is about the item that was lit when it opened, and stays
// about it while a pointer crosses the list.
const onOver = (pos: number, event: PointerEvent) => {
  if (!isPanelOpen.value) places.lightAt(pos, event)
}

/* --------------------------------- Helpers -------------------------------- */
/** What is lit is brought into sight. */
function reveal(): void {
  const pending = results.value?.reveal(selectedId.value)
  if (pending) runInBackground(pending, reportFailure)
}
</script>

<template>
  <PaletteFrame
    v-if="isOpen"
    :name="name"
    :to="to"
    @keydown="onKey"
    @ground="onBackdrop"
    @press="onPress"
  >
    <PaletteField
      ref="field"
      v-model="searchQuery"
      :uid="uid"
      :here="isPanelOpen ? -1 : here"
      :is-expanded="placed.length !== 0"
      :placeholder="placeholder"
      :name="name"
      :crumb="crumb"
    />

    <!-- What a search came back with, where it came back with nothing. It
         stands here for as long as the palette does, so what lands in it is
         read out. -->
    <span class="sr-only" aria-live="polite" data-palette="announcement">{{ announcement }}</span>

    <PaletteResults
      v-if="placed.length"
      ref="results"
      :groups="placed"
      :here="here"
      :uid="uid"
      :name="name"
      @point-at="onOver"
      @choose="chooseAt"
    >
      <template v-if="$slots.icon" #icon="{ id }">
        <slot name="icon" :id="id" />
      </template>
    </PaletteResults>

    <p
      v-else-if="$slots.silence"
      class="palette__nothing text-hushed px-2 py-1.5"
      data-palette="nothing"
    >
      <slot name="silence" />
    </p>

    <!-- Everything the lit item offers, by name. It stands over the foot of
         the palette, and the list underneath stays where it was. -->
    <PaletteActions
      v-if="isPanelOpen"
      v-model:open="isPanelOpen"
      :actions="actions"
      :words="actionWords"
      @choose="chooseAction"
    />

    <!-- What the item now lit can be asked. An item offering one action says
         one key, and the last word opens the rest. -->
    <PaletteKeyHints
      v-if="actions.length"
      :hinted="hinted"
      :action-key="actionKey"
      :name="actionWords.name"
    />
  </PaletteFrame>
</template>

<style scoped>
/* The border below the field belongs to the element that follows it. */
.palette__nothing {
  margin: 0;
  border-block-start: var(--numen-stroke) solid var(--numen-panel-border);
}
</style>
