<script setup lang="ts">
/**
 * The command palette: a search field and the grouped results for its text.
 *
 * It takes groups of items and emits the chosen item and the requested action.
 * Keyboard focus stays in the field, the highlighted item is announced to
 * screen readers, and the content behind the palette is inert while it is open.
 *
 * With `multi` it asks for several items from one list instead: Enter, a click and the action key emit
 * a toggle for the highlighted row, or a create for the row that makes an item from the typed text, and
 * the palette stays open.
 *
 * A `heading` is the question the field answers: a label above the field, outside the list, that names
 * the field and is read out when it changes.
 *
 * Enter with no row to choose emits `submit` with the text as typed, for a step that is a line of text.
 *
 * `data-palette` marks each part of the panel by name: the backdrop, `panel`, `crumb`,
 * `field`, `announcement`, `nothing`, `key` and `more`. The list below the field marks
 * its own parts.
 */
import { computed, useId, useTemplateRef, type VNode } from 'vue'
import { reportFailure, runInBackground } from '@numen/ui'
import { getUserAgent } from '@/shared/api/dom'
import PaletteKeyHints from './PaletteKeyHints.vue'
import { PaletteActions } from './palette-actions'
import { PaletteField } from './palette-field'
import { PaletteFrame } from './palette-frame'
import { PaletteResults } from './palette-results'
import { ACTION_WORDS } from '../lib/actions'
import { type PaletteItemId } from '../lib/item'
import { commandKeyChord, getShortcuts, isActionsChord } from '../lib/keys'
import { useActiveItem } from '../model/activeItem'
import { usePaletteKeys } from '../model/keys'
import { useMultiPick } from '../model/multiPick'
import { useActionPanelToggle } from '../model/useActionPanelToggle'
import { usePalettePlaces } from '../model/places'
import type { PaletteProps } from './Palette.types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<PaletteProps>(), {
  groups: () => [],
  isOpen: false,
  isBusy: false,
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
  /** In the multiple step, a row was asked to be checked or unchecked. */
  (event: 'toggle', id: string): void
  /** In the multiple step, the row that creates an item was chosen, with the text as typed. */
  (event: 'create', text: string): void
  /** Enter was pressed with no row to choose; the text is as typed. */
  (event: 'submit', text: string): void
  /** Requests that the palette close. */
  (event: 'dismiss'): void
}>()

defineSlots<{
  /**
   * Rendered before a row's text. Space for it is reserved on every row once
   * the slot is filled, so the text aligns down the list whether or not a row
   * renders anything.
   */
  icon(props: { id: string }): VNode[]
  /** Rendered when there is no group to show. */
  silence(): VNode[]
}>()

/* ---------------------------------- State --------------------------------- */
const uid = useId()

const field = useTemplateRef<InstanceType<typeof PaletteField>>('field')

/** The list, for asking it to bring what is lit into sight. */
const results = useTemplateRef<InstanceType<typeof PaletteResults>>('results')

const multiPick = useMultiPick(props, searchQuery, emit)

const places = usePalettePlaces({ getGroups: () => multiPick.groups.value })
const { placed, announcement } = places

const active = useActiveItem({
  places: places.places,
  typed: searchQuery,
  onItemActive: (item) => emit('light', item),
  tellChoice: multiPick.tellChoice,
})
const { here, activeItem, actions, selectedId, chooseAt } = active

/** The actions available through keys, shown in the palette footer. */
const hinted = computed(() => getShortcuts(activeItem.value))

const {
  open: isPanelOpen,
  chooseAction,
  onPress,
  onBackdrop,
} = useActionPanelToggle({
  getActions: () => actions.value,
  getLit: () => activeItem.value?.id,
  choose: (item, action) => emit('choose', item, action),
  dismiss: () => emit('dismiss'),
})

const { onKey } = usePaletteKeys({
  field,
  reveal,
  places,
  active,
  isPanelOpen,
  getActions: () => actions.value,
  typed: searchQuery,
  isOpen: () => props.isOpen,
  getStep: () => props.step,
  getOpensOn: () => props.opensOn,
  getOpenedFrom: () => props.from,
  goBack: () => emit('back'),
  submit: () => emit('submit', searchQuery.value),
  dismiss: () => emit('dismiss'),
})

/* -------------------------------- Handlers -------------------------------- */
const onActionKey = (event: KeyboardEvent): void => {
  if (props.multi && isActionsChord(event)) {
    event.preventDefault()
    active.chooseAt(here.value, false)
  } else onKey(event)
}

const onOver = (pos: number, event: PointerEvent) => {
  if (!isPanelOpen.value) active.highlightAt(pos, event)
}

/* --------------------------------- Helpers -------------------------------- */
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
    @keydown="onActionKey"
    @ground="onBackdrop"
    @press="onPress"
  >
    <p
      v-if="heading"
      :id="`${uid}-heading`"
      class="palette__heading caps-numen text-small text-hushed"
      data-palette="heading"
    >
      {{ heading }}
    </p>

    <PaletteField
      ref="field"
      v-model="searchQuery"
      :uid="uid"
      :here="isPanelOpen ? -1 : here"
      :is-expanded="placed.length !== 0"
      :is-busy="isBusy"
      :placeholder="multi?.placeholder ?? placeholder"
      :name="name"
      :crumb="crumb"
      :heading-id="heading ? `${uid}-heading` : undefined"
    />

    <!-- What a search returned, where it returned nothing. It
         stands here for as long as the palette does, so what lands in it is
         read out. -->
    <span class="sr-only" aria-live="polite" data-palette="announcement">
      {{ [heading, announcement].filter(Boolean).join('. ') }}
    </span>

    <PaletteResults
      v-if="placed.length"
      ref="results"
      :groups="placed"
      :here="here"
      :uid="uid"
      :name="name"
      :is-multiple="multi !== undefined"
      @point-at="onOver"
      @choose="chooseAt"
    >
      <template v-if="$slots.icon && !multi" #icon="{ id }">
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

    <!-- Everything the lit item offers, by name. It is over the foot of
         the palette, and the list underneath stays where it was. -->
    <PaletteActions
      v-if="isPanelOpen"
      v-model:open="isPanelOpen"
      :actions="actions"
      :words="actionWords"
      @choose="chooseAction"
    />

    <!-- The actions of the highlighted item. An item with one action shows
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
/* The question is small print over the field, as a group's name is over its rows. */
.palette__heading {
  margin: 0;
  padding: 0.15rem 0.5rem;
}

/* The border below the field belongs to the element that follows it. */
.palette__nothing {
  margin: 0;
  border-block-start: var(--numen-stroke) solid var(--numen-panel-border);
}
</style>
