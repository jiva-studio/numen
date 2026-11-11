<script setup lang="ts">
/**
 * The action panel: every action the highlighted item offers, by name, in its
 * own list with its own field.
 *
 * Keyboard input stays in the panel while it is open; no key event propagates to the palette
 * below it.
 *
 * `data-actions` names each part: `panel`, `list`, `name`, `hint`, `silence`
 * and `search`. An action is an option.
 */
import { useTemplateRef } from 'vue'
import ActionRow from './ActionRow.vue'
import type { ActionWords } from '../../lib/actions'
import type { PaletteAction } from '../../lib/item'
import { useActionPanel } from '../../model/useActionPanel'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  actions: readonly PaletteAction[]
  words: ActionWords
}>()

const open = defineModel<boolean>('open', { default: false })

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** An action was chosen. The identifier is the caller's, returned as given. */
  (event: 'choose', action: string): void
}>()

/* --------------------------------- State ---------------------------------- */

const searchInput = useTemplateRef<HTMLInputElement>('searchInput')

const {
  uid,
  searchQuery,
  actions,
  here,
  onRowPointerMove,
  onChooseAction,
  onKey,
  rows: { getActionName, holdRow },
} = useActionPanel({
  getActions: () => props.actions,
  open,
  searchInput,
  choose: (action) => emit('choose', action),
})
</script>

<template>
  <div
    class="actions panel-numen flex flex-col"
    data-actions="panel"
    role="dialog"
    aria-modal="true"
    :aria-label="words.name"
    @keydown.stop="onKey"
    @pointerdown.stop
  >
    <div
      :id="`${uid}-actions`"
      class="actions__list min-h-0 flex-1"
      data-actions="list"
      role="listbox"
      :aria-label="words.name"
    >
      <ActionRow
        v-for="row in actions"
        :id="getActionName(row.at)"
        :ref="(element) => holdRow(row.action.id, element)"
        :key="row.action.id"
        :row="row"
        :is-current="row.at === here"
        @pointermove="onRowPointerMove(row.at, $event)"
        @pointerdown.prevent
        @click="onChooseAction(row.at)"
      />
    </div>

    <p v-if="!actions.length" class="actions__silence text-hushed" data-actions="silence">
      {{ words.silence }}
    </p>

    <input
      ref="searchInput"
      v-model="searchQuery"
      class="actions__search w-full bg-transparent"
      data-actions="search"
      type="text"
      role="combobox"
      autocomplete="off"
      spellcheck="false"
      :placeholder="words.placeholder"
      :aria-label="words.placeholder"
      :aria-expanded="actions.length !== 0"
      :aria-controls="`${uid}-actions`"
      :aria-activedescendant="here >= 0 ? getActionName(here) : undefined"
    />
  </div>
</template>

<style scoped>
.actions {
  /* How wide the panel is and how much of the palette its list takes. */
  --panel-width: 17.5rem;
  --panel-tallest: 15rem;

  /* A transformed key cap icon in the footer paints in tree order with positioned boxes, so without a z-index it draws over the panel. */
  position: absolute;
  z-index: 1;
  inset-block-end: var(--numen-inset);
  inset-inline-end: var(--numen-inset);
  inline-size: var(--panel-width);
  max-inline-size: calc(100% - 2 * var(--numen-inset));
  max-block-size: calc(100% - 2 * var(--numen-inset));
}

.actions__list {
  max-block-size: var(--panel-tallest);
  padding: var(--numen-field-padding);
  overflow-y: auto;
  overscroll-behavior: contain;
}

/* With no action to list, the empty list would add its padding above the message. */
.actions:has(.actions__silence) .actions__list {
  display: none;
}

.actions__silence {
  display: flex;
  align-items: center;
  min-block-size: var(--numen-field-min);
  margin: 0;
  padding-inline: var(--numen-field-text-inset);
}

.actions__search {
  block-size: var(--numen-field-min);
  padding-inline: var(--numen-field-text-inset);
  border: 0;
  border-block-start: var(--numen-stroke) solid var(--numen-panel-border);
  color: inherit;
  font: inherit;
  outline: none;
}

.actions__search::placeholder {
  color: var(--numen-edge-label);
}
</style>
