<script setup lang="ts">
/**
 * The palette: a field, and everything the words in it turned up, in groups.
 *
 * It takes groups of items and says which item was chosen and what was asked of
 * it. The keyboard stays in the field the whole time, what is lit is named to a
 * screen reader, and the window behind is out of reach for as long as the
 * palette stands.
 *
 * `data-palette` names each part of the panel: `ground`, `panel`, `crumb`,
 * `field`, `said`, `nothing`, `key` and `more`. The list under the field names
 * its own.
 */
import { computed, useId, useTemplateRef } from 'vue'
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
  actionKey: () => commandKeyChord(navigator.userAgent),
})

const searchQuery = defineModel<string>({ default: '' })

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /**
   * An item was chosen, and what was asked of it. Both identifiers are the
   * caller's, handed back as given.
   */
  (event: 'choose', item: string, action: string): void
  /**
   * The item the keyboard is standing on, said whenever it moves. A list that
   * changes under the keyboard says this once, when it has settled.
   */
  (event: 'light', item: PaletteItemId): void
  /**
   * Backspace was pressed in an empty field. The caller keeps the steps and
   * decides what going back means.
   */
  (event: 'back'): void
  /** It asks to be put away. */
  (event: 'dismiss'): void
}>()

defineSlots<{
  /**
   * What is drawn before a row's words. The room for it is kept on every row
   * once the slot is filled, so the words line up down the list whether or not
   * each of them draws anything.
   */
  icon(props: { id: string }): unknown
  /** What is said while there is no group to draw. */
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

/** The actions a key reaches, which is what the foot of the palette says. */
const hinted = computed(() => getShortcuts(activeItem.value))

const {
  open: isPanelOpen,
  chooseAction,
  onPress,
  onGround,
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
  void results.value?.reveal(selectedId.value)
}
</script>

<template>
  <PaletteFrame
    v-if="isOpen"
    :name="name"
    :to="to"
    @keydown="onKey"
    @ground="onGround"
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
    <span class="sr-only" aria-live="polite" data-palette="said">{{ announcement }}</span>

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
/* The line under the field belongs to what stands beneath it. */
.palette__nothing {
  margin: 0;
  border-block-start: var(--numen-stroke) solid var(--numen-panel-border);
}
</style>
